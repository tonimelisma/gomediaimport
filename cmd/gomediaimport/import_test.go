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

func TestSetDestinationFilename(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := config{
		DestDir:          tempDir,
		AutoRenameUnique: true,
	}

	file := &FileInfo{
		SourceName:       "test.jpg",
		DestDir:          tempDir,
		CreationDateTime: time.Date(2023, 5, 1, 10, 30, 0, 0, time.UTC),
		FileType:         JPEG,
	}

	err = setDestinationFilename(file, cfg)
	if err != nil {
		t.Errorf("setDestinationFilename failed: %v", err)
	}

	expectedName := "20230501_103000000.jpg"
	if file.DestName != expectedName {
		t.Errorf("Expected destination name %s, but got %s", expectedName, file.DestName)
	}

	// Test with existing file
	_, err = os.Create(filepath.Join(tempDir, expectedName))
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = setDestinationFilename(file, cfg)
	if err != nil {
		t.Errorf("setDestinationFilename failed: %v", err)
	}

	if file.DestName == expectedName {
		t.Errorf("Expected a different destination name, but got the same: %s", file.DestName)
	}
}

func TestIsDuplicate(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	sourceFile := filepath.Join(tempDir, "source.txt")
	duplicateFile := filepath.Join(tempDir, "duplicate.txt")
	differentFile := filepath.Join(tempDir, "different.txt")

	content := []byte("test content")
	err = ioutil.WriteFile(sourceFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	err = ioutil.WriteFile(duplicateFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create duplicate file: %v", err)
	}
	err = ioutil.WriteFile(differentFile, []byte("different content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create different file: %v", err)
	}

	fileInfo := &FileInfo{
		SourceName: "source.txt",
		SourceDir:  tempDir,
		Size:       int64(len(content)),
	}

	tests := []struct {
		destPath         string
		autoRenameUnique bool
		expected         bool
	}{
		{duplicateFile, true, true},
		{duplicateFile, false, true},
		{differentFile, true, false},
		{differentFile, false, false}, // Changed this to false
	}

	for _, tt := range tests {
		result := isDuplicate(fileInfo, tt.destPath, tt.autoRenameUnique)
		if result != tt.expected {
			t.Errorf("isDuplicate(%s, %v) = %v, expected %v", tt.destPath, tt.autoRenameUnique, result, tt.expected)
		}
	}
}

func TestCalculateCRC32(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := []byte("test content")
	err = ioutil.WriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	checksum, err := calculateCRC32(testFile)
	if err != nil {
		t.Errorf("calculateCRC32 failed: %v", err)
	}

	expectedChecksum := "57f4675d"
	if checksum != expectedChecksum {
		t.Errorf("Expected checksum %s, but got %s", expectedChecksum, checksum)
	}

	// Test with non-existent file
	_, err = calculateCRC32(filepath.Join(tempDir, "non-existent.txt"))
	if err == nil {
		t.Error("Expected error for non-existent file, but got none")
	}
}
