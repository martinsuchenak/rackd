package device

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
		Usage: "List all devices",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "query", Usage: "Search query"},
			&cli.StringFlag{Name: "tags", Usage: "Filter by tags (comma-separated)"},
			&cli.StringFlag{Name: "datacenter", Usage: "Filter by datacenter ID"},
			&cli.StringFlag{Name: "network", Usage: "Filter by network ID"},
			&cli.StringFlag{Name: "pool", Usage: "Filter by pool ID"},
			&cli.StringFlag{Name: "status", Usage: "Filter by status (planned, active, maintenance, decommissioned)"},
			&cli.IntFlag{Name: "limit", Usage: "Limit number of results"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			params := url.Values{}
			if q := cmd.GetString("query"); q != "" {
				params.Set("q", q)
			}
			if tags := cmd.GetString("tags"); tags != "" {
				params.Set("tags", tags)
			}
			if dc := cmd.GetString("datacenter"); dc != "" {
				params.Set("datacenter_id", dc)
			}
			if net := cmd.GetString("network"); net != "" {
				params.Set("network_id", net)
			}
			if pool := cmd.GetString("pool"); pool != "" {
				params.Set("pool_id", pool)
			}
			if status := cmd.GetString("status"); status != "" {
				params.Set("status", status)
			}
			if limit := cmd.GetInt("limit"); limit > 0 {
				params.Set("limit", fmt.Sprintf("%d", limit))
			}

			path := "/api/devices"
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

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(devices)
			case "yaml":
				client.PrintYAML(devices)
			default:
				client.PrintDeviceTable(devices)
			}
			return nil
		},
	}
}
