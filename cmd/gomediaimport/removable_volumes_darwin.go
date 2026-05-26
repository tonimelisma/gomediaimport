//go:build darwin

package main

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

func listMountedRemovableVolumes() ([]mountedRemovableVolume, error) {
	n, err := unix.Getfsstat(nil, unix.MNT_NOWAIT)
	if err != nil {
		return nil, fmt.Errorf("failed to list mounted filesystems: %w", err)
	}
	stats := make([]unix.Statfs_t, n)
	n, err = unix.Getfsstat(stats, unix.MNT_NOWAIT)
	if err != nil {
		return nil, fmt.Errorf("failed to list mounted filesystems: %w", err)
	}

	var volumes []mountedRemovableVolume
	for _, stat := range stats[:n] {
		if stat.Flags&unix.MNT_REMOVABLE == 0 {
			continue
		}
		mountPath := nullTerminatedBytesToString(stat.Mntonname[:])
		if mountPath == "" {
			continue
		}
		label, err := darwinVolumeLabel(mountPath)
		if err != nil || label == "" {
			label = filepath.Base(mountPath)
		}
		volumes = append(volumes, mountedRemovableVolume{
			Label:     label,
			MountPath: mountPath,
		})
	}
	return volumes, nil
}

func darwinVolumeLabel(mountPath string) (string, error) {
	path, err := unix.BytePtrFromString(mountPath)
	if err != nil {
		return "", err
	}

	attrList := unix.Attrlist{
		Bitmapcount: unix.ATTR_BIT_MAP_COUNT,
		Volattr:     unix.ATTR_VOL_INFO | unix.ATTR_VOL_NAME,
	}
	buf := make([]byte, 4096)
	_, _, errno := unix.Syscall6(
		unix.SYS_GETATTRLIST,
		uintptr(unsafe.Pointer(path)),
		uintptr(unsafe.Pointer(&attrList)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		0,
		0,
	)
	if errno != 0 {
		return "", errno
	}
	if len(buf) < 12 {
		return "", fmt.Errorf("getattrlist buffer too small")
	}

	refStart := 4
	dataOffset := int(int32(binary.LittleEndian.Uint32(buf[refStart : refStart+4])))
	dataLength := int(binary.LittleEndian.Uint32(buf[refStart+4 : refStart+8]))
	if dataLength <= 0 {
		return "", fmt.Errorf("volume label is empty")
	}

	start := refStart + dataOffset
	end := start + dataLength
	if start < 0 || end > len(buf) {
		start = dataOffset
		end = start + dataLength
	}
	if start < 0 || end > len(buf) || start >= end {
		return "", fmt.Errorf("invalid volume label reference")
	}

	return strings.TrimRight(string(buf[start:end]), "\x00"), nil
}

func nullTerminatedBytesToString(buf []byte) string {
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i])
		}
	}
	return string(buf)
}
