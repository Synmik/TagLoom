package utils

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// Plain text: .txt and Markdown .md. TagLoom shows them inline, so the text
// has to be read and decoded on this side. Two limits keep a stray 500 MB log
// from stalling the metadata panel or filling the preview pane: a preview
// reads at most TextPreviewLimit bytes, counting reads at most TextCountLimit.

const (
	// TextPreviewLimit bounds what an inline preview takes from the file.
	TextPreviewLimit = 256 << 10 // 256 KiB
	// TextCountLimit bounds what line/word counting reads.
	TextCountLimit = 8 << 20 // 8 MiB
)

// Encoding names reported to the frontend. A missing BOM means UTF-8.
const (
	EncodingUTF8    = "utf-8"
	EncodingUTF16LE = "utf-16le"
	EncodingUTF16BE = "utf-16be"
)

// IsTextFile reports whether the file is a text format TagLoom reads inline.
// Keep the list in step with SupportedExtensions["document"].
func IsTextFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".txt":
		return true
	}
	return false
}

// textReader decodes runes off a stream without holding the whole file. The
// BOM is read up front (Peek does not consume, so nothing is lost); what
// followed it stays in the buffer.
type textReader struct {
	b     *bufio.Reader
	enc   string
	order binary.ByteOrder
}

// newTextReader sniffs the BOM and picks an encoding. UTF-32 is refused rather
// than decoded wrong.
func newTextReader(r io.Reader) (*textReader, error) {
	b := bufio.NewReader(r)

	head, err := b.Peek(4)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}

	t := &textReader{b: b, enc: EncodingUTF8, order: binary.LittleEndian}
	switch {
	case len(head) >= 4 && head[0] == 0xFF && head[1] == 0xFE && head[2] == 0x00 && head[3] == 0x00:
		return nil, fmt.Errorf("utf-32le text is not supported")
	case len(head) >= 4 && head[0] == 0x00 && head[1] == 0x00 && head[2] == 0xFE && head[3] == 0xFF:
		return nil, fmt.Errorf("utf-32be text is not supported")
	case len(head) >= 3 && head[0] == 0xEF && head[1] == 0xBB && head[2] == 0xBF:
		_, _ = b.Discard(3)
	case len(head) >= 2 && head[0] == 0xFF && head[1] == 0xFE:
		t.enc, t.order = EncodingUTF16LE, binary.LittleEndian
		_, _ = b.Discard(2)
	case len(head) >= 2 && head[0] == 0xFE && head[1] == 0xFF:
		t.enc, t.order = EncodingUTF16BE, binary.BigEndian
		_, _ = b.Discard(2)
	}
	return t, nil
}

// nextRune returns the next rune and how many bytes of the file it took, so
// callers can meter their own byte limit precisely.
func (t *textReader) nextRune() (rune, int, error) {
	if t.enc == EncodingUTF8 {
		// bufio handles invalid bytes (RuneError, 1) and short tails; the size
		// it reports is the bytes consumed, which is what the meter needs.
		r, size, err := t.b.ReadRune()
		if err != nil {
			return 0, 0, err
		}
		return r, size, nil
	}

	head, err := t.b.Peek(2)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return 0, 0, err
	}
	if len(head) == 0 {
		return 0, 0, io.EOF
	}
	if len(head) == 1 {
		// File ends mid code unit.
		_, _ = t.b.Discard(1)
		return utf8.RuneError, 1, nil
	}
	_, _ = t.b.Discard(2)

	unit := t.order.Uint16(head)
	if !utf16.IsSurrogate(rune(unit)) {
		return rune(unit), 2, nil
	}

	low, err := t.b.Peek(2)
	if (err == nil || err == io.EOF || err == io.ErrUnexpectedEOF) && len(low) == 2 {
		_, _ = t.b.Discard(2)
		if combined := utf16.DecodeRune(rune(unit), rune(t.order.Uint16(low))); combined != utf8.RuneError {
			return combined, 4, nil
		}
	}
	// Lone or mismatched surrogate — one replacement character, no crash.
	return utf8.RuneError, 2, nil
}

// isBinaryText refuses files that only look like text: a NUL in the first
// NulScanBytes is UTF-16 without a BOM or straight binary, and either would
// preview as noise.
const nulScanBytes = 4096

func checkNotBinary(consumed int64, r rune) error {
	if r == 0 && consumed <= nulScanBytes {
		return fmt.Errorf("not a text file (NUL byte in the first %d bytes)", nulScanBytes)
	}
	return nil
}

// TextSample is the head of a text file, decoded to UTF-8.
type TextSample struct {
	Text      string `json:"text"`
	Encoding  string `json:"encoding"`
	BytesRead int64  `json:"bytes_read"`
	Truncated bool   `json:"truncated"`
}

// ReadTextSample reads at most limit bytes of a text file, decoded per its
// BOM. Truncated says there is more in the file than the sample holds.
func ReadTextSample(path string, limit int) (*TextSample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	t, err := newTextReader(f)
	if err != nil {
		return nil, err
	}

	var (
		b        strings.Builder
		consumed int64
		ranOut   bool
	)
	for consumed < int64(limit) {
		r, size, err := t.nextRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		consumed += int64(size)
		if err := checkNotBinary(consumed, r); err != nil {
			return nil, err
		}
		b.WriteRune(r)
	}
	// Stopped at the limit, not at the end: is there anything left?
	if consumed >= int64(limit) {
		if _, _, err := t.nextRune(); err == nil {
			ranOut = true
		}
	}

	return &TextSample{
		Text:      b.String(),
		Encoding:  t.enc,
		BytesRead: consumed,
		Truncated: ranOut,
	}, nil
}

// TextCounts is a line/word tally for a text file. Lines counts what a reader
// sees, so a file whose last line has no newline still counts it.
type TextCounts struct {
	Lines     int  `json:"lines"`
	Words     int  `json:"words"`
	Truncated bool `json:"truncated"`
}

// CountText tallies lines and words over at most limit bytes. Truncated says
// the file is bigger, in which case both tallies are floors, not totals.
func CountText(path string, limit int64) (*TextCounts, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	t, err := newTextReader(f)
	if err != nil {
		return nil, err
	}

	var (
		c         TextCounts
		consumed  int64
		inWord    bool
		openLine  bool
		reachedIO bool
	)
	for consumed < limit {
		r, size, err := t.nextRune()
		if err == io.EOF {
			reachedIO = true
			break
		}
		if err != nil {
			return nil, err
		}
		consumed += int64(size)
		if err := checkNotBinary(consumed, r); err != nil {
			return nil, err
		}

		switch {
		case r == '\n':
			c.Lines++
			openLine = false
			inWord = false
		case r == '\r':
			// CRLF, or lone CR from an old Mac file — a break, not a character.
			inWord = false
		case unicode.IsSpace(r):
			inWord = false
		default:
			if !inWord {
				c.Words++
				inWord = true
			}
			openLine = true
		}
	}
	// A final line is only counted when the file ended inside this read: a
	// partial read must not invent a line it never saw the end of.
	if openLine && reachedIO {
		c.Lines++
	}
	if !reachedIO {
		if _, _, err := t.nextRune(); err == nil {
			c.Truncated = true
		}
	}

	return &c, nil
}
