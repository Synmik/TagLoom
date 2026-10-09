package utils

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/webp"
)

// basicfont.Face7x13 advances 7 px per ASCII rune, which makes the wrapping and
// truncation expectations below exact.

func TestWrapText(t *testing.T) {
	face := basicfont.Face7x13

	cases := []struct {
		name     string
		s        string
		maxWidth int
		maxLines int
		want     []string
	}{
		{
			name:     "fits on one line",
			s:        "short title",
			maxWidth: 7 * 20,
			maxLines: 4,
			want:     []string{"short title"},
		},
		{
			name:     "wraps on word boundaries",
			s:        "aaa bb ccc dd",
			maxWidth: 21, // three characters
			maxLines: 4,
			want:     []string{"aaa", "bb", "ccc", "dd"},
		},
		{
			name:     "collapses runs of whitespace",
			s:        "one\t two\nthree",
			maxWidth: 7 * 20,
			maxLines: 4,
			want:     []string{"one two three"},
		},
		{
			name:     "breaks a word wider than the box",
			s:        "aaaaaaaaaaaa",
			maxWidth: 21,
			maxLines: 4,
			want:     []string{"aaa", "aaa", "aaa", "aaa"},
		},
		{
			name:     "marks text that runs past the line budget",
			s:        "aaa bbb ccc ddd",
			maxWidth: 21,
			maxLines: 2,
			want:     []string{"aaa", "bb…"},
		},
		{name: "empty", s: "   ", maxWidth: 21, maxLines: 4, want: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := wrapText(face, tc.s, tc.maxWidth, tc.maxLines)
			if len(got) != len(tc.want) {
				t.Fatalf("wrapText(%q, %d, %d) = %q, want %q", tc.s, tc.maxWidth, tc.maxLines, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("line %d = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestTruncateText(t *testing.T) {
	face := basicfont.Face7x13

	if got := truncateText(face, "abc", 21); got != "abc" {
		t.Errorf("text that fits should be untouched, got %q", got)
	}
	if got := truncateText(face, "abcdefghij", 21); !strings.HasSuffix(got, "…") || textWidth(face, got) > 21 {
		t.Errorf("truncateText = %q, want an ellipsised fit within 21 px", got)
	}
	if got := truncateText(face, "", 21); got != "" {
		t.Errorf("empty input = %q, want empty", got)
	}
}

// TestDocCardFontLoaded makes sure the fallback card finds a real Unicode font
// on Windows rather than the Latin-only built-in face.
func TestDocCardFontLoaded(t *testing.T) {
	docCardFontOnce.Do(loadDocCardFont)
	if runtime.GOOS == "windows" && docCardFont == nil {
		t.Errorf("no system font loaded from WINDIR=%q (%s)", os.Getenv("WINDIR"), strings.Join(docCardFontNames, ", "))
	}
	t.Logf("font loaded: %v", docCardFont != nil)
}

// TestRenderDocumentCardToWebP draws a card through FFmpeg and checks the
// result is a real WebP of the requested size.
func TestRenderDocumentCardToWebP(t *testing.T) {
	if _, err := FindFFmpeg(); err != nil {
		t.Skip("ffmpeg not available")
	}

	outPath := filepath.Join(t.TempDir(), "card.webp")
	card := DocumentCard{
		Title:    "Путь к морю: дневники странствий и очень длинные подзаголовки",
		Subtitle: "Анна Иванова",
		Badge:    "EPUB",
	}
	if err := RenderDocumentCardToWebP(outPath, 128, 80, card); err != nil {
		t.Fatalf("RenderDocumentCardToWebP: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read card: %v", err)
	}
	if len(data) < 16 || !bytes.Equal(data[:4], []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WEBP")) {
		t.Fatalf("output is not WebP (%d bytes)", len(data))
	}

	img, err := webp.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode webp: %v", err)
	}
	b := img.Bounds()
	if b.Dx() > 128 || b.Dy() > 128 {
		t.Errorf("card size %dx%d exceeds the 128 px box", b.Dx(), b.Dy())
	}
	if b.Dx() < 64 || b.Dy() < 64 {
		t.Errorf("card size %dx%d is far below the requested box", b.Dx(), b.Dy())
	}
	if !cardHasInk(img) {
		t.Error("card looks blank — text was not drawn")
	}
}

// cardHasInk reports whether the card contains dark pixels (the drawn text).
func cardHasInk(img image.Image) bool {
	b := img.Bounds()
	dark := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			if r < 0x8000 && g < 0x8000 && bl < 0x8000 {
				dark++
			}
		}
	}
	return dark > 50
}
