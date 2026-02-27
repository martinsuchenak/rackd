package conflict

import "github.com/paularlott/cli"

func Command() *cli.Command {
	return &cli.Command{
		Name:  "conflict",
		Usage: "IP conflict management commands",
		Commands: []*cli.Command{
			ListCommand(),
			GetCommand(),
			DetectCommand(),
			ResolveCommand(),
			DeleteCommand(),
		},
	}
}
