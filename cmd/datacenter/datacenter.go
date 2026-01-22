package datacenter

import "github.com/paularlott/cli"

func Command() *cli.Command {
	return &cli.Command{
		Name:  "datacenter",
		Usage: "Datacenter management commands",
		Commands: []*cli.Command{
			ListCommand(),
			GetCommand(),
			AddCommand(),
			UpdateCommand(),
			DeleteCommand(),
		},
	}
}
