package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHumanReadableSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		result := humanReadableSize(tt.size)
		if result != tt.expected {
			t.Errorf("humanReadableSize(%d) = %q, want %q", tt.size, result, tt.expected)
		}
	}
}

func TestHumanReadableDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "0s"},
		{5 * time.Second, "5s"},
		{65 * time.Second, "1m5s"},
		{3661 * time.Second, "1h1m1s"},
		{90061 * time.Second, "1d1h1m1s"},
	}

	for _, tt := range tests {
		result := humanReadableDuration(tt.duration)
		if result != tt.expected {
			t.Errorf("humanReadableDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
		}
	}
}

func TestDeleteOriginalFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "delete-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source files
	copiedFile := filepath.Join(tmpDir, "copied.jpg")
	preExistingFile := filepath.Join(tmpDir, "preexisting.jpg")
	failedFile := filepath.Join(tmpDir, "failed.jpg")

	for _, f := range []string{copiedFile, preExistingFile, failedFile} {
		if err := os.WriteFile(f, []byte("data"), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", f, err)
		}
	}

	files := []FileInfo{
		{SourceName: "copied.jpg", SourceDir: tmpDir, Status: StatusCopied, Size: 4},
		{SourceName: "preexisting.jpg", SourceDir: tmpDir, Status: StatusPreExisting, Size: 4},
		{SourceName: "failed.jpg", SourceDir: tmpDir, Status: StatusFailed, Size: 4},
	}

	cfg := config{DeleteOriginals: true}
	if err := deleteOriginalFiles(files, cfg); err != nil {
		t.Fatalf("deleteOriginalFiles failed: %v", err)
	}

	// Copied and pre-existing should be deleted
	if _, err := os.Stat(copiedFile); !os.IsNotExist(err) {
		t.Error("copied file should have been deleted")
	}
	if _, err := os.Stat(preExistingFile); !os.IsNotExist(err) {
		t.Error("pre-existing file should have been deleted")
	}
	// Failed should NOT be deleted
	if _, err := os.Stat(failedFile); os.IsNotExist(err) {
		t.Error("failed file should NOT have been deleted")
	}
}

func TestCopyFilesSkipsPreExisting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "skip-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	files := []FileInfo{
		{SourceName: "a.jpg", SourceDir: tmpDir, DestName: "a.jpg", DestDir: tmpDir, Status: StatusUnnamable, Size: 100},
		{SourceName: "b.jpg", SourceDir: tmpDir, DestName: "b.jpg", DestDir: tmpDir, Status: StatusPreExisting, Size: 200},
	}

	cfg := config{DryRun: false}
	if err := copyFiles(files, cfg); err != nil {
		t.Fatalf("copyFiles failed: %v", err)
	}

	// Neither file should have been copied (both skipped)
	if files[0].Status != StatusUnnamable {
		t.Errorf("expected StatusUnnamable, got %v", files[0].Status)
	}
	if files[1].Status != StatusPreExisting {
		t.Errorf("expected StatusPreExisting, got %v", files[1].Status)
	}
}

func TestCopyFilesDryRun(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dryrun-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcFile := filepath.Join(tmpDir, "source.jpg")
	if err := os.WriteFile(srcFile, []byte("photo data"), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	destDir := filepath.Join(tmpDir, "dest")

	files := []FileInfo{
		{
			SourceName: "source.jpg",
			SourceDir:  tmpDir,
			DestName:   "source.jpg",
			DestDir:    destDir,
			Size:       10,
		},
	}

	cfg := config{DryRun: true}
	if err := copyFiles(files, cfg); err != nil {
		t.Fatalf("copyFiles failed: %v", err)
	}

	// Dest dir should not have been created
	if _, err := os.Stat(destDir); !os.IsNotExist(err) {
		t.Error("dest dir should not exist in dry run")
	}
}

func TestCopyFileErrors(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "copyerr-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test with non-existent source
	err = copyFile(filepath.Join(tmpDir, "nonexistent.jpg"), filepath.Join(tmpDir, "dest.jpg"))
	if err == nil {
		t.Error("Expected error copying non-existent source, got nil")
	}

	// Test with read-only destination directory
	srcFile := filepath.Join(tmpDir, "source.jpg")
	if err := os.WriteFile(srcFile, []byte("photo"), 0644); err != nil {
		t.Fatal(err)
	}

	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0555); err != nil {
		t.Fatal(err)
	}

	err = copyFile(srcFile, filepath.Join(readOnlyDir, "dest.jpg"))
	if err == nil {
		t.Error("Expected error copying to read-only directory, got nil")
	}

	// Restore permissions for cleanup
	os.Chmod(readOnlyDir, 0755)
}
