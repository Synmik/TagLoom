package app

import (
	"archive/zip"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"TagLoom/utils"
)

func TestThumbnailUpToDate(t *testing.T) {
	dir := t.TempDir()
	thumb := filepath.Join(dir, "thumb.webp")
	src := filepath.Join(dir, "src.png")
	if err := os.WriteFile(thumb, []byte("thumb"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("src"), 0600); err != nil {
		t.Fatal(err)
	}

	past := time.Now().Add(-2 * time.Hour)
	now := time.Now()
	future := time.Now().Add(time.Hour)

	// Thumbnail newer than source → up to date
	if err := os.Chtimes(src, past, past); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(thumb, now, now); err != nil {
		t.Fatal(err)
	}
	if !thumbnailUpToDate(thumb, src) {
		t.Error("expected up-to-date when source is older than thumbnail")
	}

	// Source newer than thumbnail → stale
	if err := os.Chtimes(src, future, future); err != nil {
		t.Fatal(err)
	}
	if thumbnailUpToDate(thumb, src) {
		t.Error("expected stale when source is newer than thumbnail")
	}

	// Missing thumbnail → not up to date (caller regenerates)
	if thumbnailUpToDate(filepath.Join(dir, "missing.webp"), src) {
		t.Error("expected not up-to-date when thumbnail is missing")
	}

	// Missing source → not up to date (caller regenerates / skips)
	if thumbnailUpToDate(thumb, filepath.Join(dir, "missing.png")) {
		t.Error("expected not up-to-date when source is missing")
	}
}

// writeTestPNG creates a small valid PNG on disk (real image bytes, so
// ffmpeg can encode it to WebP).
func writeTestPNG(t *testing.T, path string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
}

// thumbPathFor returns the thumbnail_path currently stored for a file ID.
func thumbPathFor(t *testing.T, a *App, id int64) string {
	t.Helper()
	var p *string
	if err := a.db.Conn().QueryRow("SELECT thumbnail_path FROM files WHERE id = ?", id).Scan(&p); err != nil {
		t.Fatalf("query thumbnail_path: %v", err)
	}
	if p == nil {
		return ""
	}
	return *p
}

// TestRepairAllThumbnails verifies the vault-wide repair action (fix 2.5):
// files with an empty thumbnail_path get generated, and old-vault rows whose
// thumbnail_path points at a stale path get re-pointed to the canonical
// hashed path (with the WebP actually present on disk).
func TestRepairAllThumbnails(t *testing.T) {
	if !ffmpegAvailable {
		t.Skip("ffmpeg not available")
	}
	a := newTestApp(t)

	// img1.png — fresh file, no thumbnail yet.
	writeTestPNG(t, filepath.Join(a.vaultPath, "img1.png"))
	id1 := seedFile(t, a, "img1.png")

	// img2.png — old-vault row: thumbnail_path set but stale (no WebP there).
	writeTestPNG(t, filepath.Join(a.vaultPath, "img2.png"))
	id2 := seedFile(t, a, "img2.png")
	stalePath := ".tagloom/thumbnails/zz/0000deadbeef.webp"
	if _, err := a.db.Conn().Exec("UPDATE files SET thumbnail_path = ? WHERE id = ?", stalePath, id2); err != nil {
		t.Fatal(err)
	}

	// The normal pool must generate img1 but NOT touch img2 (its path is
	// non-empty, so it is invisible to the plain pool).
	if err := a.GenerateThumbnailsPool(); err != nil {
		t.Fatalf("GenerateThumbnailsPool: %v", err)
	}
	if got := thumbPathFor(t, a, id1); got == "" {
		t.Error("expected img1 thumbnail_path to be set by pool")
	}
	if !fileExists(t, filepath.Join(a.vaultPath, thumbPathFor(t, a, id1))) {
		t.Error("expected img1 WebP to exist on disk")
	}
	if got := thumbPathFor(t, a, id2); got != stalePath {
		t.Fatalf("plain pool must not touch stale rows, got %q", got)
	}

	// Repair fixes the stale row and leaves the good one alone.
	if err := a.RepairAllThumbnails(); err != nil {
		t.Fatalf("RepairAllThumbnails: %v", err)
	}
	wantRel := a.vault().toRelativePath(a.vault().generateThumbnailAbsolutePath("img2.png"))
	if got := thumbPathFor(t, a, id2); got != wantRel {
		t.Errorf("img2 thumbnail_path = %q, want canonical %q", got, wantRel)
	}
	if !fileExists(t, filepath.Join(a.vaultPath, wantRel)) {
		t.Error("expected img2 WebP to exist at canonical path")
	}
	if got := thumbPathFor(t, a, id1); got == "" {
		t.Error("img1 thumbnail_path should still be set")
	}

	// Repair is idempotent — second run must succeed and change nothing.
	before1 := thumbPathFor(t, a, id1)
	before2 := thumbPathFor(t, a, id2)
	if err := a.RepairAllThumbnails(); err != nil {
		t.Fatalf("second RepairAllThumbnails: %v", err)
	}
	if got := thumbPathFor(t, a, id1); got != before1 {
		t.Errorf("img1 path changed on second repair: %q", got)
	}
	if got := thumbPathFor(t, a, id2); got != before2 {
		t.Errorf("img2 path changed on second repair: %q", got)
	}
}

// TestCancelThumbnailGeneration verifies that CancelThumbnailGeneration
// actually stops a running pool: most seeded files must still be left
// without a thumbnail, and the cancel handle must be cleared once the pool
// exits. (Before the cancel func was wired up, the pool ignored cancel and
// generated everything.)
func TestCancelThumbnailGeneration(t *testing.T) {
	if !ffmpegAvailable {
		t.Skip("ffmpeg not available")
	}
	a := newTestApp(t)

	const n = 500
	ids := make([]int64, n)
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("cancel-%03d.png", i)
		writeTestPNG(t, filepath.Join(a.vaultPath, name))
		ids[i] = seedFile(t, a, name)
	}

	done := make(chan error, 1)
	go func() { done <- a.GenerateThumbnailsPool() }()

	// Wait until the pool has registered its cancel handle.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.RLock()
		registered := a.thumbCancel != nil
		a.mu.RUnlock()
		if registered {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	a.CancelThumbnailGeneration()

	if err := <-done; err != nil {
		t.Fatalf("GenerateThumbnailsPool: %v", err)
	}

	a.mu.RLock()
	left := a.thumbCancel
	a.mu.RUnlock()
	if left != nil {
		t.Error("thumbCancel not cleared after pool finished")
	}

	// Cancel had an effect: the majority of rows never got a thumbnail.
	remaining := 0
	for _, id := range ids {
		if thumbPathFor(t, a, id) == "" {
			remaining++
		}
	}
	if remaining < n/2 {
		t.Errorf("cancel had no effect: only %d of %d rows left without thumbnail", remaining, n)
	}
}

func fileExists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	return err == nil
}

