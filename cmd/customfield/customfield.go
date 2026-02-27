package customfield

import "github.com/paularlott/cli"

func Command() *cli.Command {
	return &cli.Command{
		Name:  "custom-field",
		Usage: "Custom field management commands",
		Commands: []*cli.Command{
			ListCommand(),
			GetCommand(),
			CreateCommand(),
			UpdateCommand(),
			DeleteCommand(),
			TypesCommand(),
		},
	}
}
