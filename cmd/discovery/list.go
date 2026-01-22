package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func ListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List discovered devices",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "network", Usage: "Filter by network ID"},
			&cli.StringFlag{Name: "status", Usage: "Filter by status (online/offline/unknown)"},
			&cli.IntFlag{Name: "limit", Usage: "Limit number of results"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			params := url.Values{}
			if network := cmd.GetString("network"); network != "" {
				params.Set("network_id", network)
			}
			if status := cmd.GetString("status"); status != "" {
				params.Set("status", status)
			}
			if limit := cmd.GetInt("limit"); limit > 0 {
				params.Set("limit", fmt.Sprintf("%d", limit))
			}

			path := "/api/discovery/devices"
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

			var devices []map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
				return err
			}

			if cmd.GetString("output") == "json" {
				client.PrintJSON(devices)
			} else {
				client.PrintDiscoveredTable(devices)
			}
			return nil
		},
	}
}
