package main

import "github.com/martinsuchenak/rackd/pkg/app"

// Set by -ldflags "-X main.version=... -X main.commit=... -X main.date=...".
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	app.Main(app.Options{Version: version, Commit: commit, Date: date})
}
