package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/rwcarlsen/goexif/exif"
	"golang.org/x/sys/windows"
)

// SupportedExtensions maps file categories to their extensions.
var SupportedExtensions = map[string][]string{
	"image":    {".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".tiff", ".tif", ".svg", ".avif", ".jxl", ".jpegxl"},
	"video":    {".mp4", ".mov", ".avi", ".webm", ".mkv", ".wmv", ".flv", ".m4v", ".3gp", ".3g2", ".vob", ".ogv", ".mpg", ".mpeg", ".m2v", ".ts", ".mts", ".m2ts", ".asf", ".rm", ".amv", ".f4v", ".dv", ".mxf"},
	"animated": {".gif", ".webm"},
}

// MIMETypes maps lowercase file extensions (with leading dot) to MIME types.
// Used by the HTTP file API — video players need the exact content type for
// seeking, and http.DetectContentType can't recognize every supported format
// (e.g. mkv, wmv, webm).
var MIMETypes = map[string]string{
	// Images
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".bmp":  "image/bmp",
	".tiff": "image/tiff",
	".tif":  "image/tiff",
	".svg":  "image/svg+xml",
	// Video
	".mp4":  "video/mp4",
	".m4v":  "video/mp4",
	".f4v":  "video/mp4",
	".mov":  "video/quicktime",
	".avi":  "video/x-msvideo",
	".webm": "video/webm",
	".mkv":  "video/x-matroska",
	".wmv":  "video/x-ms-wmv",
	".flv":  "video/x-flv",
	".3gp":  "video/3gpp",
	".3g2":  "video/3gpp2",
	".vob":  "video/mpeg",
	".mpg":  "video/mpeg",
	".mpeg": "video/mpeg",
	".m2v":  "video/mpeg",
	".ogv":  "video/ogg",
	".ts":   "video/mp2t",
	".mts":  "video/mp2t",
	".m2ts": "video/mp2t",
	".asf":  "video/x-ms-asf",
	".rm":   "application/vnd.rn-realmedia",
	".amv":  "video/avi",
	".dv":   "video/dv",
	".mxf":  "application/mxf",
}

// MIMEType returns the MIME type for a file's extension, or
// "application/octet-stream" for unknown extensions.
func MIMEType(path string) string {
	if ct, ok := MIMETypes[strings.ToLower(filepath.Ext(path))]; ok {
		return ct
	}
	return "application/octet-stream"
}

// IsSupported checks if a file extension is supported.
func IsSupported(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, exts := range SupportedExtensions {
		for _, e := range exts {
			if ext == e {
				return true
			}
		}
	}
	return false
}

// GetFileCategory returns the category of a file based on its extension.
// Order matters for extensions in multiple categories (e.g. .gif is both image and animated).
// We check in priority order: video > animated > image.
func GetFileCategory(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	// Check in priority order so extensions in multiple categories get the most specific one
	for _, category := range []string{"video", "animated", "image"} {
		for _, e := range SupportedExtensions[category] {
			if ext == e {
				return category
			}
		}
	}
	return "unknown"
}

// GetFormatName returns a human-readable format name from the extension.
func GetFormatName(path string) string {
	ext := strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), "."))
	if ext == "JPG" || ext == "JPEG" {
		return "JPEG"
	}
	if ext == "TIF" || ext == "TIFF" {
		return "TIFF"
	}
	if ext == "AVIF" {
		return "AVIF"
	}
	if ext == "JXL" || ext == "JPEGXL" {
		return "JPEG XL"
	}
	return ext
}

// GetFilename extracts the filename from a full path.
func GetFilename(path string) string {
	return filepath.Base(path)
}

// GetCreationTimeNanos returns the file creation time from EXIF metadata.
// Returns 0 if no EXIF data is available.
func GetCreationTimeNanos(path string) int64 {
	return getEXIFCreationTime(path)
}

// getEXIFCreationTime extracts the original creation time from EXIF metadata.
// Returns Unix nanoseconds, or 0 if not available.
func getEXIFCreationTime(path string) int64 {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	exifReader, err := exif.Decode(f)
	if err != nil {
		return 0
	}

	// Try tags in priority order: DateTimeOriginal, DateTimeDigitized, DateTime
	for _, tagName := range []string{"DateTimeOriginal", "DateTimeDigitized", "DateTime"} {
		tagVal, err := exifReader.Get(exif.FieldName(tagName))
		if err != nil {
			continue
		}
		s, err := tagVal.StringVal()
		if err != nil || s == "" {
			continue
		}
		// EXIF date format: "2006:01:02 15:04:05"
		t, err := time.Parse("2006:01:02 15:04:05", s)
		if err != nil {
			continue
		}
		return t.UnixNano()
	}
	return 0
}

