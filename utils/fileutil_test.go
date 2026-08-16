package utils

import "testing"

func TestMIMEType(t *testing.T) {
	cases := map[string]string{
		"photo.jpg":    "image/jpeg",
		"photo.jpeg":   "image/jpeg",
		"photo.png":    "image/png",
		"photo.webp":   "image/webp",
		"photo.svg":    "image/svg+xml",
		"video.mp4":    "video/mp4",
		"video.m4v":    "video/mp4",
		"video.mkv":    "video/x-matroska",
		"video.webm":   "video/webm",
		"video.mov":    "video/quicktime",
		"video.avi":    "video/x-msvideo",
		"video.m2ts":   "video/mp2t",
		"file.unknown": "application/octet-stream",
		"noext":        "application/octet-stream",
	}
	for path, want := range cases {
		if got := MIMEType(path); got != want {
			t.Errorf("MIMEType(%q) = %q, want %q", path, got, want)
		}
	}

	// Case-insensitive extension lookup
	if got := MIMEType("PHOTO.JPG"); got != "image/jpeg" {
		t.Errorf("MIMEType(uppercase) = %q, want image/jpeg", got)
	}
	// Nested path
	if got := MIMEType("/a/b/clip.mp4"); got != "video/mp4" {
		t.Errorf("MIMEType(nested) = %q, want video/mp4", got)
	}
}

func TestIsSupported(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		// Images
		{"photo.jpg", true},
		{"photo.jpeg", true},
		{"photo.png", true},
		{"photo.gif", true},
		{"photo.webp", true},
		{"photo.svg", true},
		{"photo.avif", true},
		{"photo.jxl", true},
		{"photo.jpegxl", true},
		{"photo.tiff", true},
		// Videos
		{"clip.mp4", true},
		{"clip.mkv", true},
		{"clip.m2ts", true},
		{"clip.mov", true},
		// Animated (in both image and animated categories)
		{"anim.gif", true},
		{"anim.webm", true},
		// Case-insensitive, nested paths
		{"PHOTO.JPG", true},
		{"/a/b/clip.MP4", true},
		// Unsupported
		{"notes.txt", false},
		{"archive.zip", false},
		{"noext", false},
		{"photo.jpg.bak", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			if got := IsSupported(tc.path); got != tc.want {
				t.Errorf("IsSupported(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}
