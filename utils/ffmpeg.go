package utils

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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
// timestamp and saves it as a JPEG at the output path.
// timestamp is in seconds (e.g. "1" for 1 second in).
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
		"-vf", "scale="+strconv.Itoa(size)+"."+strconv.Itoa(size)+":flags=lanczos", // resize with Lanczos
		"-q:v", "2",            // JPEG quality (2 = high quality, ~80% equivalent)
		"-f", "mjpeg",          // force MJPEG format
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
