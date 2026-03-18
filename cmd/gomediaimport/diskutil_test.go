package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleDiskutilPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>VolumeName</key>
	<string>EOS_DIGITAL</string>
	<key>MountPoint</key>
	<string>/Volumes/EOS_DIGITAL</string>
	<key>FilesystemType</key>
	<string>msdos</string>
	<key>Ejectable</key>
	<true/>
	<key>RemovableMedia</key>
	<true/>
	<key>RemovableMediaOrExternalDevice</key>
	<true/>
	<key>Internal</key>
	<false/>
	<key>DeviceIdentifier</key>
	<string>disk4s1</string>
	<key>VolumeUUID</key>
	<string>ABCD-1234</string>
</dict>
</plist>`

func TestDiskutilInfoParsing(t *testing.T) {
	info, err := parseDiskutilPlist([]byte(sampleDiskutilPlist))
	if err != nil {
		t.Fatalf("parseDiskutilPlist failed: %v", err)
	}
	if info.VolumeName != "EOS_DIGITAL" {
		t.Errorf("expected VolumeName=EOS_DIGITAL, got %s", info.VolumeName)
	}
	if !info.Ejectable {
		t.Error("expected Ejectable=true")
	}
	if info.Internal {
		t.Error("expected Internal=false")
	}
	if !info.RemovableMediaOrExternalDevice {
		t.Error("expected RemovableMediaOrExternalDevice=true")
	}
	if info.MountPoint != "/Volumes/EOS_DIGITAL" {
		t.Errorf("expected MountPoint=/Volumes/EOS_DIGITAL, got %s", info.MountPoint)
	}
	if info.DeviceIdentifier != "disk4s1" {
		t.Errorf("expected DeviceIdentifier=disk4s1, got %s", info.DeviceIdentifier)
	}
}

func TestFilterVolumeRejectsNonEjectable(t *testing.T) {
	mockFn := func(mountPoint string) (*VolumeInfo, error) {
		return &VolumeInfo{
			VolumeName: "Macintosh HD",
			Ejectable:  false,
			Internal:   true,
		}, nil
	}

	cfg := config{Watch: WatchConfig{RequireDCIM: false}}
	pass, err := filterVolume("/Volumes/Macintosh HD", cfg, mockFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pass {
		t.Error("expected non-ejectable volume to be rejected")
	}
}

func TestFilterVolumeRejectsInternal(t *testing.T) {
	mockFn := func(mountPoint string) (*VolumeInfo, error) {
		return &VolumeInfo{
			VolumeName:                     "InternalDrive",
			Ejectable:                      true,
			Internal:                       true,
			RemovableMediaOrExternalDevice: false,
		}, nil
	}

	cfg := config{Watch: WatchConfig{RequireDCIM: false}}
	pass, err := filterVolume("/Volumes/InternalDrive", cfg, mockFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pass {
		t.Error("expected internal non-removable volume to be rejected")
	}
}

func TestFilterVolumeAcceptsEjectableInternal(t *testing.T) {
	mockFn := func(mountPoint string) (*VolumeInfo, error) {
		return &VolumeInfo{
			VolumeName:                     "SD_CARD",
			Ejectable:                      true,
			Internal:                       true,
			RemovableMediaOrExternalDevice: true,
		}, nil
	}

	cfg := config{Watch: WatchConfig{RequireDCIM: false}}
	pass, err := filterVolume("/Volumes/SD_CARD", cfg, mockFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !pass {
		t.Error("expected ejectable internal+removable volume to be accepted (built-in SD reader)")
	}
}

func TestFilterVolumeRejectsDiskutilFailure(t *testing.T) {
	mockFn := func(mountPoint string) (*VolumeInfo, error) {
		return nil, fmt.Errorf("diskutil failed")
	}

	cfg := config{Watch: WatchConfig{RequireDCIM: false}}
	pass, err := filterVolume("/Volumes/SomeVolume", cfg, mockFn)
	if err == nil {
		t.Error("expected error from diskutil failure")
	}
	if pass {
		t.Error("expected volume to be rejected on diskutil failure")
	}
}

func TestFilterVolumeRejectsNoDCIM(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filter-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mockFn := func(mountPoint string) (*VolumeInfo, error) {
		return &VolumeInfo{
			VolumeName:                     "CAMERA",
			Ejectable:                      true,
			Internal:                       false,
			RemovableMediaOrExternalDevice: true,
		}, nil
	}

	cfg := config{Watch: WatchConfig{RequireDCIM: true}}
	pass, err := filterVolume(tmpDir, cfg, mockFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pass {
		t.Error("expected volume without DCIM to be rejected")
	}
}

func TestFilterVolumeAcceptsNoDCIMWhenDisabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filter-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	mockFn := func(mountPoint string) (*VolumeInfo, error) {
		return &VolumeInfo{
			VolumeName:                     "CAMERA",
			Ejectable:                      true,
			Internal:                       false,
			RemovableMediaOrExternalDevice: true,
		}, nil
	}

	cfg := config{Watch: WatchConfig{RequireDCIM: false}}
	pass, err := filterVolume(tmpDir, cfg, mockFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !pass {
		t.Error("expected volume without DCIM to be accepted when DCIM check disabled")
	}
}

func TestFilterVolumeWithDCIM(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filter-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	if err := os.Mkdir(filepath.Join(tmpDir, "DCIM"), 0755); err != nil {
		t.Fatal(err)
	}

	mockFn := func(mountPoint string) (*VolumeInfo, error) {
		return &VolumeInfo{
			VolumeName:                     "CAMERA",
			Ejectable:                      true,
			Internal:                       false,
			RemovableMediaOrExternalDevice: true,
		}, nil
	}

	cfg := config{Watch: WatchConfig{RequireDCIM: true}}
	pass, err := filterVolume(tmpDir, cfg, mockFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !pass {
		t.Error("expected volume with DCIM to be accepted")
	}
}

func TestFilterVolumeAllowlistMatch(t *testing.T) {
	mockFn := func(mountPoint string) (*VolumeInfo, error) {
		return &VolumeInfo{
			VolumeName:                     "EOS_DIGITAL",
			Ejectable:                      true,
			Internal:                       false,
			RemovableMediaOrExternalDevice: true,
		}, nil
	}

	cfg := config{
		Watch: WatchConfig{
			RequireDCIM: false,
			Volumes:     []string{"EOS_*", "NIKON*"},
		},
	}
	pass, err := filterVolume("/Volumes/EOS_DIGITAL", cfg, mockFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !pass {
		t.Error("expected matching volume name to be accepted")
	}
}

func TestFilterVolumeAllowlistReject(t *testing.T) {
	mockFn := func(mountPoint string) (*VolumeInfo, error) {
		return &VolumeInfo{
			VolumeName:                     "USB_DRIVE",
			Ejectable:                      true,
			Internal:                       false,
			RemovableMediaOrExternalDevice: true,
		}, nil
	}

	cfg := config{
		Watch: WatchConfig{
			RequireDCIM: false,
			Volumes:     []string{"EOS_*", "NIKON*"},
		},
	}
	pass, err := filterVolume("/Volumes/USB_DRIVE", cfg, mockFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pass {
		t.Error("expected non-matching volume name to be rejected")
	}
}

func TestFilterVolumeEmptyAllowlistAcceptsAll(t *testing.T) {
	mockFn := func(mountPoint string) (*VolumeInfo, error) {
		return &VolumeInfo{
			VolumeName:                     "ANYTHING",
			Ejectable:                      true,
			Internal:                       false,
			RemovableMediaOrExternalDevice: true,
		}, nil
	}

	cfg := config{Watch: WatchConfig{RequireDCIM: false}}
	pass, err := filterVolume("/Volumes/ANYTHING", cfg, mockFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !pass {
		t.Error("expected all volumes to pass when allowlist is empty")
	}
}

func TestFilterVolumeVerboseLogging(t *testing.T) {
	mockFn := func(mountPoint string) (*VolumeInfo, error) {
		return &VolumeInfo{
			VolumeName: "Macintosh HD",
			Ejectable:  false,
			Internal:   true,
		}, nil
	}

	cfg := config{
		Verbose: true,
		Watch:   WatchConfig{RequireDCIM: false},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	pass, filterErr := filterVolume("/Volumes/Macintosh HD", cfg, mockFn)

	w.Close()
	os.Stdout = oldStdout

	if filterErr != nil {
		t.Fatalf("unexpected error: %v", filterErr)
	}
	if pass {
		t.Error("expected volume to be rejected")
	}

	output := make([]byte, 4096)
	n, _ := r.Read(output)
	outputStr := string(output[:n])

	if !strings.Contains(outputStr, "rejected") {
		t.Errorf("expected verbose output to contain 'rejected', got:\n%s", outputStr)
	}
	if !strings.Contains(outputStr, "not ejectable") {
		t.Errorf("expected verbose output to contain 'not ejectable', got:\n%s", outputStr)
	}
}

func TestFilterVolumeGlobPatterns(t *testing.T) {
	tests := []struct {
		name       string
		volumeName string
		patterns   []string
		wantPass   bool
	}{
		{"NIKON glob", "NIKON D850", []string{"NIKON*"}, true},
		{"EOS glob", "EOS_DIGITAL", []string{"EOS_*"}, true},
		{"no match", "SONY_CARD", []string{"EOS_*", "NIKON*"}, false},
		{"exact match", "EOS_DIGITAL", []string{"EOS_DIGITAL"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFn := func(mountPoint string) (*VolumeInfo, error) {
				return &VolumeInfo{
					VolumeName:                     tt.volumeName,
					Ejectable:                      true,
					Internal:                       false,
					RemovableMediaOrExternalDevice: true,
				}, nil
			}

			cfg := config{
				Watch: WatchConfig{
					RequireDCIM: false,
					Volumes:     tt.patterns,
				},
			}
			pass, err := filterVolume("/Volumes/test", cfg, mockFn)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pass != tt.wantPass {
				t.Errorf("volume %q with patterns %v: got pass=%v, want %v",
					tt.volumeName, tt.patterns, pass, tt.wantPass)
			}
		})
	}
}
