// Package version exposes the build version injected via ldflags.
// main.go copies its -X main.version style flags into these vars at init,
// so existing build tooling (Makefile, GoReleaser) keeps working unchanged.
package version

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)
