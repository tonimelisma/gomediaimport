package main

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tonimelisma/videometa"
)

func testFixturePath(name string) string {
	return filepath.Join("testdata", name)
}

func copyFixtureToTempDir(t *testing.T, dir, srcName, destName string) string {
	t.Helper()

	data, err := os.ReadFile(testFixturePath(srcName))
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", srcName, err)
	}

	destPath := filepath.Join(dir, destName)
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		t.Fatalf("failed to write fixture %s: %v", destName, err)
	}

	return destPath
}

func requireVideoMetadata(t *testing.T, metadata mediaMetadata) *VideoMetadata {
	t.Helper()
	if metadata.VideoMetadata == nil {
		t.Fatal("expected video metadata, got nil")
	}
	return metadata.VideoMetadata
}

func TestExtractVideoMetadataMP4Fixture(t *testing.T) {
	fallbackTime := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)

	metadata, err := extractVideoMetadata(testFixturePath("minimal.mp4"), MP4, fallbackTime)
	if err != nil {
		t.Fatalf("extractVideoMetadata returned error: %v", err)
	}

	vm := requireVideoMetadata(t, metadata)
	wantTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)

	if !metadata.CreationDateTime.Equal(wantTime) {
		t.Fatalf("got creation time %v, want %v", metadata.CreationDateTime, wantTime)
	}
	if !vm.ChosenTimestamp.Equal(wantTime) {
		t.Fatalf("got chosen timestamp %v, want %v", vm.ChosenTimestamp, wantTime)
	}
	if vm.TimestampSource != videoTimestampSourceQuickTime {
		t.Fatalf("got timestamp source %q, want %q", vm.TimestampSource, videoTimestampSourceQuickTime)
	}
	if vm.TimestampTag != "CreateDate" {
		t.Fatalf("got timestamp tag %q, want CreateDate", vm.TimestampTag)
	}
	if vm.TimestampNamespace == "" {
		t.Fatal("expected timestamp namespace to be populated")
	}
	if vm.TimestampFallbackReason != "" {
		t.Fatalf("expected no fallback reason, got %q", vm.TimestampFallbackReason)
	}
	if vm.Width != 320 || vm.Height != 240 {
		t.Fatalf("got dimensions %dx%d, want 320x240", vm.Width, vm.Height)
	}
	if vm.Duration != time.Second {
		t.Fatalf("got duration %v, want %v", vm.Duration, time.Second)
	}
	if vm.Rotation != 0 {
		t.Fatalf("got rotation %d, want 0", vm.Rotation)
	}
	if vm.Codec != "avc1" {
		t.Fatalf("got codec %q, want avc1", vm.Codec)
	}
	if vm.GPSLatitude != nil || vm.GPSLongitude != nil {
		t.Fatal("expected no GPS coordinates on minimal fixture")
	}
	if vm.Make != "" || vm.Model != "" {
		t.Fatalf("expected empty make/model, got %q/%q", vm.Make, vm.Model)
	}
}

func TestExtractVideoMetadataMOVFixture(t *testing.T) {
	fallbackTime := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)

	metadata, err := extractVideoMetadata(testFixturePath("exiftool_quicktime.mov"), MOV, fallbackTime)
	if err != nil {
		t.Fatalf("extractVideoMetadata returned error: %v", err)
	}

	vm := requireVideoMetadata(t, metadata)
	wantTime := time.Date(2005, 8, 11, 14, 3, 54, 0, time.UTC)

	if !metadata.CreationDateTime.Equal(wantTime) {
		t.Fatalf("got creation time %v, want %v", metadata.CreationDateTime, wantTime)
	}
	if vm.TimestampSource != videoTimestampSourceQuickTime {
		t.Fatalf("got timestamp source %q, want %q", vm.TimestampSource, videoTimestampSourceQuickTime)
	}
	if vm.TimestampTag != "CreateDate" {
		t.Fatalf("got timestamp tag %q, want CreateDate", vm.TimestampTag)
	}
	if vm.Width != 320 || vm.Height != 240 {
		t.Fatalf("got dimensions %dx%d, want 320x240", vm.Width, vm.Height)
	}
	if vm.Duration <= 0 {
		t.Fatalf("expected positive duration, got %v", vm.Duration)
	}
	if vm.Codec == "" {
		t.Fatal("expected codec to be populated")
	}
}

