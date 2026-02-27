package circuit

import "github.com/paularlott/cli"

func Command() *cli.Command {
	return &cli.Command{
		Name:  "circuit",
		Usage: "Circuit management commands",
		Commands: []*cli.Command{
			ListCommand(),
			GetCommand(),
			CreateCommand(),
			UpdateCommand(),
			DeleteCommand(),
		},
	}
}
