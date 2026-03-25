//go:build darwin || linux

package main

import "syscall"

// availableDiskSpaceForDir returns the available bytes for non-root users
// on the filesystem containing dir using syscall.Statfs.
func availableDiskSpaceForDir(dir string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return 0, err
	}
	// Bavail = blocks available to unprivileged users
	return stat.Bavail * uint64(stat.Bsize), nil
}
