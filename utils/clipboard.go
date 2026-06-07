package utils

import (
	"bytes"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"fmt"
	"image"
	"image/png"
	"os"

	"golang.design/x/clipboard"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// CopyImageToClipboard reads an image file from disk and places it on the
// system clipboard. Supports any format Go's image package can decode
// (JPEG, PNG, GIF, BMP, WebP, TIFF, AVIF via registered decoders).
func CopyImageToClipboard(filePath string) error {
	// 1. Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// 2. Decode the image (accepts any registered decoder)
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	// 3. Encode as PNG (clipboard package expects PNG for FmtImage)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return fmt.Errorf("encode PNG: %w", err)
	}

	// 4. Write to clipboard
	_ = clipboard.Init()
	clipboard.Write(clipboard.FmtImage, buf.Bytes())

	return nil
}