// OpenWithDefaultApp opens a file with the default OS application.
func OpenWithDefaultApp(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", filepath.ToSlash(path))
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}

	return cmd.Run()
}

// OpenFolder opens a folder in the system file explorer.
func OpenFolder(folder string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Use cmd /c start — explorer.exe can return non-zero even on success
		// (it's a persistent process), so cmd /c start is more reliable.
		// The empty string "" is the window title — required so the folder path
		// is not misinterpreted as a title by the `start` command.
		cmd = exec.Command("cmd", "/c", "start", "", folder)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	case "darwin":
		cmd = exec.Command("open", folder)
	default:
		cmd = exec.Command("xdg-open", folder)
	}

	return cmd.Run()
}

// DeleteToTrash moves a file to the system trash/recycle bin.
func DeleteToTrash(path string) error {
	switch runtime.GOOS {
	case "windows":
		return moveToRecycleBin(path)
	case "darwin":
		cmd := exec.Command("osascript", "-e", fmt.Sprintf(`tell application "Finder" to delete POSIX file "%s"`, path))
		return cmd.Run()
	default:
		// Linux: use gio trash if available, fallback to .Trash
		cmd := exec.Command("gio", "trash", path)
		if err := cmd.Run(); err != nil {
			trashDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "Trash")
			_ = os.MkdirAll(trashDir, 0700)
			_ = os.Rename(path, filepath.Join(trashDir, filepath.Base(path)))
			return nil
		}
		return nil
	}
}

const (
	foDelete          = 3
	fofAllowUndo      = 0x40
	fofNoConfirmation = 0x10
)

var (
	shell32              = syscall.NewLazyDLL("shell32.dll")
	procSHFileOperationW = shell32.NewProc("SHFileOperationW")
)

// FileTimes holds creation and modification times extracted from os.FileInfo.
type FileTimes struct {
	CreatedAt  time.Time
	ModifiedAt time.Time
}

// GetFileTimes extracts creation and modification times from an os.FileInfo.
// On Windows, creation time comes from Win32FileAttributeData.CreationTime.
// On other platforms, creation time is left as zero (not set).
func GetFileTimes(info os.FileInfo) FileTimes {
	ft := FileTimes{
		ModifiedAt: info.ModTime(),
	}

	// Try Windows creation time
	sys := info.Sys()
	if win, ok := sys.(*syscall.Win32FileAttributeData); ok {
		nanos := win.CreationTime.Nanoseconds()
		if nanos > 0 {
			ft.CreatedAt = time.Unix(0, nanos)
			return ft
		}
	}

	// Fallback: leave CreatedAt as zero time (not set)
	// DO NOT fall back to ModTime — creation time should be the real creation time or empty
	return ft
}

// moveToRecycleBin moves a file to the Windows Recycle Bin via SHFileOperationW.
func moveToRecycleBin(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("get absolute path: %w", err)
	}

	// SHFileOperationW expects a double-null-terminated UTF-16 string.
	// windows.UTF16PtrFromString gives single-null, so we build our own
	// buffer with an extra \x00\x00 at the end.
	utf16Str, err := windows.UTF16FromString(absPath)
	if err != nil {
		return fmt.Errorf("UTF16 conversion: %w", err)
	}
	buf := make([]uint16, len(utf16Str)+1) // extra null for double-null termination
	copy(buf, utf16Str)

	// Must match C SHFILEOPSTRUCT layout exactly (64-bit Windows).
	// https://learn.microsoft.com/en-us/windows/win32/api/shellapi/ns-shellapi-shfileopstructw
	type shFileOpStruct struct {
		hwnd              uintptr
		wFunc             uint32
		pFrom, pTo        *uint16
		fFlags            uint16
		fAnyOpsAborted    bool
		hNameMappings     uintptr
		lpszProgressTitle *uint16
	}

	op := &shFileOpStruct{
		wFunc:  foDelete,
		pFrom:  &buf[0],
		fFlags: fofAllowUndo | fofNoConfirmation,
	}

	ret, _, _ := procSHFileOperationW.Call(
		uintptr(unsafe.Pointer(op)),
	)
	if ret != 0 {
		return fmt.Errorf("SHFileOperationW failed with code %d", ret)
	}
	return nil
}
