package main

import (
	"testing"
)

// TestGetMediaTypeInfo tests the getMediaTypeInfo function
func TestGetMediaTypeInfo(t *testing.T) {
	testCases := []struct {
		name          string
		input         FileInfo
		expectedCat   MediaCategory
		expectedType  FileType
		shouldBeEmpty bool
	}{
		{"JPEG file", FileInfo{SourceName: "test.jpg"}, ProcessedPicture, JPEG, false},
		{"PNG file", FileInfo{SourceName: "image.png"}, ProcessedPicture, PNG, false},
		{"RAW file", FileInfo{SourceName: "photo.cr2"}, RawPicture, RAW, false},
		{"MP4 video", FileInfo{SourceName: "video.mp4"}, Video, MP4, false},
		{"Raw video", FileInfo{SourceName: "footage.braw"}, RawVideo, RAWVIDEO, false},

		// Test different extensions for the same type
		{"JPEG alternate extension", FileInfo{SourceName: "photo.jpeg"}, ProcessedPicture, JPEG, false},
		{"JPEG2000 extension", FileInfo{SourceName: "image.jp2"}, ProcessedPicture, JPEG2000, false},

		// Test case sensitivity
		{"Uppercase extension", FileInfo{SourceName: "IMAGE.PNG"}, ProcessedPicture, PNG, false},
		{"Mixed case extension", FileInfo{SourceName: "Video.Mp4"}, Video, MP4, false},

		// Test unknown extensions
		{"Unknown extension", FileInfo{SourceName: "file.xyz"}, "", "", true},

		// Test no extension
		{"No extension", FileInfo{SourceName: "filename"}, "", "", true},

		// Test with path
		{"File with path", FileInfo{SourceName: "/path/to/image.jpg"}, ProcessedPicture, JPEG, false},

		// Test with hidden file
		{"Hidden file", FileInfo{SourceName: ".hidden.png"}, ProcessedPicture, PNG, false},

		// Test edge cases
		{"Empty filename", FileInfo{SourceName: ""}, "", "", true},
		{"Only extension", FileInfo{SourceName: ".jpg"}, ProcessedPicture, JPEG, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cat, fileType := getMediaTypeInfo(tc.input)

			if tc.shouldBeEmpty {
				if cat != "" || fileType != "" {
					t.Errorf("Expected empty results, got category %v and type %v", cat, fileType)
				}
			} else {
				if cat != tc.expectedCat {
					t.Errorf("Expected category %v, got %v", tc.expectedCat, cat)
				}
				if fileType != tc.expectedType {
					t.Errorf("Expected file type %v, got %v", tc.expectedType, fileType)
				}
			}
		})
	}
}

// TestFileTypesCompleteness checks if all FileType constants are included in the fileTypes slice
func TestFileTypesCompleteness(t *testing.T) {
	allFileTypes := []FileType{
		JPEG, JPEG2000, JPEGXL, PNG, GIF, BMP, TIFF, PSD, EPS, SVG, ICO, WEBP, HEIF,
		RAW,
		MP4, AVI, MOV, WMV, FLV, MKV, WEBM, OGV, M4V, THREEGP, THREEG2, ASF, VOB, MTS,
		RAWVIDEO,
	}

	for _, fileType := range allFileTypes {
		found := false
		for _, ft := range fileTypes {
			if ft.FileType == fileType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("FileType %v is not included in the fileTypes slice", fileType)
		}
	}
}

// TestJPEGDefaultExtension checks if "jpg" is the default extension for JPEG files
func TestJPEGDefaultExtension(t *testing.T) {
	jpegExtension := getFirstExtensionForFileType(JPEG)
	if jpegExtension != "jpg" {
		t.Errorf("Expected 'jpg' to be the default extension for JPEG files, but got '%s'", jpegExtension)
	}
}

// TestGetFirstExtensionForFileType tests the getFirstExtensionForFileType function
func TestGetFirstExtensionForFileType(t *testing.T) {
	testCases := []struct {
		fileType          FileType
		expectedExtension string
	}{
		{JPEG, "jpg"},
		{PNG, "png"},
		{RAW, "arw"},
		{MP4, "mp4"},
		{RAWVIDEO, "braw"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.fileType), func(t *testing.T) {
			extension := getFirstExtensionForFileType(tc.fileType)
			if extension != tc.expectedExtension {
				t.Errorf("Expected extension %s for %s, but got %s", tc.expectedExtension, tc.fileType, extension)
			}
		})
	}
}

func TestIsSidecarExtension(t *testing.T) {
	tests := []struct {
		ext      string
		expected bool
	}{
		{"thm", true},
		{"ctg", true},
		{"xmp", true},
		{"aae", true},
		{"lrf", true},
		{"srt", true},
		{"jpg", false},
		{"mp4", false},
		{"txt", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			if got := isSidecarExtension(tt.ext); got != tt.expected {
				t.Errorf("isSidecarExtension(%q) = %v, want %v", tt.ext, got, tt.expected)
			}
		})
	}
}

func TestGetSidecarAction(t *testing.T) {
	tests := []struct {
		name      string
		ext       string
		overrides map[string]SidecarAction
		defAction SidecarAction
		expected  SidecarAction
	}{
		{"built-in default for xmp", "xmp", nil, SidecarDelete, SidecarCopy},
		{"built-in default for thm", "thm", nil, SidecarCopy, SidecarDelete},
		{"override beats built-in", "xmp", map[string]SidecarAction{"xmp": SidecarDelete}, SidecarIgnore, SidecarDelete},
		{"override beats global default", "foo", map[string]SidecarAction{"foo": SidecarCopy}, SidecarDelete, SidecarCopy},
		{"global default for unknown ext", "zzz", nil, SidecarIgnore, SidecarIgnore},
		{"global default fallthrough", "zzz", map[string]SidecarAction{}, SidecarDelete, SidecarDelete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSidecarAction(tt.ext, tt.overrides, tt.defAction)
			if got != tt.expected {
				t.Errorf("getSidecarAction(%q, ...) = %q, want %q", tt.ext, got, tt.expected)
			}
		})
	}
}

func TestIsValidSidecarAction(t *testing.T) {
	tests := []struct {
		action   SidecarAction
		expected bool
	}{
		{SidecarIgnore, true},
		{SidecarCopy, true},
		{SidecarDelete, true},
		{SidecarAction("invalid"), false},
		{SidecarAction(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			if got := isValidSidecarAction(tt.action); got != tt.expected {
				t.Errorf("isValidSidecarAction(%q) = %v, want %v", tt.action, got, tt.expected)
			}
		})
	}
}
