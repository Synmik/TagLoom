package utils

import (
	"encoding/json"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/gen2brain/svg"
)

const (
	// Default timestamp to extract video thumbnail (1 second in)
	DefaultVideoThumbTimestamp = "1"
)

// FFprobeVideoInfo holds video stream information extracted by ffprobe.
type FFprobeVideoInfo struct {
	DurationSeconds float64 `json:"duration_seconds"`
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	CodecName       string  `json:"codec_name"`
	BitRate         int64   `json:"bit_rate"`
}

// FFprobeResult is the parsed output of ffprobe JSON.
type FFprobeResult struct {
	Streams []struct {
		Index         int     `json:"index"`
		CodecType     string  `json:"codec_type"`
		CodecName     string  `json:"codec_name"`
		Width         int     `json:"width"`
		Height        int     `json:"height"`
		Duration      string  `json:"duration"`
		BitsPerRawSample string `json:"bits_per_raw_sample"`
		SampleRate    string  `json:"sample_rate"`
		BitRate       string  `json:"bit_rate"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
		Size     string `json:"size"`
	} `json:"format"`
}

// noWindowAttr creates SysProcAttr to hide the console window on Windows.
func noWindowAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

// FindFFmpeg locates the ffmpeg binary. Checks PATH first, then common install locations.
func FindFFmpeg() (string, error) {
	path, err := exec.LookPath("ffmpeg")
	if err == nil {
		return filepath.Clean(path), nil
	}
	// Common Windows install locations
	candidates := []string{
		"C:\\ffmpeg\\bin\\ffmpeg.exe",
		"C:\\Program Files\\ffmpeg\\bin\\ffmpeg.exe",
		"C:\\Program Files (x86)\\ffmpeg\\bin\\ffmpeg.exe",
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			return filepath.Clean(c), nil
		}
	}
	return "", exec.ErrNotFound
}

// FindFFprobe locates the ffprobe binary.
func FindFFprobe() (string, error) {
	path, err := exec.LookPath("ffprobe")
	if err == nil {
		return filepath.Clean(path), nil
	}
	// ffprobe is usually next to ffmpeg
	if ffmpegPath, err := FindFFmpeg(); err == nil {
		probePath := strings.Replace(ffmpegPath, "ffmpeg", "ffprobe", 1)
		if _, err := exec.LookPath(probePath); err == nil {
			return filepath.Clean(probePath), nil
		}
	}
	return "", exec.ErrNotFound
}

// ExtractVideoFrame uses ffmpeg to extract a single frame from a video at the given
// timestamp and saves it as a WebP at the output path.
// timestamp is in seconds (e.g. "1" for 1 second in).
//
// EncodeImageToWebP uses ffmpeg to convert an image file to WebP format at the given size.
// It supports all image formats FFmpeg can decode (jpg, png, gif, webp, bmp, tiff, etc.).
func ExtractVideoFrame(videoPath, outputJpg string, size int, timestamp string) error {
	ffmpeg, err := FindFFmpeg()
	if err != nil {
		return &FFmpegError{Msg: "ffmpeg not found", Err: err}
	}

	// -ss before -i is fast seek (keyframe), -ss after -i is precise seek but slower.
	// We use -ss before -i for speed, which is fine for thumbnails.
	cmd := exec.Command(ffmpeg,
		"-y",                   // overwrite output
		"-ss", timestamp,       // seek to timestamp (fast seek)
		"-i", videoPath,        // input file
		"-vframes", "1",        // extract exactly 1 frame
		"-vf", "scale="+strconv.Itoa(size)+":"+strconv.Itoa(size)+":force_original_aspect_ratio=decrease:flags=lanczos", // fit within box, keep aspect ratio
		"-q:v", "50",           // WebP quality (0-100, 50 ≈ 80% JPEG equivalent)
		"-f", "webp",           // force WebP format
		outputJpg,
	)
	cmd.SysProcAttr = noWindowAttr()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &FFmpegError{
			Msg:   "failed to extract video frame",
			Err:   err,
			Stderr: string(output),
		}
	}
	return nil
}

// EncodeImageToWebP converts an image file to WebP format using FFmpeg.
// The output is resized to fit within size×size while maintaining aspect ratio.
func EncodeImageToWebP(imagePath, outputWebp string, size, quality int) error {
	ffmpeg, err := FindFFmpeg()
	if err != nil {
		return &FFmpegError{Msg: "ffmpeg not found", Err: err}
	}

	cmd := exec.Command(ffmpeg,
		"-y",                   // overwrite output
		"-i", imagePath,        // input file
		"-vframes", "1",        // extract exactly 1 frame (for GIFs/animated)
		"-vf", "scale="+strconv.Itoa(size)+":"+strconv.Itoa(size)+":force_original_aspect_ratio=decrease:flags=lanczos", // fit within box, keep aspect ratio
		"-c:v", "libwebp",     // force static WebP (not libwebp_anim)
		"-q:v", strconv.Itoa(quality), // WebP quality (0-100)
		"-f", "webp",           // force WebP format
		outputWebp,
	)
	cmd.SysProcAttr = noWindowAttr()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &FFmpegError{
			Msg:   "failed to encode image to WebP",
			Err:   err,
			Stderr: string(output),
		}
	}
	return nil
}

// ProbeVideo uses ffprobe to extract video metadata (duration, resolution, codec).
func ProbeVideo(videoPath string) (*FFprobeVideoInfo, error) {
	ffprobe, err := FindFFprobe()
	if err != nil {
		return nil, &FFmpegError{Msg: "ffprobe not found", Err: err}
	}

	cmd := exec.Command(ffprobe,
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		videoPath,
	)
	cmd.SysProcAttr = noWindowAttr()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, &FFmpegError{
			Msg:    "ffprobe failed",
			Err:    err,
			Stderr: string(output),
		}
	}

	var result FFprobeResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, &FFmpegError{Msg: "failed to parse ffprobe output", Err: err}
	}

	info := &FFprobeVideoInfo{}

	// Find the first video stream
	for _, stream := range result.Streams {
		if stream.CodecType == "video" {
			info.Width = stream.Width
			info.Height = stream.Height
			info.CodecName = stream.CodecName
			if stream.BitRate != "" {
				info.BitRate, _ = strconv.ParseInt(stream.BitRate, 10, 64)
			}
			// Use stream duration if available
			if stream.Duration != "" {
				if d, err := strconv.ParseFloat(stream.Duration, 64); err == nil {
					info.DurationSeconds = d
				}
			}
			break
		}
	}

	// Fallback: use format duration if stream duration not available
	if info.DurationSeconds == 0 && result.Format.Duration != "" {
		info.DurationSeconds, _ = strconv.ParseFloat(result.Format.Duration, 64)
	}

	return info, nil
}

// EncodeSVGToWebP rasterizes an SVG file to a WebP thumbnail.
// Uses gen2brain/svg to decode the SVG, then writes a temp PNG that is
// converted to WebP via FFmpeg. The temp PNG is cleaned up automatically.
func EncodeSVGToWebP(svgPath, outputWebp string, size, quality int) error {
	// Read SVG file
	data, err := os.ReadFile(svgPath)
	if err != nil {
		return &FFmpegError{Msg: "failed to read SVG file", Err: err}
	}

	// Decode SVG to image
	img, err := svg.Decode(strings.NewReader(string(data)))
	if err != nil {
		return &FFmpegError{Msg: "failed to decode SVG", Err: err}
	}

	// Create temp PNG file
	tmpFile, err := os.CreateTemp("", "svg-thumb-*.png")
	if err != nil {
		return &FFmpegError{Msg: "failed to create temp file", Err: err}
	}
	tmpPath := tmpFile.Name()
	// Ensure cleanup
	defer os.Remove(tmpPath)

	if err := png.Encode(tmpFile, img); err != nil {
		tmpFile.Close()
		return &FFmpegError{Msg: "failed to encode temp PNG", Err: err}
	}
	tmpFile.Close()

	// Convert PNG → WebP via FFmpeg
	return EncodeImageToWebP(tmpPath, outputWebp, size, quality)
}

// FFmpegError provides detailed error information for FFmpeg/ffprobe failures.
type FFmpegError struct {
	Msg    string
	Err    error
	Stderr string
}

func (e *FFmpegError) Error() string {
	if e.Stderr != "" {
		return e.Msg + ": " + e.Stderr
	}
	return e.Msg + ": " + e.Err.Error()
}

func (e *FFmpegError) Unwrap() error {
	return e.Err
}
