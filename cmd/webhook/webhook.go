package webhook

import "github.com/paularlott/cli"

func Command() *cli.Command {
	return &cli.Command{
		Name:  "webhook",
		Usage: "Webhook management commands",
		Commands: []*cli.Command{
			ListCommand(),
			GetCommand(),
			CreateCommand(),
			UpdateCommand(),
			DeleteCommand(),
			PingCommand(),
			EventsCommand(),
		},
	}
}
