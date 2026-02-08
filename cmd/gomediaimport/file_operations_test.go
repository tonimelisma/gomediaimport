package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// Helper function to construct the eject command without running it
func constructEjectCommand(sourceDir string) *exec.Cmd {
	return exec.Command("diskutil", "eject", sourceDir)
}

func TestEjectDriveMacOS_CommandConstruction(t *testing.T) {
	sourceDir := "/test/source/dir"
	cmd := constructEjectCommand(sourceDir)

	expectedPath := "diskutil"
	if filepath.Base(cmd.Path) != expectedPath {
		t.Errorf("Expected command path %s, but got %s", expectedPath, cmd.Path)
	}

	expectedArgs := []string{"diskutil", "eject", sourceDir}
	if !reflect.DeepEqual(cmd.Args, expectedArgs) {
		t.Errorf("Expected command args %v, but got %v", expectedArgs, cmd.Args)
	}
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
	files, err := enumerateFiles(tempDir, false)
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
	_, err = enumerateFiles("/non/existent/dir", false)
	if err == nil {
		t.Error("Expected error for non-existent directory, but got none")
	}

	// Test with empty source directory
	emptyDir, err := ioutil.TempDir("", "empty")
	if err != nil {
		t.Fatalf("Failed to create empty temporary directory: %v", err)
	}
	defer os.RemoveAll(emptyDir)

	emptyFiles, err := enumerateFiles(emptyDir, false)
	if err != nil {
		t.Fatalf("enumerateFiles failed for empty directory: %v", err)
	}
	if len(emptyFiles) != 0 {
		t.Errorf("Expected 0 files in empty directory, but got %d", len(emptyFiles))
	}
}

func TestSetFinalDestinationFilename(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cfg := config{
		DestDir:            tempDir,
		ChecksumDuplicates: true,
	}

	files := []FileInfo{
		{
			SourceName:       "test.jpg",
			DestDir:          tempDir,
			CreationDateTime: time.Date(2023, 5, 1, 10, 30, 0, 0, time.UTC),
			FileType:         JPEG,
		},
	}

	initialFilename := "20230501_103000.jpg"
	sizeTimeIndex := make(map[fileSizeTime][]int)

	err = setFinalDestinationFilename(&files, 0, initialFilename, cfg, sizeTimeIndex)
	if err != nil {
		t.Errorf("setFinalDestinationFilename failed: %v", err)
	}

	if files[0].DestName != initialFilename {
		t.Errorf("Expected destination name %s, but got %s", initialFilename, files[0].DestName)
	}

	// Test with existing file
	_, err = os.Create(filepath.Join(tempDir, initialFilename))
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = setFinalDestinationFilename(&files, 0, initialFilename, cfg, sizeTimeIndex)
	if err != nil {
		t.Errorf("setFinalDestinationFilename failed: %v", err)
	}

	expectedName := "20230501_103000_001.jpg"
	if files[0].DestName != expectedName {
		t.Errorf("Expected destination name %s, but got %s", expectedName, files[0].DestName)
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
		destPath           string
		checksumDuplicates bool
		expected           bool
	}{
		{duplicateFile, true, true},
		{duplicateFile, false, true},
		{differentFile, true, false},
		{differentFile, false, false},
	}

	for _, tt := range tests {
		result := isDuplicate(fileInfo, tt.destPath, tt.checksumDuplicates)
		if result != tt.expected {
			t.Errorf("isDuplicate(%s, %v) = %v, expected %v", tt.destPath, tt.checksumDuplicates, result, tt.expected)
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
