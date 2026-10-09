package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"runtime"
	"sync"
	"time"

	gopdfium "github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
	"github.com/tetratelabs/wazero"
)

// PDF thumbnails are rendered by PDFium (Google's renderer, the one Chrome
// uses) compiled to WebAssembly and run by wazero. The dependency chain stays
// permissively licensed — go-pdfium MIT, PDFium Apache-2.0, wazero Apache-2.0
// — and needs no CGO and no shipped DLLs: the ~6 MB WASM module is embedded
// in the Go binary.
const (
	// pdfRenderScale renders the page larger than the thumbnail box and lets
	// FFmpeg downsample with lanczos. PDFium at 1x looks rough on text pages.
	pdfRenderScale = 2
	// pdfMaxRenderDim caps the render size. WASM is 32-bit: one bitmap
	// allocation is bounded (~536 megapixels) and every pixel costs memory in
	// the sandbox. Thumbnails never need more.
	pdfMaxRenderDim = 1024
	// pdfInstanceTimeout bounds how long a worker waits for a PDFium instance.
	pdfInstanceTimeout = 30 * time.Second
)

var (
	pdfPoolMu   sync.Mutex
	pdfPoolInst gopdfium.Pool
)

// pdfPoolSize is the number of live PDFium sandboxes. Each keeps its own
// WebAssembly memory, so the pool stays small even when the thumbnail pool is
// large — extra workers simply queue for an instance.
func pdfPoolSize() int {
	n := runtime.NumCPU() / 2
	if n < 1 {
		n = 1
	}
	if n > 4 {
		n = 4
	}
	return n
}

// pdfiumPool lazily starts the shared PDFium WebAssembly runtime.
func pdfiumPool() (gopdfium.Pool, error) {
	pdfPoolMu.Lock()
	defer pdfPoolMu.Unlock()

	if pdfPoolInst != nil {
		return pdfPoolInst, nil
	}

	size := pdfPoolSize()
	pool, err := webassembly.Init(webassembly.Config{
		MinIdle:  1,
		MaxIdle:  size,
		MaxTotal: size,
		// No mounts: PDFium cannot read the disk, bytes arrive via FileReader.
		FSConfig: wazero.NewFSConfig(),
		// A GUI app has no console — PDFium's chatter goes nowhere.
		Stdout: io.Discard,
		Stderr: io.Discard,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start PDFium runtime: %w", err)
	}

	pdfPoolInst = pool
	return pool, nil
}

// ClosePDFRenderer shuts down the shared runtime. Safe when it was never
// started; a later thumbnail request starts a fresh one.
func ClosePDFRenderer() error {
	pdfPoolMu.Lock()
	defer pdfPoolMu.Unlock()

	if pdfPoolInst == nil {
		return nil
	}
	err := pdfPoolInst.Close()
	pdfPoolInst = nil
	return err
}

// withPDF opens a PDF in a pooled PDFium instance and runs fn while the
// document handle is valid. The file reaches PDFium through an io.ReadSeeker,
// so the sandbox keeps no access to the filesystem.
func withPDF(pdfPath string, fn func(instance gopdfium.Pdfium, doc references.FPDF_DOCUMENT) error) error {
	f, err := os.Open(pdfPath)
	if err != nil {
		return fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat PDF: %w", err)
	}

	pool, err := pdfiumPool()
	if err != nil {
		return err
	}

	instance, err := pool.GetInstance(pdfInstanceTimeout)
	if err != nil {
		return fmt.Errorf("failed to get PDFium instance: %w", err)
	}
	defer func() { _ = instance.Close() }()

	doc, err := instance.OpenDocument(&requests.OpenDocument{
		FileReader:     f,
		FileReaderSize: info.Size(),
	})
	if err != nil {
		return fmt.Errorf("failed to open PDF (encrypted or corrupt?): %w", err)
	}
	defer func() {
		_, _ = instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{Document: doc.Document})
	}()

	return fn(instance, doc.Document)
}

