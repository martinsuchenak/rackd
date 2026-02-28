package main

import (
	"context"
	"fmt"
	"os"

	"github.com/martinsuchenak/rackd/cmd/apikey"
	"github.com/martinsuchenak/rackd/cmd/audit"
	"github.com/martinsuchenak/rackd/cmd/circuit"
	cmdconflict "github.com/martinsuchenak/rackd/cmd/conflict"
	"github.com/martinsuchenak/rackd/cmd/customfield"
	"github.com/martinsuchenak/rackd/cmd/datacenter"
	"github.com/martinsuchenak/rackd/cmd/device"
	"github.com/martinsuchenak/rackd/cmd/discovery"
	"github.com/martinsuchenak/rackd/cmd/export"
	importcmd "github.com/martinsuchenak/rackd/cmd/import"
	"github.com/martinsuchenak/rackd/cmd/nat"
	"github.com/martinsuchenak/rackd/cmd/network"
	"github.com/martinsuchenak/rackd/cmd/reservation"
	"github.com/martinsuchenak/rackd/cmd/role"
	"github.com/martinsuchenak/rackd/cmd/server"
	"github.com/martinsuchenak/rackd/cmd/user"
	"github.com/martinsuchenak/rackd/cmd/webhook"
	"github.com/paularlott/cli"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	app := &cli.Command{
		Name:    "rackd",
		Usage:   "Device inventory and IPAM management",
		Version: version,
		Commands: []*cli.Command{
			server.Command(),
			device.Command(),
			network.Command(),
			datacenter.Command(),
			discovery.Command(),
			cmdconflict.Command(),
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
			{
				Name:  "version",
				Usage: "Show version information",
				Run: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Printf("Version: %s\nCommit: %s\nBuilt: %s\n", version, commit, date)
					return nil
				},
			},
		},
	}

	if err := app.Execute(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
