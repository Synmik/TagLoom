package utils

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/webp"
)

// buildPDF builds a minimal one-page PDF with a correct xref table so PDFium
// accepts it. A non-empty content stream is painted on the page; an empty one
// leaves the page untouched, which PDFium renders fully transparent.
func buildPDF(t *testing.T, width, height int, content string) []byte {
	t.Helper()

	pageObj := fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %d %d] >>", width, height)
	bodies := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		pageObj,
	}
	if content != "" {
		bodies[2] = fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %d %d] /Contents 4 0 R >>", width, height)
		bodies = append(bodies, fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(content), content))
	}

	return buildPDFFromBodies(t, bodies)
}

// buildPDFFromBodies wraps object bodies 1..n into a PDF with a correct xref
// table, so PDFium accepts files with any number of pages.
func buildPDFFromBodies(t *testing.T, bodies []string) []byte {
	t.Helper()

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	offsets := make([]int, len(bodies)+1)
	for i, body := range bodies {
		offsets[i+1] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%sendobj\n", i+1, body)
	}

	xrefPos := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(bodies)+1)
	for i := 1; i <= len(bodies); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(bodies)+1, xrefPos)

	return buf.Bytes()
}

func makeTestPDF(t *testing.T, width, height int) []byte {
	t.Helper()
	return buildPDF(t, width, height, fmt.Sprintf("1 0 0 rg\n0 0 %d %d re\nf\n", width, height))
}

func makeBlankTestPDF(t *testing.T, width, height int) []byte {
	t.Helper()
	return buildPDF(t, width, height, "")
}

func writeTempFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestRenderPDFPNG_FirstPageIsRendered(t *testing.T) {
	pdfPath := writeTempFile(t, "red.pdf", makeTestPDF(t, 200, 300))

	pngBytes, err := renderPDFPNG(pdfPath, 64)
	if err != nil {
		t.Fatalf("renderPDFPNG: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("decode rendered PNG: %v", err)
	}

	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		t.Fatalf("empty render: %dx%d", b.Dx(), b.Dy())
	}
	if b.Dx() > 64 || b.Dy() > 64 {
		t.Errorf("render exceeds requested size: %dx%d", b.Dx(), b.Dy())
	}
	// 200x300 page must keep its aspect ratio: taller than wide.
	if b.Dy() <= b.Dx() {
		t.Errorf("aspect ratio not preserved: %dx%d for a 200x300 page", b.Dx(), b.Dy())
	}

	// The page is painted solid red — a white pixel would mean PDFium
	// produced an empty bitmap rather than rendering the content stream.
	r, g, bl, _ := img.At(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2).RGBA()
	if r>>8 < 150 || g>>8 > 80 || bl>>8 > 80 {
		t.Errorf("page not painted red: rgba=(%d,%d,%d)", r>>8, g>>8, bl>>8)
	}
}

func TestRenderPDFPNG_BlankPageIsWhite(t *testing.T) {
	// An unpainted page is transparent in PDFium. The thumbnail must be
	// flattened onto white, otherwise FFmpeg drops the alpha onto black and
	// the thumbnail reads as a black card.
	pdfPath := writeTempFile(t, "blank.pdf", makeBlankTestPDF(t, 150, 150))

	raw, err := renderPDFPNG(pdfPath, 32)
	if err != nil {
		t.Fatalf("renderPDFPNG: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("decode rendered PNG: %v", err)
	}

	b := img.Bounds()
	r, g, bl, a := img.At(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2).RGBA()
	if r>>8 < 240 || g>>8 < 240 || bl>>8 < 240 {
		t.Errorf("blank page not flattened to white: rgba=(%d,%d,%d,%d)", r>>8, g>>8, bl>>8, a>>8)
	}
}

func TestRenderPDFPNG_CorruptPDF(t *testing.T) {
	pdfPath := writeTempFile(t, "junk.pdf", []byte("%PDF-1.4 this is not a PDF"))

	if _, err := renderPDFPNG(pdfPath, 64); err == nil {
		t.Error("expected an error for a corrupt PDF")
	}
}

func TestRenderPDFPNG_RestartsAfterClose(t *testing.T) {
	pdfPath := writeTempFile(t, "red.pdf", makeTestPDF(t, 100, 100))

	if _, err := renderPDFPNG(pdfPath, 32); err != nil {
		t.Fatalf("first render: %v", err)
	}
	if err := ClosePDFRenderer(); err != nil {
		t.Fatalf("ClosePDFRenderer: %v", err)
	}
	if err := ClosePDFRenderer(); err != nil {
		t.Fatalf("ClosePDFRenderer (second call): %v", err)
	}
	if _, err := renderPDFPNG(pdfPath, 32); err != nil {
		t.Fatalf("render after runtime close: %v", err)
	}
}

// TestRenderPDFToWebP runs the full chain: PDFium renders, FFmpeg resizes and
// encodes. Skipped when FFmpeg is not on PATH.
func TestRenderPDFToWebP(t *testing.T) {
	if _, err := FindFFmpeg(); err != nil {
		t.Skip("ffmpeg not available")
	}

	pdfPath := writeTempFile(t, "red.pdf", makeTestPDF(t, 200, 300))
	outPath := filepath.Join(t.TempDir(), "thumb.webp")

	if err := RenderPDFToWebP(pdfPath, outPath, 96, 80); err != nil {
		t.Fatalf("RenderPDFToWebP: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read thumbnail: %v", err)
	}
	if len(data) < 16 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		t.Fatalf("output is not a WebP file (%d bytes)", len(data))
	}

	cfg, err := webp.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode WebP: %v", err)
	}
	w, h := cfg.Width, cfg.Height
	if w > 96 || h > 96 || w <= 0 || h <= 0 {
		t.Errorf("thumbnail %dx%d exceeds the 96px box", w, h)
	}
	if h <= w {
		t.Errorf("thumbnail %dx%d lost the page aspect ratio", w, h)
	}
}

// TestPDFPageCount checks the page count read for the metadata panel.
func TestPDFPageCount(t *testing.T) {
	onePage := writeTempFile(t, "one.pdf", makeTestPDF(t, 100, 100))
	if count, err := PDFPageCount(onePage); err != nil || count != 1 {
		t.Errorf("PDFPageCount(one page) = %d, %v; want 1, nil", count, err)
	}

	twoPages := writeTempFile(t, "two.pdf", buildPDFFromBodies(t, []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R 4 0 R] /Count 2 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] >>",
	}))
	if count, err := PDFPageCount(twoPages); err != nil || count != 2 {
		t.Errorf("PDFPageCount(two pages) = %d, %v; want 2, nil", count, err)
	}

	corrupt := writeTempFile(t, "corrupt.pdf", []byte("%PDF-1.4 not really a pdf"))
	if _, err := PDFPageCount(corrupt); err == nil {
		t.Error("PDFPageCount(corrupt) returned no error")
	}
}
