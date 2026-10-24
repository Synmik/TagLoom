package utils

import (
	"embed"
	"fmt"
	"strings"
)

// Text documents have no artwork of their own to show, so their thumbnails are
// the bundled format icons — the same pair the design language uses elsewhere.
// They live in the binary rather than on disk: a vault is portable, and a
// thumbnail that depended on a loose PNG would break on another machine.

//go:embed assets/mdicon.png assets/txticon.png
var documentIconFS embed.FS

// DocumentIconPNG returns the bundled icon PNG for a document extension
// (with or without the leading dot), or false when there is none.
func DocumentIconPNG(ext string) ([]byte, bool) {
	name, ok := documentIconName(ext)
	if !ok {
		return nil, false
	}
	data, err := documentIconFS.ReadFile("assets/" + name)
	if err != nil {
		return nil, false
	}
	return data, true
}

// HasDocumentIcon reports whether a format thumbnails from a bundled icon.
func HasDocumentIcon(ext string) bool {
	_, ok := documentIconName(ext)
	return ok
}

func documentIconName(ext string) (string, bool) {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "md":
		return "mdicon.png", true
	case "txt":
		return "txticon.png", true
	}
	return "", false
}

// RenderDocumentIconToWebP writes the bundled icon for ext as a WebP thumbnail
// through FFmpeg, so it matches every other thumbnail in the vault in size and
// quality. The icons carry alpha, which libwebp keeps — the transparent corners
// let the icon sit on the grid background instead of a white box.
func RenderDocumentIconToWebP(ext, outputWebp string, size, quality int) error {
	icon, ok := DocumentIconPNG(ext)
	if !ok {
		return fmt.Errorf("no bundled icon for %s", ext)
	}
	return encodePNGBytesToWebP(icon, outputWebp, size, quality)
}
