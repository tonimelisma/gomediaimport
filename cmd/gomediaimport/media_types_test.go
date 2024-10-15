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

// TestFileExtensionToFileTypeCompleteness checks if all FileType constants are mapped in fileExtensionToFileType
func TestFileExtensionToFileTypeCompleteness(t *testing.T) {
	allFileTypes := []FileType{
		JPEG, JPEG2000, JPEGXL, PNG, GIF, BMP, TIFF, PSD, EPS, SVG, ICO, WEBP, HEIF,
		RAW,
		MP4, AVI, MOV, WMV, FLV, MKV, WEBM, OGV, M4V, THREEGP, THREEG2, ASF, VOB, MTS,
		RAWVIDEO,
	}

	for _, fileType := range allFileTypes {
		found := false
		for _, mappedType := range fileExtensionToFileType {
			if mappedType == fileType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("FileType %v is not mapped in fileExtensionToFileType", fileType)
		}
	}
}

// TestFileTypeToMediaCategoryCompleteness checks if all FileType constants are mapped in fileTypeToMediaCategory
func TestFileTypeToMediaCategoryCompleteness(t *testing.T) {
	allFileTypes := []FileType{
		JPEG, JPEG2000, JPEGXL, PNG, GIF, BMP, TIFF, PSD, EPS, SVG, ICO, WEBP, HEIF,
		RAW,
		MP4, AVI, MOV, WMV, FLV, MKV, WEBM, OGV, M4V, THREEGP, THREEG2, ASF, VOB, MTS,
		RAWVIDEO,
	}

	for _, fileType := range allFileTypes {
		if _, ok := fileTypeToMediaCategory[fileType]; !ok {
			t.Errorf("FileType %v is not mapped in fileTypeToMediaCategory", fileType)
		}
	}
}
