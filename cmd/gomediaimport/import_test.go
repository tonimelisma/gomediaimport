package main

import (
	"testing"
	"time"
)

func TestFileInfo(t *testing.T) {
	// Test FileInfo struct
	now := time.Now()
	fi := FileInfo{
		SourceName:       "test.jpg",
		CreationDateTime: now,
		Size:             1024,
		MediaCategory:    MediaCategory("Image"),
		FileType:         FileType("JPEG"),
	}

	if fi.SourceName != "test.jpg" {
		t.Errorf("Expected SourceName 'test.jpg', but got '%s'", fi.SourceName)
	}

	if fi.Size != 1024 {
		t.Errorf("Expected Size 1024, but got %d", fi.Size)
	}

	if fi.MediaCategory != MediaCategory("Image") {
		t.Errorf("Expected MediaCategory Image, but got %v", fi.MediaCategory)
	}

	if fi.FileType != FileType("JPEG") {
		t.Errorf("Expected FileType JPEG, but got %v", fi.FileType)
	}

	if !fi.CreationDateTime.Equal(now) {
		t.Errorf("Expected CreationDateTime %v, but got %v", now, fi.CreationDateTime)
	}
}
