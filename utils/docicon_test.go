package utils

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestDocumentIcons(t *testing.T) {
	for _, ext := range []string{".md", ".MD", "txt", ".TXT"} {
		if !HasDocumentIcon(ext) {
			t.Errorf("HasDocumentIcon(%q) = false, want true", ext)
		}
		data, ok := DocumentIconPNG(ext)
		if !ok {
			t.Fatalf("DocumentIconPNG(%q): not found", ext)
		}
		// PNG signature — the icon is embedded, not generated.
		if len(data) < 8 || !bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}) {
			t.Errorf("DocumentIconPNG(%q): %d bytes, not a PNG", ext, len(data))
		}
	}

	for _, ext := range []string{".pdf", ".epub", ".txtx", ""} {
		if HasDocumentIcon(ext) {
			t.Errorf("HasDocumentIcon(%q) = true, want false", ext)
		}
	}

	if err := RenderDocumentIconToWebP(".pdf", filepath.Join(t.TempDir(), "x.webp"), 256, 80); err == nil {
		t.Error("expected an error when there is no icon for the extension")
	}
}
