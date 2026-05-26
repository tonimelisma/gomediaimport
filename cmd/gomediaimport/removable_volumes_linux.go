//go:build linux

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

func listMountedRemovableVolumes() ([]mountedRemovableVolume, error) {
	labelByDev, err := linuxLabelsByDevice()
	if err != nil {
		return nil, err
	}

	file, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to open mountinfo: %w", err)
	}
	defer func() { _ = file.Close() }()

	var volumes []mountedRemovableVolume
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		entry, ok := parseLinuxMountInfoLine(scanner.Text())
		if !ok {
			continue
		}
		sysPath, err := filepath.EvalSymlinks(filepath.Join("/sys/dev/block", entry.majorMinor))
		if err != nil {
			continue
		}
		if !linuxDeviceIsRemovable(sysPath) {
			continue
		}

		volumes = append(volumes, mountedRemovableVolume{
			Label:     labelByDev[entry.majorMinor],
			MountPath: entry.mountPoint,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read mountinfo: %w", err)
	}

	return volumes, nil
}

type linuxMountInfoEntry struct {
	majorMinor string
	mountPoint string
}

func parseLinuxMountInfoLine(line string) (linuxMountInfoEntry, bool) {
	fields := strings.Fields(line)
	if len(fields) < 10 {
		return linuxMountInfoEntry{}, false
	}
	separator := -1
	for i, field := range fields {
		if field == "-" {
			separator = i
			break
		}
	}
	if separator < 0 || separator+2 >= len(fields) {
		return linuxMountInfoEntry{}, false
	}
	return linuxMountInfoEntry{
		majorMinor: fields[2],
		mountPoint: linuxUnescapeMountField(fields[4]),
	}, true
}

func linuxDeviceIsRemovable(sysPath string) bool {
	for {
		data, err := os.ReadFile(filepath.Join(sysPath, "removable"))
		if err == nil && strings.TrimSpace(string(data)) == "1" {
			return true
		}

		parent := filepath.Dir(sysPath)
		if parent == sysPath || parent == "/" {
			return false
		}
		sysPath = parent
	}
}

func linuxLabelsByDevice() (map[string]string, error) {
	labels := make(map[string]string)
	entries, err := os.ReadDir("/dev/disk/by-label")
	if err != nil {
		if os.IsNotExist(err) {
			return labels, nil
		}
		return nil, fmt.Errorf("failed to read /dev/disk/by-label: %w", err)
	}

	for _, entry := range entries {
		labelPath := filepath.Join("/dev/disk/by-label", entry.Name())
		var stat unix.Stat_t
		if err := unix.Stat(labelPath, &stat); err != nil {
			continue
		}
		major := unix.Major(stat.Rdev)
		minor := unix.Minor(stat.Rdev)
		labels[fmt.Sprintf("%d:%d", major, minor)] = linuxUnescapeUdevLabel(entry.Name())
	}

	return labels, nil
}

func linuxUnescapeMountField(value string) string {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' && i+3 < len(value) {
			if decoded, err := strconv.ParseUint(value[i+1:i+4], 8, 8); err == nil {
				b.WriteByte(byte(decoded))
				i += 3
				continue
			}
		}
		b.WriteByte(value[i])
	}
	return b.String()
}

func linuxUnescapeUdevLabel(value string) string {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' && i+3 < len(value) && value[i+1] == 'x' {
			if decoded, err := strconv.ParseUint(value[i+2:i+4], 16, 8); err == nil {
				b.WriteByte(byte(decoded))
				i += 3
				continue
			}
		}
		b.WriteByte(value[i])
	}
	return b.String()
}