// PDFPageCount returns the page count of a PDF (metadata panel).
func PDFPageCount(pdfPath string) (int, error) {
	count := 0
	err := withPDF(pdfPath, func(instance gopdfium.Pdfium, doc references.FPDF_DOCUMENT) error {
		res, err := instance.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc})
		if err != nil {
			return fmt.Errorf("failed to read page count: %w", err)
		}
		count = res.PageCount
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// renderPDFPNG renders the first page of a PDF at most maxDim pixels wide/tall
// and encodes it to PNG bytes. Encoding happens inside the callback because in
// WASM mode the pixels are a view into sandbox memory that the bitmap's
// CleanupFunc releases.
func renderPDFPNG(pdfPath string, maxDim int) ([]byte, error) {
	maxDim = clampRenderDim(maxDim)

	var out []byte
	err := withPDF(pdfPath, func(instance gopdfium.Pdfium, doc references.FPDF_DOCUMENT) error {
		pageCount, err := instance.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc})
		if err != nil {
			return fmt.Errorf("failed to read page count: %w", err)
		}
		if pageCount.PageCount < 1 {
			return fmt.Errorf("PDF has no pages")
		}

		page, err := instance.RenderPageInPixels(&requests.RenderPageInPixels{
			Page: requests.Page{
				ByIndex: &requests.PageByIndex{Document: doc, Index: 0},
			},
			Width:  maxDim,
			Height: maxDim,
		})
		if err != nil {
			return fmt.Errorf("failed to render PDF page: %w", err)
		}
		if page.CleanupFunc != nil {
			defer page.CleanupFunc()
		}
		if page.Result.RenderedImage == nil {
			return fmt.Errorf("PDFium returned an empty page image")
		}

		// PDFium renders with a transparent page background; viewers paint it
		// white. WebP keeps alpha, and FFmpeg would drop it onto black.
		if rgba, ok := page.Result.RenderedImage.(*image.RGBA); ok {
			compositeOnWhite(rgba)
		}

		var buf bytes.Buffer
		if err := png.Encode(&buf, page.Result.RenderedImage); err != nil {
			return fmt.Errorf("failed to encode rendered page: %w", err)
		}
		out = buf.Bytes()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// compositeOnWhite flattens a non-premultiplied RGBA bitmap onto white using
// its alpha channel, in place. go-pdfium writes pixels BGRA-reversed into RGBA
// order, so they are already in the order this reads them.
func compositeOnWhite(img *image.RGBA) {
	// Weighted average of a channel against white. Two bytes weighted by
	// alpha never exceed 255, so the narrowing conversion is safe.
	blend := func(v, a uint8) uint8 {
		return uint8((uint32(v)*uint32(a) + 255*uint32(255-a)) / 255) // #nosec G115 -- quotient bounded by 255
	}

	for i := 0; i+3 < len(img.Pix); i += 4 {
		a := img.Pix[i+3]
		switch a {
		case 255:
			continue
		case 0:
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = 255, 255, 255, 255
		default:
			img.Pix[i+0] = blend(img.Pix[i+0], a)
			img.Pix[i+1] = blend(img.Pix[i+1], a)
			img.Pix[i+2] = blend(img.Pix[i+2], a)
			img.Pix[i+3] = 255
		}
	}
}

// clampRenderDim keeps a render inside the WASM memory ceiling.
func clampRenderDim(dim int) int {
	if dim > pdfMaxRenderDim {
		return pdfMaxRenderDim
	}
	if dim < 1 {
		return 1
	}
	return dim
}

// RenderPDFToWebP rasterizes the first page of a PDF into a WebP thumbnail:
// PDFium renders, FFmpeg resizes and encodes (the same path SVG takes).
func RenderPDFToWebP(pdfPath, outputWebp string, size, quality int) error {
	pngBytes, err := renderPDFPNG(pdfPath, size*pdfRenderScale)
	if err != nil {
		return err
	}
	return encodePNGBytesToWebP(pngBytes, outputWebp, size, quality)
}

// encodePNGBytesToWebP hands in-memory PNG bytes to FFmpeg through a temp file.
func encodePNGBytesToWebP(pngBytes []byte, outputWebp string, size, quality int) error {
	tmpFile, err := os.CreateTemp("", "img-thumb-*.png")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(pngBytes); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write image bytes: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to write image bytes: %w", err)
	}

	return EncodeImageToWebP(tmpPath, outputWebp, size, quality)
}
