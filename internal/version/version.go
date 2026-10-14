// Package version exposes the build version. pkg/app sets it from the
// app.Options that main passes, which main fills from its -X main.version style
// ldflags, so existing build tooling (Taskfile, GoReleaser) keeps working.
package version

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)
