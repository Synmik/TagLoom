package utils

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// EPUB previews come from the book's own cover image — no HTML engine needed.
// An EPUB is a zip: META-INF/container.xml points at the OPF package document,
// and the OPF manifest names the cover through `properties="cover-image"`
// (EPUB 3) or `<meta name="cover">` (EPUB 2).

// epubMaxCoverBytes bounds what a single cover entry may decompress to, so a
// hostile zip cannot exhaust memory.
const epubMaxCoverBytes = 64 << 20

// epubItem is an entry in the OPF <manifest>.
type epubItem struct {
	ID         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,attr"`
}

// epubPackage is the subset of the OPF package document we need. Dublin Core
// elements are namespaced, but encoding/xml matches struct fields on the local
// name, so plain `title`/`creator` tags pick up `dc:title`/`dc:creator`.
type epubPackage struct {
	Metadata struct {
		Title    string `xml:"title"`
		Creators []struct {
			Value string `xml:",chardata"`
		} `xml:"creator"`
		Meta []struct {
			Name    string `xml:"name,attr"`
			Content string `xml:"content,attr"`
		} `xml:"meta"`
	} `xml:"metadata"`
	Manifest struct {
		Items []epubItem `xml:"item"`
	} `xml:"manifest"`
	Spine struct {
		ItemRefs []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}

// epubContainer maps META-INF/container.xml to the OPF path.
type epubContainer struct {
	Rootfiles struct {
		Rootfiles []struct {
			FullPath string `xml:"full-path,attr"`
		} `xml:"rootfile"`
	} `xml:"rootfiles"`
}

// EPUBInfo is what TagLoom reads out of an EPUB: library metadata plus the
// cover image (nil when the book simply has none).
type EPUBInfo struct {
	Title    string
	Creator  string
	Cover    []byte
	CoverExt string
}

// ReadEPUB opens an EPUB and returns its metadata and cover.
// Cover lookup order:
//  1. manifest item with properties="cover-image" (EPUB 3)
//  2. <meta name="cover" content="id"> (EPUB 2)
//  3. a manifest image named like a cover
//  4. the first <img> in a spine document named like a cover page
//  5. the first image in the manifest
func ReadEPUB(epubPath string) (*EPUBInfo, error) {
	f, err := os.Open(epubPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open EPUB: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat EPUB: %w", err)
	}

	zr, err := zip.NewReader(f, info.Size())
	if err != nil {
		return nil, fmt.Errorf("failed to read EPUB archive: %w", err)
	}

	// Normalized entry index: exact name and lowercase name -> file.
	entries := make(map[string]*zip.File, len(zr.File))
	lower := make(map[string]*zip.File, len(zr.File))
	for _, file := range zr.File {
		name := path.Clean(strings.ReplaceAll(file.Name, "\\", "/"))
		if _, exists := entries[name]; !exists {
			entries[name] = file
		}
		lname := strings.ToLower(name)
		if _, exists := lower[lname]; !exists {
			lower[lname] = file
		}
	}

	opfPath, err := findEPUBPackage(zr, entries, lower)
	if err != nil {
		return nil, err
	}

	opfFile, ok := lookupEPUBEntry(entries, lower, opfPath)
	if !ok {
		return nil, fmt.Errorf("EPUB OPF not found: %s", opfPath)
	}
	opfData, err := readZipEntry(opfFile, 8<<20)
	if err != nil {
		return nil, fmt.Errorf("failed to read OPF: %w", err)
	}

	var pkg epubPackage
	if err := xml.Unmarshal(opfData, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse OPF: %w", err)
	}

	out := &EPUBInfo{Title: strings.TrimSpace(pkg.Metadata.Title)}
	for _, c := range pkg.Metadata.Creators {
		if v := strings.TrimSpace(c.Value); v != "" {
			out.Creator = v
			break
		}
	}

	byID := make(map[string]epubItem, len(pkg.Manifest.Items))
	for _, item := range pkg.Manifest.Items {
		byID[item.ID] = item
	}
	opfDir := path.Dir(opfPath) // "." when the OPF sits at the archive root

	// Resolve a candidate manifest item to an archive entry and read it.
	tryItem := func(item epubItem) ([]byte, string, bool) {
		entry, found := lookupEPUBEntry(entries, lower, resolveEPUBHref(opfDir, item.Href))
		if !found {
			return nil, "", false
		}
		data, readErr := readZipEntry(entry, epubMaxCoverBytes)
		if readErr != nil || len(data) == 0 {
			return nil, "", false
		}
		return data, epubCoverExt(entry.Name, item.MediaType), true
	}

	isImage := func(item epubItem) bool {
		return strings.HasPrefix(strings.ToLower(item.MediaType), "image/")
	}
	hasProperty := func(item epubItem, want string) bool {
		for _, p := range strings.Fields(item.Properties) {
			if strings.EqualFold(p, want) {
				return true
			}
		}
		return false
	}
	looksLikeCover := func(name string) bool {
		base := strings.ToLower(path.Base(name))
		for _, hint := range []string{"cover", "titlepage", "title-page", "folder"} {
			if strings.Contains(base, hint) {
				return true
			}
		}
		return false
	}

	// 1. EPUB 3 cover-image property.
	for _, item := range pkg.Manifest.Items {
		if hasProperty(item, "cover-image") {
			if data, ext, ok := tryItem(item); ok {
				out.Cover, out.CoverExt = data, ext
				return out, nil
			}
		}
	}

	// 2. EPUB 2 <meta name="cover" content="item-id">.
	for _, meta := range pkg.Metadata.Meta {
		if !strings.EqualFold(meta.Name, "cover") || meta.Content == "" {
			continue
		}
		item, exists := byID[meta.Content]
		if !exists {
			continue
		}
		if isImage(item) {
			if data, ext, ok := tryItem(item); ok {
				out.Cover, out.CoverExt = data, ext
				return out, nil
			}
		} else if data, ext, ok := coverFromDocument(entries, lower, opfDir, item); ok {
			out.Cover, out.CoverExt = data, ext
			return out, nil
		}
	}

	// 3. Manifest image whose name suggests a cover.
	for _, item := range pkg.Manifest.Items {
		if isImage(item) && looksLikeCover(item.Href) {
			if data, ext, ok := tryItem(item); ok {
				out.Cover, out.CoverExt = data, ext
				return out, nil
			}
		}
	}

	// 4. Spine document that looks like a cover page — take its first <img>.
	for _, ref := range pkg.Spine.ItemRefs {
		item, exists := byID[ref.IDRef]
		if !exists || isImage(item) || !looksLikeCover(item.Href) {
			continue
		}
		if data, ext, ok := coverFromDocument(entries, lower, opfDir, item); ok {
			out.Cover, out.CoverExt = data, ext
			return out, nil
		}
	}

	// 5. Any image in the manifest.
	for _, item := range pkg.Manifest.Items {
		if isImage(item) {
			if data, ext, ok := tryItem(item); ok {
				out.Cover, out.CoverExt = data, ext
				return out, nil
			}
		}
	}

	return out, nil
}

// ExtractEPUBCover returns just the cover image of an EPUB.
func ExtractEPUBCover(epubPath string) ([]byte, string, error) {
	info, err := ReadEPUB(epubPath)
	if err != nil {
		return nil, "", err
	}
	if len(info.Cover) == 0 {
		return nil, "", fmt.Errorf("no cover image found in EPUB")
	}
	return info.Cover, info.CoverExt, nil
}

// findEPUBPackage locates the OPF: the container.xml pointer, or any .opf
// entry in a non-conforming archive.
func findEPUBPackage(zr *zip.Reader, entries, lower map[string]*zip.File) (string, error) {
	opfPath := ""
	if cf, ok := lookupEPUBEntry(entries, lower, "META-INF/container.xml"); ok {
		data, readErr := readZipEntry(cf, 1<<20)
		if readErr == nil {
			var container epubContainer
			if xmlErr := xml.Unmarshal(data, &container); xmlErr == nil && len(container.Rootfiles.Rootfiles) > 0 {
				opfPath = strings.ReplaceAll(container.Rootfiles.Rootfiles[0].FullPath, "\\", "/")
			}
		}
	}
	if opfPath == "" {
		for _, file := range zr.File {
			name := path.Clean(strings.ReplaceAll(file.Name, "\\", "/"))
			if strings.EqualFold(path.Ext(name), ".opf") {
				opfPath = name
				break
			}
		}
	}
	if opfPath == "" {
		return "", fmt.Errorf("EPUB has no OPF package document")
	}
	return path.Clean(opfPath), nil
}

// coverFromDocument parses an XHTML spine document and returns the first image
// it references.
func coverFromDocument(entries map[string]*zip.File, lower map[string]*zip.File, opfDir string, doc epubItem) ([]byte, string, bool) {
	entry, found := lookupEPUBEntry(entries, lower, resolveEPUBHref(opfDir, doc.Href))
	if !found {
		return nil, "", false
	}
	html, err := readZipEntry(entry, 4<<20)
	if err != nil {
		return nil, "", false
	}

	href := firstImageHref(html)
	if href == "" {
		return nil, "", false
	}

	docDir := path.Dir(resolveEPUBHref(opfDir, doc.Href))
	imgEntry, found := lookupEPUBEntry(entries, lower, resolveEPUBHref(docDir, href))
	if !found {
		return nil, "", false
	}
	data, err := readZipEntry(imgEntry, epubMaxCoverBytes)
	if err != nil || len(data) == 0 {
		return nil, "", false
	}
	return data, epubCoverExt(imgEntry.Name, ""), true
}

// firstImageHref returns the src of the first <img> (or SVG <image>) element.
func firstImageHref(html []byte) string {
	dec := xml.NewDecoder(bytes.NewReader(html))
	dec.Strict = false
	for {
		tok, err := dec.Token()
		if err != nil {
			return ""
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		name := strings.ToLower(start.Name.Local)
		if name != "img" && name != "image" {
			continue
		}
		for _, attr := range start.Attr {
			switch strings.ToLower(attr.Name.Local) {
			case "src", "href":
				if v := strings.TrimSpace(attr.Value); v != "" {
					return v
				}
			}
		}
	}
}

// resolveEPUBHref resolves an OPF/XHTML href against a directory, dropping
// fragments and percent-escapes.
func resolveEPUBHref(dir, href string) string {
	href = strings.ReplaceAll(href, "\\", "/")
	if i := strings.IndexAny(href, "#?"); i >= 0 {
		href = href[:i]
	}
	if decoded, err := url.PathUnescape(href); err == nil {
		href = decoded
	}
	href = strings.TrimPrefix(href, "./")
	if dir == "" || dir == "." || dir == "/" {
		return path.Clean(href)
	}
	return path.Clean(path.Join(dir, href))
}

// lookupEPUBEntry finds an archive entry by exact name, then case-insensitively.
func lookupEPUBEntry(entries, lower map[string]*zip.File, name string) (*zip.File, bool) {
	name = path.Clean(strings.ReplaceAll(name, "\\", "/"))
	if file, ok := entries[name]; ok {
		return file, true
	}
	if file, ok := lower[strings.ToLower(name)]; ok {
		return file, true
	}
	return nil, false
}

// readZipEntry reads an entry, bounded by maxBytes, and errors on overflow so
// a zip bomb cannot slip through.
func readZipEntry(file *zip.File, maxBytes int64) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()

	data, err := io.ReadAll(io.LimitReader(rc, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("entry %s exceeds %d bytes", file.Name, maxBytes)
	}
	return data, nil
}

// epubCoverExt picks an extension for the cover bytes from the entry name,
// falling back to the declared media type.
func epubCoverExt(entryName, mediaType string) string {
	ext := strings.ToLower(filepath.Ext(entryName))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp", ".tif", ".tiff", ".avif", ".jxl":
		return ext
	}
	switch strings.ToLower(mediaType) {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "image/avif":
		return ".avif"
	}
	if ext != "" {
		return ext
	}
	return ".jpg"
}

// EncodeEPUBCoverToWebP writes an EPUB's cover image to a WebP thumbnail.
// The cover is handed to the same encoders other images use: FFmpeg for
// rasters, the Go SVG decoder for vector covers.
func EncodeEPUBCoverToWebP(epubPath, outputWebp string, size, quality int) error {
	info, err := ReadEPUB(epubPath)
	if err != nil {
		return err
	}
	if len(info.Cover) == 0 {
		return fmt.Errorf("no cover image found in EPUB")
	}

	tmpFile, err := os.CreateTemp("", "epub-cover-*"+info.CoverExt)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(info.Cover); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write cover image: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to write cover image: %w", err)
	}

	if info.CoverExt == ".svg" {
		return EncodeSVGToWebP(tmpPath, outputWebp, size, quality)
	}
	return EncodeImageToWebP(tmpPath, outputWebp, size, quality)
}
