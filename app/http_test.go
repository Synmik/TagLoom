package app

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// passthrough returns a middleware chain whose fallback records that the
// next handler was reached.
func middlewareChain(t *testing.T, a *App, hit *bool) http.Handler {
	t.Helper()
	return a.AssetMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*hit = true
		w.WriteHeader(http.StatusTeapot)
	}))
}

func doRequest(t *testing.T, h http.Handler, method, target, rangeHeader string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	if rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAssetMiddlewareOriginalRange(t *testing.T) {
	a := newTestApp(t)
	content := []byte("0123456789")
	if err := os.WriteFile(filepath.Join(a.vaultPath, "clip.mp4"), content, 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	id := seedFile(t, a, "clip.mp4")

	hit := false
	h := middlewareChain(t, a, &hit)

	// Full GET: 200, correct content type, full body
	rec := doRequest(t, h, "GET", fmt.Sprintf("/api/original/%d", id), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "video/mp4" {
		t.Fatalf("Content-Type = %q, want video/mp4", ct)
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "0123456789" {
		t.Fatalf("body = %q, want full file", body)
	}
	if hit {
		t.Fatal("API path passed through to next handler")
	}

	// Explicit range: bytes=2-5 → 206, "2345", Content-Range header
	rec = doRequest(t, h, "GET", fmt.Sprintf("/api/original/%d", id), "bytes=2-5")
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("range status = %d, want 206 (body: %q)", rec.Code, rec.Body.String())
	}
	if cr := rec.Header().Get("Content-Range"); cr != "bytes 2-5/10" {
		t.Fatalf("Content-Range = %q, want bytes 2-5/10", cr)
	}
	if got := rec.Body.String(); got != "2345" {
		t.Fatalf("range body = %q, want 2345", got)
	}

	// Suffix-style range: bytes=7- → 206, "789"
	rec = doRequest(t, h, "GET", fmt.Sprintf("/api/original/%d", id), "bytes=7-")
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("suffix range status = %d, want 206", rec.Code)
	}
	if got := rec.Body.String(); got != "789" {
		t.Fatalf("suffix range body = %q, want 789", got)
	}

	// Unsatisfiable range → 416
	rec = doRequest(t, h, "GET", fmt.Sprintf("/api/original/%d", id), "bytes=100-200")
	if rec.Code != http.StatusRequestedRangeNotSatisfiable {
		t.Fatalf("bad range status = %d, want 416", rec.Code)
	}
}

func TestAssetMiddlewareParseErrors(t *testing.T) {
	a := newTestApp(t)
	hit := false
	h := middlewareChain(t, a, &hit)

	// Missing ID
	rec := doRequest(t, h, "GET", "/api/original/", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing id status = %d, want 400", rec.Code)
	}
	// Garbage ID (fmt.Sscanf silently accepted this before the fix)
	rec = doRequest(t, h, "GET", "/api/original/abc", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("garbage id status = %d, want 400", rec.Code)
	}
	// Negative ID
	rec = doRequest(t, h, "GET", "/api/original/-1", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("negative id status = %d, want 400", rec.Code)
	}
	// Unknown (but valid) ID → 404
	rec = doRequest(t, h, "GET", "/api/original/9999", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown id status = %d, want 404", rec.Code)
	}
	if hit {
		t.Fatal("API path passed through to next handler")
	}

	// Non-API path → passthrough to next handler
	rec = doRequest(t, h, "GET", "/index.html", "")
	if rec.Code != http.StatusTeapot {
		t.Fatalf("passthrough status = %d, want 418", rec.Code)
	}
	if !hit {
		t.Fatal("non-API path did not reach next handler")
	}
}

func TestAssetMiddlewareOriginalMissingOnDisk(t *testing.T) {
	a := newTestApp(t)
	// DB row exists, file on disk does not
	id := seedFile(t, a, "ghost.png")

	hit := false
	h := middlewareChain(t, a, &hit)

	rec := doRequest(t, h, "GET", fmt.Sprintf("/api/original/%d", id), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestAssetMiddlewareUnknownExtensionContentType(t *testing.T) {
	a := newTestApp(t)
	if err := os.WriteFile(filepath.Join(a.vaultPath, "data.bin"), []byte("x"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	id := seedFile(t, a, "data.bin")

	hit := false
	h := middlewareChain(t, a, &hit)

	rec := doRequest(t, h, "GET", fmt.Sprintf("/api/original/%d", id), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Fatalf("Content-Type = %q, want application/octet-stream", ct)
	}
}
