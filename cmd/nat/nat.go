package nat

import "github.com/paularlott/cli"

func Command() *cli.Command {
	return &cli.Command{
		Name:  "nat",
		Usage: "NAT mapping management commands",
		Commands: []*cli.Command{
			ListCommand(),
			GetCommand(),
			CreateCommand(),
			UpdateCommand(),
			DeleteCommand(),
		},
	}
}
