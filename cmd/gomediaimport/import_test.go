package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImportMedia(t *testing.T) {
	// Create temporary directories for testing
	sourceDir, err := ioutil.TempDir("", "source")
	if err != nil {
		t.Fatalf("Failed to create temporary source directory: %v", err)
	}
	defer os.RemoveAll(sourceDir)

	destDir, err := ioutil.TempDir("", "dest")
	if err != nil {
		t.Fatalf("Failed to create temporary destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Create test files
	testFiles := []string{"test1.jpg", "test2.mp4", "test3.txt"}
	for _, file := range testFiles {
		_, err := os.Create(filepath.Join(sourceDir, file))
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Test case
	cfg := config{
		SourceDir:        sourceDir,
		DestDir:          destDir,
		OrganizeByDate:   true,
		RenameByDateTime: true,
		AutoRenameUnique: true,
	}

	err = importMedia(cfg)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	// TODO: Add more assertions here to check if the import was successful
	// For example, check if files were copied to the destination directory,
	// if they were renamed correctly, etc.
}

func TestEnumerateFiles(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := []string{"test1.jpg", "test2.mp4", "test3.txt"}
	for _, file := range testFiles {
		_, err := os.Create(filepath.Join(tempDir, file))
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Create a subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Test enumerateFiles
	files, err := enumerateFiles(tempDir)
	if err != nil {
		t.Fatalf("enumerateFiles failed: %v", err)
	}

	// Check if the correct number of media files were enumerated
	expectedCount := 2 // Only media files should be counted
	if len(files) != expectedCount {
		t.Errorf("Expected %d files, but got %d", expectedCount, len(files))
	}

	// Check if the enumerated files have correct information
	for _, file := range files {
		if file.SourceName != "test1.jpg" && file.SourceName != "test2.mp4" {
			t.Errorf("Unexpected file enumerated: %s", file.SourceName)
		}
		if file.SourceDir != tempDir {
			t.Errorf("Incorrect source directory for file %s: expected %s, got %s", file.SourceName, tempDir, file.SourceDir)
		}
	}

	// Test with non-existent directory
	_, err = enumerateFiles("/non/existent/dir")
	if err == nil {
		t.Error("Expected error for non-existent directory, but got none")
	}

	// Test with empty source directory
	emptyDir, err := ioutil.TempDir("", "empty")
	if err != nil {
		t.Fatalf("Failed to create empty temporary directory: %v", err)
	}
	defer os.RemoveAll(emptyDir)

	emptyFiles, err := enumerateFiles(emptyDir)
	if err != nil {
		t.Fatalf("enumerateFiles failed for empty directory: %v", err)
	}
	if len(emptyFiles) != 0 {
		t.Errorf("Expected 0 files in empty directory, but got %d", len(emptyFiles))
	}
}

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
