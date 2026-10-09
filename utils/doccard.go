package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// Documents without a readable cover — an EPUB with no cover image, a PDF that
// will not render — still get a thumbnail: a paper-coloured card with the
// format badge, the title and an optional author line. Better a card than a
// hole in the grid.

const (
	// docCardScale supersamples the card so glyph edges survive the FFmpeg
	// downsample; drawing at thumbnail size produces jagged text.
	docCardScale = 2
	// docCardAspectNum/Den give a portrait 3:4 card, like a document page.
	docCardAspectNum = 4
	docCardAspectDen = 3
	// docCardMaxTitleLines bounds the title block.
	docCardMaxTitleLines = 4
)

var (
	docCardPaper    = color.RGBA{0xFA, 0xFA, 0xF9, 0xFF}
	docCardRule     = color.RGBA{0xD9, 0xD7, 0xD3, 0xFF}
	docCardInk      = color.RGBA{0x24, 0x27, 0x2B, 0xFF}
	docCardMuted    = color.RGBA{0x6A, 0x6F, 0x76, 0xFF}
	docCardBadgeInk = color.RGBA{0x8A, 0x3B, 0x33, 0xFF}
)

// DocumentCard is the text drawn on a fallback thumbnail.
type DocumentCard struct {
	Title    string
	Subtitle string
	Badge    string // "PDF", "EPUB"
}

// docCardFont is the parsed system font, or nil when no candidate was found —
// then the built-in 7x13 face is used, which covers Latin only.
var (
	docCardFontOnce sync.Once
	docCardFont     *opentype.Font
)

// docCardFontNames are tried in order from a system font directory. Segoe UI
// ships with Windows and covers Cyrillic, Greek and CJK, so non-Latin titles
// render as text instead of notdef boxes.
var docCardFontNames = []string{
	"segoeui.ttf",
	"seguisb.ttf",
	"tahoma.ttf",
	"arial.ttf",
	"verdana.ttf",
	"calibri.ttf",
	"DejaVuSans.ttf",
	"FreeSans.ttf",
}

func loadDocCardFont() {
	var dirs []string
	if windir := os.Getenv("WINDIR"); windir != "" {
		dirs = append(dirs, filepath.Join(windir, "Fonts"))
	}
	dirs = append(dirs, "/usr/share/fonts/truetype", "/usr/share/fonts/truetype/dejavu", "/Library/Fonts")

	for _, dir := range dirs {
		for _, name := range docCardFontNames {
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				continue
			}
			if parsed, err := opentype.Parse(data); err == nil {
				docCardFont = parsed
				return
			}
		}
	}
}

// docCardFace returns a face at the requested point size. Every call gets its
// own face: a face is not safe for concurrent use, and thumbnails render on
// several goroutines.
func docCardFace(size float64) font.Face {
	docCardFontOnce.Do(loadDocCardFont)

	if docCardFont != nil {
		if face, err := opentype.NewFace(docCardFont, &opentype.FaceOptions{
			Size:    size,
			DPI:     72,
			Hinting: font.HintingNone,
		}); err == nil {
			return face
		}
	}
	return basicfont.Face7x13
}

// RenderDocumentCardToWebP draws a DocumentCard and encodes it as a WebP
// thumbnail through FFmpeg, so fallback cards match real thumbnails in size
// and quality.
func RenderDocumentCardToWebP(outputWebp string, size, quality int, card DocumentCard) error {
	width := clampRenderDim(size * docCardScale)
	height := width * docCardAspectNum / docCardAspectDen

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{docCardPaper}, image.Point{}, draw.Src)
	drawCardFrame(img, width, height)

	padding := width / 10
	avail := width - 2*padding

	badgeFace := docCardFace(float64(height) / 24)
	titleFace := docCardFace(float64(height) / 13)
	subtitleFace := docCardFace(float64(height) / 17)

	if card.Badge != "" {
		drawText(img, badgeFace, docCardBadgeInk, padding, padding+ascent(badgeFace), card.Badge)
	}

	titleLines := wrapText(titleFace, card.Title, avail, docCardMaxTitleLines)
	subtitle := truncateText(subtitleFace, card.Subtitle, avail)

	titleLineH := lineHeight(titleFace)
	subtitleLineH := lineHeight(subtitleFace)
	blockH := titleLineH*len(titleLines) + subtitleLineH/2 + subtitleLineH*boolToInt(subtitle != "")

	top := padding + ascent(badgeFace) + padding/2
	bottom := height - padding
	if bottom-top < blockH {
		top = padding
	}
	y := top + (bottom-top-blockH)/2

	for _, line := range titleLines {
		drawText(img, titleFace, docCardInk, padding, y+ascent(titleFace), line)
		y += titleLineH
	}
	if subtitle != "" {
		y += titleLineH / 2
		drawText(img, subtitleFace, docCardMuted, padding, y+ascent(subtitleFace), subtitle)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return fmt.Errorf("failed to encode fallback card: %w", err)
	}
	return encodePNGBytesToWebP(buf.Bytes(), outputWebp, size, quality)
}

