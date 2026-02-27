package reservation

import "github.com/paularlott/cli"

func Command() *cli.Command {
	return &cli.Command{
		Name:  "reservation",
		Usage: "IP address reservation management commands",
		Commands: []*cli.Command{
			ListCommand(),
			GetCommand(),
			CreateCommand(),
			UpdateCommand(),
			DeleteCommand(),
			ReleaseCommand(),
		},
	}
}
