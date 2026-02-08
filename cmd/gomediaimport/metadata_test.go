package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// buildMinimalMP4 creates a minimal MP4 file containing only a moov box with
// an mvhd box inside. The mvhd box is version 0 (32-bit fields).
// creationTime is in Apple epoch (seconds since 1904-01-01).
func buildMinimalMP4(t *testing.T, dir string, creationTime uint32) string {
	t.Helper()

	// mvhd box (version 0): 108 bytes total
	// 4 bytes size + 4 bytes type + 1 version + 3 flags + 4 creation + 4 modification
	// + 4 timescale + 4 duration + 4 rate + 2 volume + 2 reserved + 8 reserved2
	// + 36 matrix + 24 predefined + 4 next_track_id = 108
	mvhdSize := uint32(108)
	mvhd := make([]byte, mvhdSize)
	binary.BigEndian.PutUint32(mvhd[0:4], mvhdSize)
	copy(mvhd[4:8], "mvhd")
	// version=0, flags=0 (bytes 8-11 are zero)
	binary.BigEndian.PutUint32(mvhd[12:16], creationTime)  // creation_time
	binary.BigEndian.PutUint32(mvhd[16:20], creationTime)  // modification_time
	binary.BigEndian.PutUint32(mvhd[20:24], 1000)          // timescale
	binary.BigEndian.PutUint32(mvhd[24:28], 0)             // duration
	binary.BigEndian.PutUint32(mvhd[28:32], 0x00010000)    // rate = 1.0 (fixed 16.16)
	binary.BigEndian.PutUint16(mvhd[32:34], 0x0100)        // volume = 1.0 (fixed 8.8)
	// bytes 34-42: reserved (zeros)
	// matrix: identity matrix in fixed-point 16.16
	// [0x00010000, 0, 0, 0, 0x00010000, 0, 0, 0, 0x40000000]
	binary.BigEndian.PutUint32(mvhd[42:46], 0x00010000)
	binary.BigEndian.PutUint32(mvhd[58:62], 0x00010000)
	binary.BigEndian.PutUint32(mvhd[74:78], 0x40000000)
	// pre_defined: 24 bytes of zeros (78-102)
	binary.BigEndian.PutUint32(mvhd[102:106], 1) // next_track_id

	// moov box wrapping mvhd
	moovSize := uint32(8 + mvhdSize)
	moov := make([]byte, 8)
	binary.BigEndian.PutUint32(moov[0:4], moovSize)
	copy(moov[4:8], "moov")

	// ftyp box (minimal, required for valid MP4)
	ftyp := make([]byte, 20)
	binary.BigEndian.PutUint32(ftyp[0:4], 20)
	copy(ftyp[4:8], "ftyp")
	copy(ftyp[8:12], "isom")
	binary.BigEndian.PutUint32(ftyp[12:16], 0x200) // minor version
	copy(ftyp[16:20], "isom")

	filePath := filepath.Join(dir, "test.mp4")
	f, err := os.Create(filePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	f.Write(ftyp)
	f.Write(moov)
	f.Write(mvhd)

	return filePath
}

func TestExtractVideoCreationTime_ValidMP4(t *testing.T) {
	dir := t.TempDir()

	// 2024-06-15 12:30:00 UTC in Apple epoch
	// Unix timestamp: 1718451000
	// Apple epoch: 1718451000 + 2082844800 = 3801295800
	wantTime := time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC)
	appleTime := uint32(wantTime.Unix() + appleEpochOffset)

	filePath := buildMinimalMP4(t, dir, appleTime)

	got, err := extractVideoCreationTime(filePath, MP4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(wantTime) {
		t.Errorf("got %v, want %v", got, wantTime)
	}
}

func TestExtractVideoCreationTime_ZeroCreationTime(t *testing.T) {
	dir := t.TempDir()
	filePath := buildMinimalMP4(t, dir, 0)

	_, err := extractVideoCreationTime(filePath, MP4)
	if err == nil {
		t.Fatal("expected error for zero creation time, got nil")
	}
}

func TestExtractVideoCreationTime_NonISOBMFF(t *testing.T) {
	_, err := extractVideoCreationTime("/nonexistent.avi", AVI)
	if err == nil {
		t.Fatal("expected error for non-ISO-BMFF file type, got nil")
	}
}

func TestExtractVideoCreationTime_RawVideoType(t *testing.T) {
	fi := FileInfo{
		SourceDir:     "/tmp",
		SourceName:    "test.braw",
		MediaCategory: RawVideo,
		FileType:      RAWVIDEO,
	}
	_, err := extractCreationDateTimeFromMetadata(fi)
	if err == nil {
		t.Fatal("expected error for raw video, got nil")
	}
}

func TestExtractVideoCreationTime_MOVFileType(t *testing.T) {
	dir := t.TempDir()

	wantTime := time.Date(2023, 3, 10, 8, 0, 0, 0, time.UTC)
	appleTime := uint32(wantTime.Unix() + appleEpochOffset)

	filePath := buildMinimalMP4(t, dir, appleTime)
	// Rename to .mov — same container format
	movPath := filepath.Join(dir, "test.mov")
	os.Rename(filePath, movPath)

	got, err := extractVideoCreationTime(movPath, MOV)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(wantTime) {
		t.Errorf("got %v, want %v", got, wantTime)
	}
}

func TestSidecarDefaults_MPLAndCPI(t *testing.T) {
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
