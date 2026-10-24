package utils

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

func writeTempText(t *testing.T, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sample.txt")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestIsTextFile(t *testing.T) {
	cases := map[string]bool{
		"notes.md":       true,
		"notes.TXT":      true,
		"/a/b/README.Md": true,
		"book.epub":      false,
		"doc.pdf":        false,
		"photo.jpg":      false,
		"noextension":    false,
		"archive.md.gz":  false,
		"notes.markdown": false, // .markdown is not one of the two text formats
	}
	for path, want := range cases {
		if got := IsTextFile(path); got != want {
			t.Errorf("IsTextFile(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestReadTextSampleEncodings(t *testing.T) {
	const text = "Привет мир" // Cyrillic, plus an emoji below for UTF-16 surrogates

	utf8File := append([]byte{0xEF, 0xBB, 0xBF}, []byte(text)...)
	// The emoji is astral, so this file holds a surrogate pair — the shape
	// textDecode has to reassemble.
	utf16LE := utf16Bytes(binary.LittleEndian, []byte{0xFF, 0xFE}, text+"🙂")
	utf16BE := utf16Bytes(binary.BigEndian, []byte{0xFE, 0xFF}, "Ж")

	cases := []struct {
		name     string
		content  []byte
		want     string
		encoding string
	}{
		{"plain utf-8", []byte(text), text, EncodingUTF8},
		{"utf-8 with bom", utf8File, text, EncodingUTF8},
		{"utf-16le bom", utf16LE, text + "🙂", EncodingUTF16LE},
		{"utf-16be bom", utf16BE, "Ж", EncodingUTF16BE},
		{"empty", nil, "", EncodingUTF8},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ReadTextSample(writeTempText(t, c.content), TextPreviewLimit)
			if err != nil {
				t.Fatalf("ReadTextSample: %v", err)
			}
			if got.Text != c.want {
				t.Errorf("text = %q, want %q", got.Text, c.want)
			}
			if got.Encoding != c.encoding {
				t.Errorf("encoding = %q, want %q", got.Encoding, c.encoding)
			}
			if got.Truncated {
				t.Errorf("Truncated = true for a %d byte file", len(c.content))
			}
		})
	}
}

// utf16Bytes writes s as UTF-16 code units in the given byte order behind bom.
// Passing a ByteOrder keeps the two orders one line apart instead of a
// hand-rolled byte split.
func utf16Bytes(order binary.ByteOrder, bom []byte, s string) []byte {
	out := append([]byte(nil), bom...)
	var unit [2]byte
	for _, u := range utf16.Encode([]rune(s)) {
		order.PutUint16(unit[:], u)
		out = append(out, unit[:]...)
	}
	return out
}

func TestReadTextSampleTruncates(t *testing.T) {
	path := writeTempText(t, []byte(strings.Repeat("abcd", 1000)))

	got, err := ReadTextSample(path, 16)
	if err != nil {
		t.Fatalf("ReadTextSample: %v", err)
	}
	if len(got.Text) != 16 {
		t.Errorf("text length = %d, want 16: %q", len(got.Text), got.Text)
	}
	if !got.Truncated {
		t.Error("Truncated = false, want true for a file past the limit")
	}
	if got.BytesRead > 16 {
		t.Errorf("BytesRead = %d, want <= 16", got.BytesRead)
	}

	// A file that ends exactly at the limit is not truncated.
	exact := writeTempText(t, []byte("12345678"))
	whole, err := ReadTextSample(exact, 8)
	if err != nil {
		t.Fatalf("ReadTextSample: %v", err)
	}
	if whole.Truncated || whole.Text != "12345678" {
		t.Errorf("exact-size read = %+v, want the whole file and no truncation", whole)
	}
}

func TestReadTextSampleRejectsBinaryAndUTF32(t *testing.T) {
	if _, err := ReadTextSample(writeTempText(t, []byte("a\x00b\x00c\x00")), TextPreviewLimit); err == nil {
		t.Error("expected an error for a NUL-bearing file (UTF-16 without a BOM)")
	}

	utf32LE := []byte{0xFF, 0xFE, 0x00, 0x00, 'a', 0, 0, 0}
	if _, err := ReadTextSample(writeTempText(t, utf32LE), TextPreviewLimit); err == nil {
		t.Error("expected an error for a UTF-32LE BOM")
	}
}

func TestReadTextSampleMissingFile(t *testing.T) {
	if _, err := ReadTextSample(filepath.Join(t.TempDir(), "gone.txt"), TextPreviewLimit); err == nil {
		t.Error("expected an error for a missing file")
	}
}

func TestCountText(t *testing.T) {
	cases := []struct {
		name    string
		content string
		lines   int
		words   int
	}{
		{"empty", "", 0, 0},
		{"one line no newline", "hello there", 1, 2},
		{"one line with newline", "hello there\n", 1, 2},
		{"three lines", "one two\nthree\nfour five six\n", 3, 6},
		{"crlf", "one two\r\nthree\r\n", 2, 3},
		// "a\n\n" is a line and an empty one after it — two lines, as an editor shows
		{"final blank line", "a\n\n", 2, 1},
		{"blank line in the middle", "a\n\nb", 3, 2},
		{"tabs and spaces", "a\tb  c\n\td", 2, 4},
		{"cyrillic words", "Привет\tмир\nкак дела", 2, 4},
		// words are whitespace-separated runs, like wc -w: punctuation does not split them
		{"punctuation does not split words", "one,two;three", 1, 1},
		{"cjk without spaces", "你好世界", 1, 1},
		{"emoji counts as text", "🙂 🙂", 1, 2},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := CountText(writeTempText(t, []byte(c.content)), TextCountLimit)
			if err != nil {
				t.Fatalf("CountText: %v", err)
			}
			if got.Lines != c.lines || got.Words != c.words {
				t.Errorf("lines=%d words=%d, want lines=%d words=%d",
					got.Lines, got.Words, c.lines, c.words)
			}
			if got.Truncated {
				t.Error("Truncated = true for a small file")
			}
		})
	}
}

