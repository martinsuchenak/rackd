package dns

import "github.com/paularlott/cli"

func Command() *cli.Command {
	return &cli.Command{
		Name:  "dns",
		Usage: "DNS management commands",
		Commands: []*cli.Command{
			ProviderCommand(),
			ZoneCommand(),
			SyncCommand(),
			ImportCommand(),
			RecordsCommand(),
		},
	}
}
