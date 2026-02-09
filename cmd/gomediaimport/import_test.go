package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

func TestSidecarCopyFollowsParentRename(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sidecar-parent-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a JPEG and its XMP sidecar
	if err := os.WriteFile(filepath.Join(srcDir, "IMG_001.jpg"), []byte("jpeg data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "IMG_001.xmp"), []byte("xmp data"), 0644); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)

	cfg := config{
		SourceDir:      srcDir,
		DestDir:        destDir,
		RenameByDateTime: true,
		SidecarDefault: SidecarDelete,
		Sidecars:       map[string]SidecarAction{},
	}

	files, err := enumerateFiles(srcDir, cfg)
	if err != nil {
		t.Fatalf("enumerateFiles failed: %v", err)
	}

	// Set CreationDateTime on all files for consistent testing
	for i := range files {
		files[i].CreationDateTime = now
		files[i].ParentIndex = -1
	}

	// Pass 1: process media files
	sizeTimeIndex := make(map[fileSizeTime][]int)
	for i := range files {
		if files[i].MediaCategory == Sidecar {
			continue
		}
		files[i].DestDir = destDir
		initialFilename := now.Format("20060102_150405") + filepath.Ext(files[i].SourceName)
		if err := setFinalDestinationFilename(&files, i, initialFilename, cfg, sizeTimeIndex); err != nil {
			t.Fatalf("setFinalDestinationFilename failed: %v", err)
		}
		key := fileSizeTime{Size: files[i].Size, Timestamp: files[i].CreationDateTime}
		sizeTimeIndex[key] = append(sizeTimeIndex[key], i)
	}

	// Build parent index
	type parentKey struct {
		dir      string
		baseName string
	}
	parentIdx := make(map[parentKey]int)
	for i := range files {
		if files[i].MediaCategory == Sidecar {
			continue
		}
		ext := filepath.Ext(files[i].SourceName)
		base := files[i].SourceName[:len(files[i].SourceName)-len(ext)]
		key := parentKey{dir: files[i].SourceDir, baseName: strings.ToLower(base)}
		if _, exists := parentIdx[key]; !exists {
			parentIdx[key] = i
		}
	}

	// Pass 2: process sidecars
	for i := range files {
		if files[i].MediaCategory != Sidecar {
			continue
		}
		sidecarExt := strings.ToLower(filepath.Ext(files[i].SourceName))
		if sidecarExt != "" {
			sidecarExt = sidecarExt[1:]
		}
		action := getSidecarAction(sidecarExt, cfg.Sidecars, cfg.SidecarDefault)
		if action == SidecarDelete {
			files[i].Status = StatusSidecarDeleted
			continue
		}

		ext := filepath.Ext(files[i].SourceName)
		base := files[i].SourceName[:len(files[i].SourceName)-len(ext)]
		key := parentKey{dir: files[i].SourceDir, baseName: strings.ToLower(base)}
		if pi, ok := parentIdx[key]; ok {
			files[i].ParentIndex = pi
			parentFile := files[pi]
			files[i].DestDir = parentFile.DestDir
			parentDestExt := filepath.Ext(parentFile.DestName)
			parentDestBase := parentFile.DestName[:len(parentFile.DestName)-len(parentDestExt)]
			files[i].DestName = parentDestBase + ext
		}
	}

	// Verify: XMP should follow parent's renamed destination
	for _, f := range files {
		if f.SourceName == "IMG_001.xmp" {
			if f.DestName != "20240315_143000.xmp" {
				t.Errorf("Expected sidecar dest name 20240315_143000.xmp, got %s", f.DestName)
			}
			if f.DestDir != destDir {
				t.Errorf("Expected sidecar dest dir %s, got %s", destDir, f.DestDir)
			}
			return
		}
	}
	t.Error("XMP sidecar not found in files list")
}

func TestSidecarDeleteAction(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sidecar-delete-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create media + THM sidecar
	if err := os.WriteFile(filepath.Join(srcDir, "VID_001.mp4"), []byte("video data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "VID_001.thm"), []byte("thumb data"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config{
		SourceDir:       srcDir,
		DestDir:         destDir,
		SidecarDefault:  SidecarDelete,
		Sidecars:        map[string]SidecarAction{},
		DeleteOriginals: true,
	}

	if err := importMedia(cfg); err != nil {
		t.Fatalf("importMedia failed: %v", err)
	}

	// THM should be deleted from source (sidecar delete + delete originals)
	if _, err := os.Stat(filepath.Join(srcDir, "VID_001.thm")); !os.IsNotExist(err) {
		t.Error("THM sidecar should have been deleted from source")
	}

	// MP4 should be deleted from source (copied + delete originals)
	if _, err := os.Stat(filepath.Join(srcDir, "VID_001.mp4")); !os.IsNotExist(err) {
		t.Error("MP4 should have been deleted from source")
	}

	// MP4 should exist in dest
	if _, err := os.Stat(filepath.Join(destDir, "VID_001.mp4")); os.IsNotExist(err) {
		t.Error("MP4 should exist in destination")
	}

	// THM should NOT exist in dest (it was action=delete, not copy)
	if _, err := os.Stat(filepath.Join(destDir, "VID_001.thm")); !os.IsNotExist(err) {
		t.Error("THM sidecar should NOT exist in destination")
	}
}

func TestOrphanedSidecarCopiedIndependently(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orphan-sidecar-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create ONLY a sidecar file (no parent media)
	if err := os.WriteFile(filepath.Join(srcDir, "IMG_001.xmp"), []byte("xmp data"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config{
		SourceDir:      srcDir,
		DestDir:        destDir,
		SidecarDefault: SidecarDelete,
		Sidecars:       map[string]SidecarAction{},
	}
	// xmp built-in default is copy, so it should be copied independently

	if err := importMedia(cfg); err != nil {
		t.Fatalf("importMedia failed: %v", err)
	}

	// XMP should exist in dest
	if _, err := os.Stat(filepath.Join(destDir, "IMG_001.xmp")); os.IsNotExist(err) {
		t.Error("Orphaned XMP sidecar should have been copied to destination")
	}
}

func TestOrphanedSidecarDeletedWithDeleteOriginals(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "orphan-del-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create orphaned CTG and XMP files (no parent media)
	if err := os.WriteFile(filepath.Join(srcDir, "index.ctg"), []byte("ctg data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "IMG_001.xmp"), []byte("xmp data"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config{
		SourceDir:       srcDir,
		DestDir:         destDir,
		SidecarDefault:  SidecarDelete,
		Sidecars:        map[string]SidecarAction{},
		DeleteOriginals: true,
	}

	if err := importMedia(cfg); err != nil {
		t.Fatalf("importMedia failed: %v", err)
	}

	// CTG should be deleted from source (action=delete + delete_originals)
	if _, err := os.Stat(filepath.Join(srcDir, "index.ctg")); !os.IsNotExist(err) {
		t.Error("CTG should have been deleted from source")
	}

	// XMP should be deleted from source (action=copy → copied → delete_originals)
	if _, err := os.Stat(filepath.Join(srcDir, "IMG_001.xmp")); !os.IsNotExist(err) {
		t.Error("XMP should have been deleted from source after copy")
	}

	// XMP should exist in dest
	if _, err := os.Stat(filepath.Join(destDir, "IMG_001.xmp")); os.IsNotExist(err) {
		t.Error("XMP should have been copied to destination")
	}

	// CTG should NOT exist in dest
	if _, err := os.Stat(filepath.Join(destDir, "index.ctg")); !os.IsNotExist(err) {
		t.Error("CTG should NOT exist in destination")
	}
}

func TestEffectiveWorkers(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{0, 4},
		{-1, 4},
		{-100, 4},
		{1, 1},
		{4, 4},
		{8, 8},
		{16, 16},
	}

	for _, tt := range tests {
		result := effectiveWorkers(tt.input)
		if result != tt.expected {
			t.Errorf("effectiveWorkers(%d) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestCopyFilesNonTTY(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nontty-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a few source files
	numFiles := 3
	files := make([]FileInfo, numFiles)
	for i := 0; i < numFiles; i++ {
		name := fmt.Sprintf("file_%03d.jpg", i)
		content := []byte(fmt.Sprintf("content %d padding %s", i, strings.Repeat("x", 500)))
		srcPath := filepath.Join(srcDir, name)
		if err := os.WriteFile(srcPath, content, 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", srcPath, err)
		}
		files[i] = FileInfo{
			SourceName:       name,
			SourceDir:        srcDir,
			DestName:         name,
			DestDir:          destDir,
			Size:             int64(len(content)),
			CreationDateTime: time.Now(),
		}
	}

	// Capture stdout by redirecting to a pipe
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stdout = w

	cfg := config{Verbose: true, Workers: 1}
	copyErr := copyFiles(files, cfg)

	w.Close()
	os.Stdout = oldStdout

	if copyErr != nil {
		t.Fatalf("copyFiles failed: %v", copyErr)
	}

	output, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}

	outputStr := string(output)

	// In non-TTY mode (pipe), output should not contain ANSI escape codes
	if strings.Contains(outputStr, "\033") {
		t.Errorf("Output contains ANSI escape codes in non-TTY mode:\n%s", outputStr)
	}

	// Should still contain file copy lines
	if !strings.Contains(outputStr, "file_000.jpg") {
		t.Errorf("Output should contain file copy lines, got:\n%s", outputStr)
	}

	// Should contain progress info (without ANSI codes)
	if !strings.Contains(outputStr, "remaining") {
		t.Errorf("Output should contain progress info, got:\n%s", outputStr)
	}
}

func TestCopyFilesConcurrent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "concurrent-copy-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create 20 small files with varying sizes
	numFiles := 20
	files := make([]FileInfo, numFiles)
	for i := 0; i < numFiles; i++ {
		name := fmt.Sprintf("file_%03d.jpg", i)
		content := []byte(fmt.Sprintf("content for file %d, padding: %s", i, strings.Repeat("x", i*100)))
		srcPath := filepath.Join(srcDir, name)
		if err := os.WriteFile(srcPath, content, 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", srcPath, err)
		}
		files[i] = FileInfo{
			SourceName:       name,
			SourceDir:        srcDir,
			DestName:         name,
			DestDir:          destDir,
			Size:             int64(len(content)),
			CreationDateTime: time.Now(),
		}
	}

	cfg := config{Workers: 4}
	if err := copyFiles(files, cfg); err != nil {
		t.Fatalf("copyFiles failed: %v", err)
	}

	// Verify all files were copied correctly
	for i := 0; i < numFiles; i++ {
		if files[i].Status != StatusCopied {
			t.Errorf("file %d: expected StatusCopied, got %v", i, files[i].Status)
		}

		destPath := filepath.Join(destDir, files[i].DestName)
		got, err := os.ReadFile(destPath)
		if err != nil {
			t.Errorf("file %d: failed to read dest: %v", i, err)
			continue
		}

		srcPath := filepath.Join(srcDir, files[i].SourceName)
		want, err := os.ReadFile(srcPath)
		if err != nil {
			t.Errorf("file %d: failed to read source: %v", i, err)
			continue
		}

		if string(got) != string(want) {
			t.Errorf("file %d: content mismatch", i)
		}
	}
}
