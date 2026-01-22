package network

import "github.com/paularlott/cli"

func Command() *cli.Command {
	return &cli.Command{
		Name:  "network",
		Usage: "Network management commands",
		Commands: []*cli.Command{
			ListCommand(),
			GetCommand(),
			AddCommand(),
			DeleteCommand(),
			PoolCommand(),
		},
	}
}
