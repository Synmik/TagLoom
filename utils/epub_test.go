package utils

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// writeTestEPUB builds an EPUB-shaped zip on disk and returns its path.
// Entries are written with "mimetype" first, as real EPUBs do.
func writeTestEPUB(t *testing.T, files map[string][]byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "book.epub")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create epub: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	write := func(name string, body []byte) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := w.Write(body); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	write("mimetype", []byte("application/epub+zip"))
	for name, body := range files {
		write(name, body)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return path
}

// epubText assembles a plain-text EPUB body from string parts.
func epubText(parts ...string) map[string][]byte {
	m := make(map[string][]byte, len(parts)/2)
	for i := 0; i+1 < len(parts); i += 2 {
		m[parts[i]] = []byte(parts[i+1])
	}
	return m
}

const epubContainerXML = `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`

// pngBytes returns a tiny valid PNG in a solid colour, so cover payloads are
// distinguishable from one another.
func pngBytes(t *testing.T, r, g, b uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestExtractEPUBCover_EPUB3Property(t *testing.T) {
	cover := pngBytes(t, 200, 10, 10)
	files := epubText(
		"META-INF/container.xml", epubContainerXML,
		"OEBPS/content.opf", `<?xml version="1.0"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf">
  <metadata><dc:title>Test</dc:title></metadata>
  <manifest>
    <item id="cover" href="images/cover.png" media-type="image/png" properties="cover-image"/>
    <item id="c1" href="text/chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine><itemref idref="c1"/></spine>
</package>`,
	)
	files["OEBPS/images/cover.png"] = cover

	data, ext, err := ExtractEPUBCover(writeTestEPUB(t, files))
	if err != nil {
		t.Fatalf("ExtractEPUBCover: %v", err)
	}
	if !bytes.Equal(data, cover) {
		t.Errorf("cover bytes mismatch: got %d bytes, want %d", len(data), len(cover))
	}
	if ext != ".png" {
		t.Errorf("ext = %q, want .png", ext)
	}
}

func TestExtractEPUBCover_EPUB2MetaCover(t *testing.T) {
	cover := pngBytes(t, 10, 200, 10)
	files := epubText(
		"META-INF/container.xml", epubContainerXML,
		"OEBPS/content.opf", `<?xml version="1.0"?>
<package version="2.0">
  <metadata><meta name="cover" content="cover-img"/></metadata>
  <manifest>
    <item id="cover-img" href="img/COVER.PNG" media-type="image/png"/>
    <item id="c1" href="text/chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine><itemref idref="c1"/></spine>
</package>`,
	)
	// Entry name differs in case from the href — must still resolve.
	files["OEBPS/img/cover.png"] = cover

	data, _, err := ExtractEPUBCover(writeTestEPUB(t, files))
	if err != nil {
		t.Fatalf("ExtractEPUBCover: %v", err)
	}
	if !bytes.Equal(data, cover) {
		t.Errorf("cover bytes mismatch (%d vs %d)", len(data), len(cover))
	}
}

func TestExtractEPUBCover_FromSpineCoverPage(t *testing.T) {
	// No declared cover: the first <img> of the title page wins, resolved
	// relative to the document that references it (here: ../images/ and a
	// percent-escaped filename).
	cover := pngBytes(t, 10, 10, 200)
	files := epubText(
		"META-INF/container.xml", epubContainerXML,
		"OEBPS/content.opf", `<?xml version="1.0"?>
<package version="2.0">
  <metadata></metadata>
  <manifest>
    <item id="tp" href="text/titlepage.xhtml" media-type="application/xhtml+xml"/>
    <item id="c1" href="text/chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine><itemref idref="tp"/><itemref idref="c1"/></spine>
</package>`,
		"OEBPS/text/titlepage.xhtml", `<html><body><img src="../images/cover%20one.png" alt="cover"/></body></html>`,
		"OEBPS/text/chapter1.xhtml", `<html><body><p>hi</p></body></html>`,
	)
	files["OEBPS/images/cover one.png"] = cover

	data, ext, err := ExtractEPUBCover(writeTestEPUB(t, files))
	if err != nil {
		t.Fatalf("ExtractEPUBCover: %v", err)
	}
	if !bytes.Equal(data, cover) {
		t.Errorf("cover bytes mismatch (%d vs %d)", len(data), len(cover))
	}
	if ext != ".png" {
		t.Errorf("ext = %q, want .png", ext)
	}
}

func TestExtractEPUBCover_NoImages(t *testing.T) {
	files := epubText(
		"META-INF/container.xml", epubContainerXML,
		"OEBPS/content.opf", `<?xml version="1.0"?>
<package version="2.0">
  <metadata></metadata>
  <manifest><item id="c1" href="text/chapter1.xhtml" media-type="application/xhtml+xml"/></manifest>
  <spine><itemref idref="c1"/></spine>
</package>`,
	)

	if _, _, err := ExtractEPUBCover(writeTestEPUB(t, files)); err == nil {
		t.Error("expected an error for an EPUB without images")
	}
}

func TestExtractEPUBCover_NotAnEPUB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "junk.epub")
	if err := os.WriteFile(path, []byte("not a zip at all"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, _, err := ExtractEPUBCover(path); err == nil {
		t.Error("expected an error for a non-zip .epub")
	}
}

func TestResolveEPUBHref(t *testing.T) {
	cases := []struct{ dir, href, want string }{
		{"OEBPS", "images/cover.png", "OEBPS/images/cover.png"},
		{"OEBPS/text", "../images/cover.png", "OEBPS/images/cover.png"},
		{"OEBPS", "./cover.xhtml#top", "OEBPS/cover.xhtml"},
		{"OEBPS", "images/cover%20one.png", "OEBPS/images/cover one.png"},
		{".", "content.opf", "content.opf"},
	}
	for _, c := range cases {
		if got := resolveEPUBHref(c.dir, c.href); got != c.want {
			t.Errorf("resolveEPUBHref(%q, %q) = %q, want %q", c.dir, c.href, got, c.want)
		}
	}
}

// TestEncodeEPUBCoverToWebP extracts a cover and encodes it through FFmpeg.
// Skipped when FFmpeg is not on PATH.
func TestEncodeEPUBCoverToWebP(t *testing.T) {
	if _, err := FindFFmpeg(); err != nil {
		t.Skip("ffmpeg not available")
	}

	files := epubText(
		"META-INF/container.xml", epubContainerXML,
		"OEBPS/content.opf", `<?xml version="1.0"?>
<package version="3.0">
  <metadata></metadata>
  <manifest>
    <item id="cover" href="images/cover.png" media-type="image/png" properties="cover-image"/>
  </manifest>
  <spine></spine>
</package>`,
	)
	files["OEBPS/images/cover.png"] = pngBytes(t, 20, 200, 20)

	outPath := filepath.Join(t.TempDir(), "thumb.webp")
	if err := EncodeEPUBCoverToWebP(writeTestEPUB(t, files), outPath, 128, 80); err != nil {
		t.Fatalf("EncodeEPUBCoverToWebP: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read thumbnail: %v", err)
	}
	if len(data) < 16 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		t.Fatalf("output is not a WebP file (%d bytes)", len(data))
	}
}

