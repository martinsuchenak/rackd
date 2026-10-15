package storage

import (
	"golang.org/x/sys/unix"

	"github.com/martinsuchenak/rackd/internal/log"
)

// Filesystem magic numbers from statfs(2) for network filesystems.
var networkFS = map[uint32]string{
	0x6969:     "NFS",
	0x517B:     "SMB",
	0xFF534D42: "CIFS",
	0xFE534D42: "SMB2",
}

// warnIfNetworkFS logs a warning when the database's directory is on a
// network filesystem, where SQLite's locking and WAL are slow and unreliable.
func warnIfNetworkFS(dir string) {
	var st unix.Statfs_t
	if err := unix.Statfs(dir, &st); err != nil {
		return
	}
	if kind, ok := networkFS[uint32(st.Type)]; ok {
		log.Warn("The database is on a network filesystem; SQLite is slow there and its locking is unreliable. Use a local disk for the data directory, and back it up to the network share instead (rackd backup).",
			"dir", dir, "filesystem", kind)
	}
}