func TestExtractVideoMetadataPreservesTimezoneGPSAndCamera(t *testing.T) {
	fallbackTime := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)

	metadata, err := extractVideoMetadata(testFixturePath("with_gps.mp4"), MP4, fallbackTime)
	if err != nil {
		t.Fatalf("extractVideoMetadata returned error: %v", err)
	}

	vm := requireVideoMetadata(t, metadata)

	if got := metadata.CreationDateTime.Format("2006-01-02T15:04:05-07:00"); got != "2024-06-15T10:30:00-07:00" {
		t.Fatalf("got timestamp %q, want %q", got, "2024-06-15T10:30:00-07:00")
	}
	if vm.TimestampSource != videoTimestampSourceQuickTime {
		t.Fatalf("got timestamp source %q, want %q", vm.TimestampSource, videoTimestampSourceQuickTime)
	}
	if vm.TimestampTag != "CreationDate" {
		t.Fatalf("got timestamp tag %q, want CreationDate", vm.TimestampTag)
	}
	if vm.GPSLatitude == nil || vm.GPSLongitude == nil {
		t.Fatal("expected GPS coordinates to be populated")
	}
	if math.Abs(*vm.GPSLatitude-34.0592) > 1e-6 {
		t.Fatalf("got latitude %v, want 34.0592", *vm.GPSLatitude)
	}
	if math.Abs(*vm.GPSLongitude-(-118.446)) > 1e-6 {
		t.Fatalf("got longitude %v, want -118.446", *vm.GPSLongitude)
	}
	if vm.Make != "TestCamera" || vm.Model != "TestModel" {
		t.Fatalf("got make/model %q/%q, want TestCamera/TestModel", vm.Make, vm.Model)
	}
}

func TestResolveVideoTimestampProvenancePrefersQuickTime(t *testing.T) {
	var tags videometa.Tags
	tags.Add(videometa.TagInfo{
		Source:    videometa.VENDOR,
		Tag:       "CreationDate",
		Namespace: "vendor/date",
		Value:     "2025:01:02 03:04:05",
	})
	tags.Add(videometa.TagInfo{
		Source:    videometa.QUICKTIME,
		Tag:       "CreateDate",
		Namespace: "moov/mvhd",
		Value:     "2024:01:02 03:04:05",
	})

	gotTime, source, tag, namespace, found := resolveVideoTimestampProvenance(tags)
	if !found {
		t.Fatal("expected provenance to be found")
	}

	wantTime := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	if !gotTime.Equal(wantTime) {
		t.Fatalf("got time %v, want %v", gotTime, wantTime)
	}
	if source != videoTimestampSourceQuickTime {
		t.Fatalf("got source %q, want %q", source, videoTimestampSourceQuickTime)
	}
	if tag != "CreateDate" || namespace != "moov/mvhd" {
		t.Fatalf("got tag/namespace %q/%q, want CreateDate/moov/mvhd", tag, namespace)
	}

	gotFromLibrary, err := tags.GetDateTime()
	if err != nil {
		t.Fatalf("Tags.GetDateTime returned error: %v", err)
	}
	if !gotFromLibrary.Equal(wantTime) {
		t.Fatalf("Tags.GetDateTime got %v, want %v", gotFromLibrary, wantTime)
	}
}

func TestResolveVideoTimestampProvenanceUsesVendorDateTimeOriginal(t *testing.T) {
	var tags videometa.Tags
	tags.Add(videometa.TagInfo{
		Source:    videometa.VENDOR,
		Tag:       "DateTimeOriginal",
		Namespace: "sony/nrtm",
		Value:     "2024-06-15T10:30:00-07:00",
	})

	gotTime, source, tag, namespace, found := resolveVideoTimestampProvenance(tags)
	if !found {
		t.Fatal("expected provenance to be found")
	}
	if got := gotTime.Format("2006-01-02T15:04:05-07:00"); got != "2024-06-15T10:30:00-07:00" {
		t.Fatalf("got time %q, want %q", got, "2024-06-15T10:30:00-07:00")
	}
	if source != videoTimestampSourceVendor {
		t.Fatalf("got source %q, want %q", source, videoTimestampSourceVendor)
	}
	if tag != "DateTimeOriginal" || namespace != "sony/nrtm" {
		t.Fatalf("got tag/namespace %q/%q, want DateTimeOriginal/sony/nrtm", tag, namespace)
	}
}

