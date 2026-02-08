package main

import (
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
	tempDir, err := os.MkdirTemp("", "test")
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
	emptyDir, err := os.MkdirTemp("", "empty")
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
	tempDir, err := os.MkdirTemp("", "test")
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
	tempDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	sourceFile := filepath.Join(tempDir, "source.txt")
	duplicateFile := filepath.Join(tempDir, "duplicate.txt")
	differentFile := filepath.Join(tempDir, "different.txt")

	content := []byte("test content")
	err = os.WriteFile(sourceFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	err = os.WriteFile(duplicateFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create duplicate file: %v", err)
	}
	err = os.WriteFile(differentFile, []byte("different content"), 0644)
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

func TestCalculateXXHash(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := []byte("test content")
	err = os.WriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	checksum, err := calculateXXHash(testFile)
	if err != nil {
		t.Errorf("calculateXXHash failed: %v", err)
	}

	expectedChecksum := "0e6882304e9adbd5"
	if checksum != expectedChecksum {
		t.Errorf("Expected checksum %s, but got %s", expectedChecksum, checksum)
	}

	// Test with non-existent file
	_, err = calculateXXHash(filepath.Join(tempDir, "non-existent.txt"))
	if err == nil {
		t.Error("Expected error for non-existent file, but got none")
	}
}

func TestSetFileTimes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "setfiletimes-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	targetTime := time.Date(2020, 6, 15, 12, 0, 0, 0, time.UTC)
	if err := setFileTimes(testFile, targetTime); err != nil {
		t.Fatalf("setFileTimes failed: %v", err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	if !info.ModTime().Equal(targetTime) {
		t.Errorf("Expected mod time %v, got %v", targetTime, info.ModTime())
	}
}

func TestIsDuplicateInPreviousFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dupcheck-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := []byte("duplicate content")
	differentContent := []byte("different stuff!")

	// Create source files
	if err := os.WriteFile(filepath.Join(tmpDir, "file1.jpg"), content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file2.jpg"), content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file3.jpg"), differentContent, 0644); err != nil {
		t.Fatal(err)
	}

	now := time.Now()

	// Test with checksum-disabled path (size+time match is enough)
	files := []FileInfo{
		{SourceName: "file1.jpg", SourceDir: tmpDir, Size: int64(len(content)), CreationDateTime: now},
		{SourceName: "file2.jpg", SourceDir: tmpDir, Size: int64(len(content)), CreationDateTime: now},
	}
	sizeTimeIndex := map[fileSizeTime][]int{
		{Size: int64(len(content)), Timestamp: now}: {0},
	}

	result := isDuplicateInPreviousFiles(&files, 1, false, sizeTimeIndex)
	if !result {
		t.Error("Expected duplicate without checksum (same size+time), got false")
	}

	// Test with checksum-enabled path — matching content
	files[0].SourceChecksum = ""
	files[1].SourceChecksum = ""
	result = isDuplicateInPreviousFiles(&files, 1, true, sizeTimeIndex)
	if !result {
		t.Error("Expected duplicate with checksum (same content), got false")
	}

	// Test with checksum-enabled path — different content but same size
	files2 := []FileInfo{
		{SourceName: "file1.jpg", SourceDir: tmpDir, Size: int64(len(content)), CreationDateTime: now},
		{SourceName: "file3.jpg", SourceDir: tmpDir, Size: int64(len(differentContent)), CreationDateTime: now},
	}
	sizeTimeIndex2 := map[fileSizeTime][]int{
		{Size: int64(len(content)), Timestamp: now}:          {0},
		{Size: int64(len(differentContent)), Timestamp: now}: {0}, // deliberately share index entry
	}

	result = isDuplicateInPreviousFiles(&files2, 1, true, sizeTimeIndex2)
	if result {
		t.Error("Expected no duplicate with checksum (different content), got true")
	}

	// Test with no matching index entry
	emptyIndex := map[fileSizeTime][]int{}
	result = isDuplicateInPreviousFiles(&files, 1, false, emptyIndex)
	if result {
		t.Error("Expected no duplicate with empty index, got true")
	}
}

func TestCopyFilesActualCopy(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "actualcopy-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	content1 := []byte("photo data one")
	content2 := []byte("photo data two")
	if err := os.WriteFile(filepath.Join(srcDir, "a.jpg"), content1, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "b.jpg"), content2, 0644); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	files := []FileInfo{
		{SourceName: "a.jpg", SourceDir: srcDir, DestName: "a.jpg", DestDir: destDir, Size: int64(len(content1)), CreationDateTime: now},
		{SourceName: "b.jpg", SourceDir: srcDir, DestName: "b.jpg", DestDir: destDir, Size: int64(len(content2)), CreationDateTime: now},
	}

	cfg := config{DryRun: false}
	if err := copyFiles(files, cfg); err != nil {
		t.Fatalf("copyFiles failed: %v", err)
	}

	// Verify files were copied
	for i, fi := range files {
		if fi.Status != StatusCopied {
			t.Errorf("files[%d] status = %v, want StatusCopied", i, fi.Status)
		}
		destPath := filepath.Join(destDir, fi.DestName)
		data, err := os.ReadFile(destPath)
		if err != nil {
			t.Errorf("Failed to read copied file %s: %v", destPath, err)
			continue
		}
		var expected []byte
		if i == 0 {
			expected = content1
		} else {
			expected = content2
		}
		if string(data) != string(expected) {
			t.Errorf("files[%d] content mismatch: got %q, want %q", i, data, expected)
		}
	}
}