// minimalPDF is a one-page PDF (100x140 pt, solid red) with a valid xref
// table, small enough to inline. PDFium renders it; the content stream is a
// filled rectangle so the render is not blank.
const minimalPDF = "%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>endobj\n2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>endobj\n3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 140] /Contents 4 0 R >>endobj\n4 0 obj\n<< /Length 26 >>\nstream\n1 0 0 rg\n0 0 100 140 re\nf\nendstreamendobj\nxref\n0 5\n0000000000 65535 f \n0000000009 00000 n \n0000000057 00000 n \n0000000113 00000 n \n0000000199 00000 n \ntrailer\n<< /Size 5 /Root 1 0 R >>\nstartxref\n273\n%%EOF\n"

// writeTestEPUB writes a two-entry EPUB whose manifest declares a cover image.
func writeTestEPUB(t *testing.T, path string) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	entries := []struct{ name, body string }{
		{"mimetype", "application/epub+zip"},
		{"META-INF/container.xml", `<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`},
		{"OEBPS/content.opf", `<?xml version="1.0"?><package version="3.0"><metadata></metadata><manifest><item id="cover" href="images/cover.png" media-type="image/png" properties="cover-image"/></manifest><spine></spine></package>`},
	}
	for _, e := range entries {
		w, err := zw.Create(e.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}

	// Cover: a real 8x8 PNG so FFmpeg can decode it.
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 200, G: 40, B: 40, A: 255})
		}
	}

	w, err := zw.Create("OEBPS/images/cover.png")
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(w, img); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestScanAndThumbnailDocuments covers PDF/EPUB support end to end: the
// scanner indexes both, GetFiles reports their formats, and the thumbnail pool
// renders a PDF page and an EPUB cover to WebP.
func TestScanAndThumbnailDocuments(t *testing.T) {
	if !ffmpegAvailable {
		t.Skip("ffmpeg not available")
	}
	a := newTestApp(t)

	if err := os.WriteFile(filepath.Join(a.vaultPath, "doc.pdf"), []byte(minimalPDF), 0600); err != nil {
		t.Fatal(err)
	}
	writeTestEPUB(t, filepath.Join(a.vaultPath, "book.epub"))
	writeTestEPUBNoCover(t, filepath.Join(a.vaultPath, "nocover.epub"))
	// A PDF whose page table is nonsense: the renderer fails, the card wins.
	if err := os.WriteFile(filepath.Join(a.vaultPath, "broken.pdf"), []byte("%PDF-1.4 broken beyond repair"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := a.ScanVault(); err != nil {
		t.Fatalf("ScanVault: %v", err)
	}

	// The scanner must index both documents (there is no format column — the
	// format is derived from the extension, as ThumbnailCell's badge shows).
	for _, name := range []string{"doc.pdf", "broken.pdf", "book.epub", "nocover.epub"} {
		var n int
		if err := a.db.Conn().QueryRow("SELECT COUNT(*) FROM files WHERE vault_path = ?", name).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", name, err)
		}
		if n != 1 {
			t.Errorf("%s indexed %d times, want 1", name, n)
		}
	}
	if got := utils.GetFormatName("doc.pdf"); got != "PDF" {
		t.Errorf("GetFormatName(doc.pdf) = %q, want PDF", got)
	}
	if got := utils.GetFormatName("book.epub"); got != "EPUB" {
		t.Errorf("GetFormatName(book.epub) = %q, want EPUB", got)
	}

	if err := a.GenerateThumbnailsPool(); err != nil {
		t.Fatalf("GenerateThumbnailsPool: %v", err)
	}

	for _, name := range []string{"doc.pdf", "broken.pdf", "book.epub", "nocover.epub"} {
		var id int64
		if err := a.db.Conn().QueryRow("SELECT id FROM files WHERE vault_path = ?", name).Scan(&id); err != nil {
			t.Fatalf("id for %s: %v", name, err)
		}
		rel := thumbPathFor(t, a, id)
		if rel == "" {
			t.Errorf("%s: thumbnail_path empty after pool", name)
			continue
		}
		full := filepath.Join(a.vaultPath, rel)
		data, err := os.ReadFile(full)
		if err != nil {
			t.Errorf("%s: thumbnail missing: %v", name, err)
			continue
		}
		if len(data) < 16 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
			t.Errorf("%s: thumbnail is not WebP (%d bytes)", name, len(data))
		}
	}

	// The single-file path (Regenerate thumbnail, on-demand serving) must
	// render documents too: delete the WebP and ask for it directly.
	for _, name := range []string{"doc.pdf", "broken.pdf", "book.epub", "nocover.epub"} {
		var id int64
		if err := a.db.Conn().QueryRow("SELECT id FROM files WHERE vault_path = ?", name).Scan(&id); err != nil {
			t.Fatalf("id for %s: %v", name, err)
		}
		rel := thumbPathFor(t, a, id)
		if rel == "" {
			continue
		}
		full := filepath.Join(a.vaultPath, rel)
		if err := os.Remove(full); err != nil {
			t.Fatalf("remove %s: %v", full, err)
		}
		if _, err := a.GenerateThumbnail(id); err != nil {
			t.Errorf("GenerateThumbnail(%s): %v", name, err)
			continue
		}
		if _, err := os.Stat(full); err != nil {
			t.Errorf("%s: GenerateThumbnail did not recreate %s", name, rel)
		}
	}

	// Document metadata comes from the container itself: page count for the
	// PDF, package title/author for the EPUB.
	type wantMeta struct {
		path     string
		count    int
		title    string
		author   string
		mime     string
		category string
	}
	for _, w := range []wantMeta{
		{path: "doc.pdf", count: 1, mime: "application/pdf", category: "document"},
		{path: "broken.pdf", mime: "application/pdf", category: "document"},
		{
			path:     "nocover.epub",
			title:    "Зимний дневник",
			author:   "Пётр Петров",
			mime:     "application/epub+zip",
			category: "document",
		},
	} {
		var id int64
		if err := a.db.Conn().QueryRow("SELECT id FROM files WHERE vault_path = ?", w.path).Scan(&id); err != nil {
			t.Fatalf("id for %s: %v", w.path, err)
		}
		md, err := a.GetFileMetadata(id)
		if err != nil {
			t.Errorf("GetFileMetadata(%s): %v", w.path, err)
			continue
		}
		if md.MimeType != w.mime {
			t.Errorf("%s mime = %q, want %q", w.path, md.MimeType, w.mime)
		}
		if md.Category != w.category {
			t.Errorf("%s category = %q, want %q", w.path, md.Category, w.category)
		}
		if md.PageCount != w.count {
			t.Errorf("%s page count = %d, want %d", w.path, md.PageCount, w.count)
		}
		if md.Title != w.title {
			t.Errorf("%s title = %q, want %q", w.path, md.Title, w.title)
		}
		if md.Author != w.author {
			t.Errorf("%s author = %q, want %q", w.path, md.Author, w.author)
		}
	}
}