func TestExtractVideoMetadataUnsupportedContainerFallsBack(t *testing.T) {
	fallbackTime := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)

	metadata, err := extractVideoMetadata("/does/not/matter.avi", AVI, fallbackTime)
	if err != nil {
		t.Fatalf("extractVideoMetadata returned error: %v", err)
	}

	vm := requireVideoMetadata(t, metadata)
	if !metadata.CreationDateTime.Equal(fallbackTime) {
		t.Fatalf("got creation time %v, want fallback %v", metadata.CreationDateTime, fallbackTime)
	}
	if !vm.ChosenTimestamp.Equal(fallbackTime) {
		t.Fatalf("got chosen timestamp %v, want fallback %v", vm.ChosenTimestamp, fallbackTime)
	}
	if vm.TimestampSource != videoTimestampSourceFallback {
		t.Fatalf("got timestamp source %q, want %q", vm.TimestampSource, videoTimestampSourceFallback)
	}
	if vm.TimestampFallbackReason != videoTimestampFallbackUnsupportedContainer {
		t.Fatalf("got fallback reason %q, want %q", vm.TimestampFallbackReason, videoTimestampFallbackUnsupportedContainer)
	}
}

func TestExtractVideoMetadataCorruptSupportedFileFallsBack(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "corrupt.mp4")
	if err := os.WriteFile(filePath, []byte("not a valid mp4"), 0644); err != nil {
		t.Fatalf("failed to write corrupt file: %v", err)
	}

	fallbackTime := time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)
	metadata, err := extractVideoMetadata(filePath, MP4, fallbackTime)
	if err != nil {
		t.Fatalf("extractVideoMetadata returned error: %v", err)
	}

	vm := requireVideoMetadata(t, metadata)
	if !metadata.CreationDateTime.Equal(fallbackTime) {
		t.Fatalf("got creation time %v, want fallback %v", metadata.CreationDateTime, fallbackTime)
	}
	if vm.TimestampSource != videoTimestampSourceFallback {
		t.Fatalf("got timestamp source %q, want %q", vm.TimestampSource, videoTimestampSourceFallback)
	}
	if vm.TimestampFallbackReason != videoTimestampFallbackDecodeError {
		t.Fatalf("got fallback reason %q, want %q", vm.TimestampFallbackReason, videoTimestampFallbackDecodeError)
	}
	if len(vm.Warnings) == 0 {
		t.Fatal("expected decode warning to be captured")
	}
}

func TestExtractMetadataRawVideoType(t *testing.T) {
	fi := FileInfo{
		SourceDir:     "/tmp",
		SourceName:    "test.braw",
		MediaCategory: RawVideo,
		FileType:      RAWVIDEO,
	}

	_, err := extractMetadata(fi)
	if err == nil {
		t.Fatal("expected error for raw video, got nil")
	}
}

func TestEnumerateFilesAttachesVideoMetadata(t *testing.T) {
	tempDir := t.TempDir()

	copyFixtureToTempDir(t, tempDir, "with_gps.mp4", "clip.mp4")
	if err := os.WriteFile(filepath.Join(tempDir, "clip.thm"), []byte("thm"), 0644); err != nil {
		t.Fatalf("failed to write sidecar: %v", err)
	}

	files, err := enumerateFiles(tempDir, config{SidecarDefault: SidecarDelete})
	if err != nil {
		t.Fatalf("enumerateFiles failed: %v", err)
	}

	var videoFound, sidecarFound bool
	for _, file := range files {
		switch file.SourceName {
		case "clip.mp4":
			videoFound = true
			if file.VideoMetadata == nil {
				t.Fatal("expected video metadata to be attached to clip.mp4")
			}
			if file.VideoMetadata.Make != "TestCamera" || file.VideoMetadata.Model != "TestModel" {
				t.Fatalf("got make/model %q/%q, want TestCamera/TestModel", file.VideoMetadata.Make, file.VideoMetadata.Model)
			}
		case "clip.thm":
			sidecarFound = true
			if file.MediaCategory != Sidecar {
				t.Fatalf("expected clip.thm to be a sidecar, got %v", file.MediaCategory)
			}
			if file.VideoMetadata != nil {
				t.Fatal("expected no video metadata on sidecar file")
			}
		}
	}

	if !videoFound {
		t.Fatal("expected clip.mp4 to be enumerated")
	}
	if !sidecarFound {
		t.Fatal("expected clip.thm to be enumerated as a sidecar")
	}
}

func TestSidecarDefaultsMPLAndCPI(t *testing.T) {
	for _, ext := range []string{"mpl", "cpi"} {
		if !isSidecarExtension(ext) {
			t.Errorf("%s should be a recognized sidecar extension", ext)
		}
		action := getSidecarAction(ext, nil, SidecarIgnore)
		if action != SidecarDelete {
			t.Errorf("expected SidecarDelete for %s, got %v", ext, action)
		}
	}
}
