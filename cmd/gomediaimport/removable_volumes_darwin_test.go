//go:build darwin

package main

import (
	"testing"

	"golang.org/x/sys/unix"
)

// Validates: GMI-064
func TestDarwinIsRemovableVolumeCandidate(t *testing.T) {
	tests := []struct {
		name        string
		mountPath   string
		mountedFrom string
		flags       uint32
		want        bool
	}{
		{
			name:        "explicit removable flag",
			mountPath:   "/Volumes/CAM",
			mountedFrom: "/dev/disk6s1",
			flags:       uint32(unix.MNT_REMOVABLE),
			want:        true,
		},
		{
			name:        "local disk under volumes without removable flag",
			mountPath:   "/Volumes/4152150790",
			mountedFrom: "/dev/disk6s1",
			flags:       uint32(unix.MNT_LOCAL),
			want:        true,
		},
		{
			name:        "network share under volumes",
			mountPath:   "/Volumes/share",
			mountedFrom: "//user@nas.example/share",
			flags:       0,
			want:        false,
		},
		{
			name:        "system data volume",
			mountPath:   "/System/Volumes/Data",
			mountedFrom: "/dev/disk3s1",
			flags:       uint32(unix.MNT_LOCAL),
			want:        false,
		},
		{
			name:        "nested volumes path",
			mountPath:   "/Volumes/CAM/NESTED",
			mountedFrom: "/dev/disk6s1",
			flags:       uint32(unix.MNT_LOCAL),
			want:        false,
		},
		{
			name:        "non-device local mount under volumes",
			mountPath:   "/Volumes/CAM",
			mountedFrom: "map auto_home",
			flags:       uint32(unix.MNT_LOCAL),
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := darwinIsRemovableVolumeCandidate(tt.mountPath, tt.mountedFrom, tt.flags)
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
