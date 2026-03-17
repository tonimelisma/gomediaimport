package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"howett.net/plist"
)

// VolumeInfo holds parsed output from diskutil info -plist
type VolumeInfo struct {
	VolumeName                     string `plist:"VolumeName"`
	MountPoint                     string `plist:"MountPoint"`
	FilesystemType                 string `plist:"FilesystemType"`
	Ejectable                      bool   `plist:"Ejectable"`
	RemovableMedia                 bool   `plist:"RemovableMedia"`
	RemovableMediaOrExternalDevice bool   `plist:"RemovableMediaOrExternalDevice"`
	Internal                       bool   `plist:"Internal"`
	DeviceIdentifier               string `plist:"DeviceIdentifier"`
	VolumeUUID                     string `plist:"VolumeUUID"`
}

// diskutilInfoFn is the function type for getting volume info
type diskutilInfoFn func(mountPoint string) (*VolumeInfo, error)

// diskutilInfoReal runs diskutil info -plist and parses the output
func diskutilInfoReal(mountPoint string) (*VolumeInfo, error) {
	cmd := exec.Command("diskutil", "info", "-plist", mountPoint)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("diskutil info failed for %s: %w", mountPoint, err)
	}
	var info VolumeInfo
	_, err = plist.Unmarshal(output, &info)
	if err != nil {
		return nil, fmt.Errorf("failed to parse diskutil output for %s: %w", mountPoint, err)
	}
	return &info, nil
}

// parseDiskutilPlist parses raw plist XML bytes into a VolumeInfo struct
func parseDiskutilPlist(data []byte) (*VolumeInfo, error) {
	var info VolumeInfo
	_, err := plist.Unmarshal(data, &info)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plist data: %w", err)
	}
	return &info, nil
}

// filterVolume determines if a mounted volume should be auto-imported.
// It applies the multi-stage filter pipeline: diskutil properties, DCIM folder, volume allowlist.
func filterVolume(mountPoint string, cfg config, diskutilFn diskutilInfoFn) (bool, error) {
	// Stage 1: diskutil properties
	info, err := diskutilFn(mountPoint)
	if err != nil {
		return false, err
	}
	if !info.Ejectable {
		return false, nil
	}
	if info.Internal && !info.RemovableMediaOrExternalDevice {
		return false, nil
	}

	// Stage 2: DCIM folder check
	if cfg.Watch.RequireDCIM {
		dcimPath := filepath.Join(mountPoint, "DCIM")
		fi, err := os.Stat(dcimPath)
		if err != nil || !fi.IsDir() {
			return false, nil
		}
	}

	// Stage 3: Volume allowlist
	if len(cfg.Watch.Volumes) > 0 {
		volumeName := info.VolumeName
		matched := false
		for _, pattern := range cfg.Watch.Volumes {
			if ok, _ := filepath.Match(pattern, volumeName); ok {
				matched = true
				break
			}
		}
		if !matched {
			return false, nil
		}
	}

	return true, nil
}
