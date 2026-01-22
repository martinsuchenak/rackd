package discovery

import "github.com/paularlott/cli"

func Command() *cli.Command {
	return &cli.Command{
		Name:  "discovery",
		Usage: "Network discovery commands",
		Commands: []*cli.Command{
			ScanCommand(),
			ListCommand(),
			PromoteCommand(),
		},
	}
}
