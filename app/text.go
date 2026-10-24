package app

import (
	"fmt"

	"TagLoom/utils"
)

// TextContent is the text of a Markdown or plain-text file, decoded to UTF-8
// and capped, for the inline preview in the detail panel and the modal.
//
// It travels as a binding rather than through /api/original because the file
// has to be decoded (BOM, UTF-16) and cut to a sane size before a web view
// sees it, and because a preview should not pull a multi-gigabyte .txt across
// the asset server.
type TextContent struct {
	FileID    int64  `json:"file_id"`
	Text      string `json:"text"`
	Encoding  string `json:"encoding"` // utf-8 | utf-16le | utf-16be
	BytesRead int64  `json:"bytes_read"`
	Truncated bool   `json:"truncated"` // the file continues past what is shown
}

// GetTextContent reads the head of a .md or .txt file for inline display.
// Anything else is refused: the binding takes a file ID from the UI, and it
// should not be a way to read an arbitrary file out of the vault.
func (a *App) GetTextContent(fileID int64) (*TextContent, error) {
	v := a.vault()
	if v.db == nil {
		return nil, fmt.Errorf("no vault open")
	}

	file, err := a.GetFileByID(fileID)
	if err != nil {
		return nil, fmt.Errorf("file not found: %w", err)
	}

	absPath := v.resolvePath(file.VaultPath)
	if !utils.IsTextFile(absPath) {
		return nil, fmt.Errorf("%s is not a text file", utils.GetFilename(absPath))
	}

	sample, err := utils.ReadTextSample(absPath, utils.TextPreviewLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", utils.GetFilename(absPath), err)
	}

	return &TextContent{
		FileID:    fileID,
		Text:      sample.Text,
		Encoding:  sample.Encoding,
		BytesRead: sample.BytesRead,
		Truncated: sample.Truncated,
	}, nil
}
