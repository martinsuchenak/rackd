package conflict

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func ListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all conflicts",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "type", Usage: "Filter by conflict type (duplicate_ip, overlapping_subnet)"},
			&cli.StringFlag{Name: "status", Usage: "Filter by status (active, resolved, ignored)"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			params := url.Values{}
			if t := cmd.GetString("type"); t != "" {
				params.Set("type", t)
			}
			if s := cmd.GetString("status"); s != "" {
				params.Set("status", s)
			}

			path := "/api/conflicts"
			if len(params) > 0 {
				path += "?" + params.Encode()
			}

			resp, err := c.DoRequest("GET", path, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var conflicts []map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&conflicts); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(conflicts)
			case "yaml":
				client.PrintYAML(conflicts)
			default:
				client.PrintConflictTable(conflicts)
			}
			return nil
		},
	}
}
