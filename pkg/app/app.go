// Package app builds a Rackd binary: the CLI with all of core's commands and
// the server, plus any extensions compiled in. Rackd's own main is
//
//	app.Main(app.Options{Version: version, Commit: commit, Date: date})
//
// and a binary with extensions passes them in Options.Features.
package app

import (
	"context"
	"fmt"
	"io/fs"
	"os"

	"github.com/martinsuchenak/rackd/cmd/apikey"
	"github.com/martinsuchenak/rackd/cmd/audit"
	"github.com/martinsuchenak/rackd/cmd/backup"
	"github.com/martinsuchenak/rackd/cmd/circuit"
	cmdconflict "github.com/martinsuchenak/rackd/cmd/conflict"
	"github.com/martinsuchenak/rackd/cmd/credential"
	"github.com/martinsuchenak/rackd/cmd/customfield"
	"github.com/martinsuchenak/rackd/cmd/datacenter"
	"github.com/martinsuchenak/rackd/cmd/device"
	"github.com/martinsuchenak/rackd/cmd/discovery"
	"github.com/martinsuchenak/rackd/cmd/dns"
	"github.com/martinsuchenak/rackd/cmd/export"
	importcmd "github.com/martinsuchenak/rackd/cmd/import"
	"github.com/martinsuchenak/rackd/cmd/migrate"
	"github.com/martinsuchenak/rackd/cmd/nat"
	"github.com/martinsuchenak/rackd/cmd/network"
	"github.com/martinsuchenak/rackd/cmd/oauth"
	"github.com/martinsuchenak/rackd/cmd/reservation"
	"github.com/martinsuchenak/rackd/cmd/role"
	"github.com/martinsuchenak/rackd/cmd/scanprofile"
	"github.com/martinsuchenak/rackd/cmd/scheduledscan"
	"github.com/martinsuchenak/rackd/cmd/server"
	"github.com/martinsuchenak/rackd/cmd/user"
	"github.com/martinsuchenak/rackd/cmd/webhook"
	"github.com/martinsuchenak/rackd/internal/extensions"
	buildversion "github.com/martinsuchenak/rackd/internal/version"
	"github.com/martinsuchenak/rackd/pkg/extension"
	"github.com/paularlott/cli"
)

// Options describes the binary.
type Options struct {
	// Name and Usage head the CLI help; they default to Rackd's.
	Name  string
	Usage string
	// Version, Commit and Date are the build information shown by the version
	// command and /api/v1/config.
	Version string
	Commit  string
	Date    string
	// Edition is reported in /api/v1/config; empty means "oss".
	Edition string
	// Features are extensions to run in addition to Rackd's built-in ones.
	Features []extension.Feature
	// UI replaces the built-in web UI, for a binary that builds its own; nil
	// serves the built-in one. It must hold index.html at its root. To add
	// pages to the built-in UI instead, use extension.UI.
	UI fs.FS
}

// Main runs the CLI and exits with status 1 on an error.
func Main(opts Options) {
	cmd, err := Command(opts)
	if err == nil {
		err = cmd.Execute(context.Background())
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Command builds the CLI. It fails if two extensions, or an extension and
// core, add a command with the same name.
func Command(opts Options) (*cli.Command, error) {
	if opts.Name == "" {
		opts.Name = "rackd"
	}
	if opts.Usage == "" {
		opts.Usage = "Device inventory and IPAM management"
	}
	if opts.Version == "" {
		opts.Version = "dev"
	}
	buildversion.Version, buildversion.Commit, buildversion.Date = opts.Version, opts.Commit, opts.Date

	features := append(builtinFeatures(), opts.Features...)
	// Validate up front so every command, not just the server, refuses
	// malformed or clashing features.
	if err := extensions.Validate(features); err != nil {
		return nil, err
	}
	if opts.UI != nil {
		if _, err := fs.Stat(opts.UI, "index.html"); err != nil {
			return nil, fmt.Errorf("app: UI: %w", err)
		}
	}
	commands := []*cli.Command{
		server.NewCommand(features, opts.Edition, opts.UI),
		device.Command(),
		network.Command(),
		datacenter.Command(),
		discovery.Command(),
		cmdconflict.Command(),
		credential.Command(),
		circuit.Command(),
		nat.Command(),
		reservation.Command(),
		webhook.Command(),
		customfield.Command(),
		apikey.Command(),
		user.Command(),
		role.Command(),
		audit.Command(),
		export.Command(),
		importcmd.Command(),
		scanprofile.Command(),
		scheduledscan.Command(),
		oauth.Command(),
		backup.Command(),
		dns.Command(),
		migrate.NewCommand(features),
		{
			Name:  "version",
			Usage: "Show version information",
			Run: func(ctx context.Context, cmd *cli.Command) error {
				fmt.Printf("Version: %s\nCommit: %s\nBuilt: %s\n", opts.Version, opts.Commit, opts.Date)
				return nil
			},
		},
	}

	names := map[string]string{}
	for _, c := range commands {
		names[c.Name] = "core"
	}
	for _, f := range features {
		cp, ok := f.(extension.CommandProvider)
		if !ok {
			continue
		}
		name := f.Manifest().Name
		for _, c := range cp.Commands() {
			if owner, taken := names[c.Name]; taken {
				return nil, fmt.Errorf("extension %s: command %q is already provided by %s", name, c.Name, owner)
			}
			names[c.Name] = name
			commands = append(commands, c)
		}
	}

	return &cli.Command{
		Name:     opts.Name,
		Usage:    opts.Usage,
		Version:  opts.Version,
		Commands: commands,
	}, nil
}

// builtinFeatures are the parts of Rackd built on the extension API.
func builtinFeatures() []extension.Feature {
	return nil
}
