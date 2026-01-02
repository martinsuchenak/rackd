package ui

import (
	"embed"
	"io/fs"
)

//go:embed index.html
var Files embed.FS

// GetFS returns the UI filesystem
func GetFS() fs.FS {
	return Files
}