// TestReadEPUBMetadata checks that Dublin Core title/creator come out of the
// OPF even when the book has no cover image at all.
func TestReadEPUBMetadata(t *testing.T) {
	files := epubText(
		"META-INF/container.xml", epubContainerXML,
		"OEBPS/content.opf", `<?xml version="1.0"?>
<package version="3.0" xmlns:dc="http://purl.org/dc/elements/1.1/">
  <metadata>
    <dc:title>  Путь к морю </dc:title>
    <dc:creator>Анна Иванова</dc:creator>
    <dc:creator>Второй автор</dc:creator>
    <meta name="cover" content="nope"/>
  </metadata>
  <manifest>
    <item id="nav" href="text/toc.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="nav"/>
  </spine>
</package>`,
	)

	info, err := ReadEPUB(writeTestEPUB(t, files))
	if err != nil {
		t.Fatalf("ReadEPUB: %v", err)
	}
	if info.Title != "Путь к морю" {
		t.Errorf("Title = %q, want %q", info.Title, "Путь к морю")
	}
	if info.Creator != "Анна Иванова" {
		t.Errorf("Creator = %q, want first creator %q", info.Creator, "Анна Иванова")
	}
	if len(info.Cover) != 0 {
		t.Errorf("Cover should be empty for an imageless EPUB, got %d bytes", len(info.Cover))
	}

	if _, _, err := ExtractEPUBCover(writeTestEPUB(t, files)); err == nil {
		t.Error("ExtractEPUBCover should fail when the EPUB has no images")
	}
}