func TestCountTextUTF16(t *testing.T) {
	body := utf16Bytes(binary.LittleEndian, []byte{0xFF, 0xFE}, "одна\nдва три\n")

	got, err := CountText(writeTempText(t, body), TextCountLimit)
	if err != nil {
		t.Fatalf("CountText: %v", err)
	}
	if got.Lines != 2 || got.Words != 3 {
		t.Errorf("lines=%d words=%d, want 2 and 3", got.Lines, got.Words)
	}
}

func TestCountTextTruncated(t *testing.T) {
	// 4-byte words: a 12-byte limit stops mid-line, so the tallies are floors.
	path := writeTempText(t, []byte("aaaa bbbb cccc dddd eeee\n"))

	got, err := CountText(path, 12)
	if err != nil {
		t.Fatalf("CountText: %v", err)
	}
	if !got.Truncated {
		t.Error("Truncated = false, want true for a file past the count limit")
	}
	if got.Words != 3 {
		t.Errorf("words = %d, want 3 (counted up to the limit)", got.Words)
	}
	// The only newline in the file is past the limit, so no line was seen whole.
	if got.Lines != 0 {
		t.Errorf("lines = %d, want 0 (no newline read)", got.Lines)
	}
}

func TestCountTextRejectsBinary(t *testing.T) {
	if _, err := CountText(writeTempText(t, []byte("a\x00b\x00")), TextCountLimit); err == nil {
		t.Error("expected an error for a NUL-bearing file")
	}
}

func TestTextReaderHandlesShortTail(t *testing.T) {
	// UTF-16LE BOM plus one stray byte: the reader must not spin forever, and
	// the cut code unit becomes one replacement character.
	got, err := ReadTextSample(writeTempText(t, []byte{0xFF, 0xFE, 0x41}), TextPreviewLimit)
	if err != nil {
		t.Fatalf("ReadTextSample: %v", err)
	}
	if got.Text != "\ufffd" {
		t.Errorf("text = %q, want one replacement character", got.Text)
	}
	if got.BytesRead != 1 {
		t.Errorf("bytes read = %d, want 1 (the BOM is not counted)", got.BytesRead)
	}
}