// drawCardFrame draws a hairline border so light cards stay visible on light
// backgrounds.
func drawCardFrame(img *image.RGBA, width, height int) {
	th := width / 160
	if th < 1 {
		th = 1
	}
	src := image.NewUniform(docCardRule)
	for i := 0; i < th; i++ {
		draw.Draw(img, image.Rect(i, i, width-i, i+1), src, image.Point{}, draw.Over)
		draw.Draw(img, image.Rect(i, height-1-i, width-i, height-i), src, image.Point{}, draw.Over)
		draw.Draw(img, image.Rect(i, i, i+1, height-i), src, image.Point{}, draw.Over)
		draw.Draw(img, image.Rect(width-1-i, i, width-i, height-i), src, image.Point{}, draw.Over)
	}
}

// ascent returns the font ascent in pixels.
func ascent(face font.Face) int {
	return int(face.Metrics().Ascent >> 6)
}

// lineHeight is the face's ascent plus descent, scaled to 1.3x.
func lineHeight(face font.Face) int {
	m := face.Metrics()
	return int((m.Ascent+m.Descent)>>6) * 13 / 10
}

// drawText draws one run of text with its baseline at (x, baselineY).
func drawText(dst draw.Image, face font.Face, col color.Color, x, baselineY int, s string) {
	if s == "" {
		return
	}
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(x, baselineY),
	}
	d.DrawString(s)
}

// textWidth measures a string in pixels.
func textWidth(face font.Face, s string) int {
	return int(font.MeasureString(face, s) >> 6)
}

// wrapText splits s into at most maxLines lines no wider than maxWidth. A word
// that alone exceeds the width is broken by rune, and text left over after
// maxLines ends in an ellipsis on the last line.
func wrapText(face font.Face, s string, maxWidth, maxLines int) []string {
	s = strings.Join(strings.FieldsFunc(s, func(r rune) bool {
		return unicode.IsSpace(r)
	}), " ")
	if s == "" {
		return nil
	}

	var lines []string
	rest := s
	for rest != "" && len(lines) < maxLines {
		if textWidth(face, rest) <= maxWidth {
			lines = append(lines, rest)
			rest = ""
			break
		}

		words := strings.Split(rest, " ")
		fit := 0
		for i := 1; i <= len(words); i++ {
			if textWidth(face, strings.Join(words[:i], " ")) <= maxWidth {
				fit = i
			} else {
				break
			}
		}

		if fit == 0 {
			// One word wider than the box: cut it by rune.
			r := []rune(rest)
			n := 1
			for n < len(r) && textWidth(face, string(r[:n+1])) <= maxWidth {
				n++
			}
			lines = append(lines, string(r[:n]))
			rest = strings.Join(strings.Fields(string(r[n:])), " ")
			continue
		}

		lines = append(lines, strings.Join(words[:fit], " "))
		rest = strings.Join(words[fit:], " ")
	}

	if rest != "" && len(lines) > 0 {
		last := len(lines) - 1
		lines[last] = truncateText(face, lines[last]+"…", maxWidth)
	}
	return lines
}

// truncateText shortens s with an ellipsis when it is too wide.
func truncateText(face font.Face, s string, maxWidth int) string {
	if s == "" || textWidth(face, s) <= maxWidth {
		return s
	}
	r := []rune(s)
	for i := len(r); i > 0; i-- {
		candidate := strings.TrimRight(string(r[:i]), " ,.;:-–—") + "…"
		if textWidth(face, candidate) <= maxWidth {
			return candidate
		}
	}
	return "…"
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
