package device

import "github.com/paularlott/cli"

func Command() *cli.Command {
	return &cli.Command{
		Name:  "device",
		Usage: "Device management commands",
		Commands: []*cli.Command{
			ListCommand(),
			GetCommand(),
			AddCommand(),
			UpdateCommand(),
			DeleteCommand(),
		},
	}
}