// TestGenerateDocumentThumbnailFailures checks the document paths that cannot
// produce anything: an unsupported extension and an uncreatable output
// directory are reported and leave no partial file. Documents that merely have
// no cover to show fall back to a drawn card instead.
func TestGenerateDocumentThumbnailFailures(t *testing.T) {
	if !ffmpegAvailable {
		t.Skip("ffmpeg not available")
	}
	dir := t.TempDir()

	// Unsupported extension: no renderer and no card.
	if err := generateDocumentThumbnail(filepath.Join(dir, "note.txt"), filepath.Join(dir, "note.webp"), 256, 80); err == nil {
		t.Error("expected an error for an unsupported extension")
	}

	// Corrupt PDF: PDFium cannot open it.
	broken := filepath.Join(dir, "broken.pdf")
	if err := os.WriteFile(broken, []byte("%PDF-1.4 not really a pdf"), 0600); err != nil {
		t.Fatal(err)
	}

	// EPUB with no images at all.
	noCover := filepath.Join(dir, "nocover.epub")
	writeTestEPUBNoCover(t, noCover)

	// Neither has a cover to show, so a card is drawn instead of failing.
	for _, src := range []string{broken, noCover} {
		thumb := filepath.Join(dir, ".tagloom", "thumbnails", filepath.Base(src)+".webp")
		if err := generateDocumentThumbnail(src, thumb, 128, 80); err != nil {
			t.Errorf("generateDocumentThumbnail(%s): %v", filepath.Base(src), err)
			continue
		}
		data, err := os.ReadFile(thumb)
		if err != nil {
			t.Errorf("%s: no fallback card written: %v", filepath.Base(src), err)
			continue
		}
		if len(data) < 16 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
			t.Errorf("%s: fallback card is not WebP (%d bytes)", filepath.Base(src), len(data))
		}
	}

	// Unwritable destination: reported, and nothing partial is left behind.
	blocked := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := generateDocumentThumbnail(broken, filepath.Join(blocked, "thumb.webp"), 128, 80); err == nil {
		t.Error("expected an error when the thumbnail directory cannot be created")
	}
}

// writeTestEPUBNoCover writes an EPUB whose manifest has no image items.
func writeTestEPUBNoCover(t *testing.T, path string) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	entries := []struct{ name, body string }{
		{"mimetype", "application/epub+zip"},
		{"META-INF/container.xml", `<?xml version="1.0"?><container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container"><rootfiles><rootfile full-path="content.opf" media-type="application/oebps-package+xml"/></rootfiles></container>`},
		{"content.opf", `<?xml version="1.0"?><package version="3.0"><metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Зимний дневник</dc:title><dc:creator>Пётр Петров</dc:creator></metadata><manifest><item id="c1" href="c1.xhtml" media-type="application/xhtml+xml"/></manifest><spine><itemref idref="c1"/></spine></package>`},
		{"c1.xhtml", `<html><body><p>text only</p></body></html>`},
	}
	for _, e := range entries {
		w, err := zw.Create(e.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}
