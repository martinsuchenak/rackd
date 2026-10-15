//go:build !linux

package storage

// warnIfNetworkFS only detects network filesystems on Linux.
func warnIfNetworkFS(string) {}
