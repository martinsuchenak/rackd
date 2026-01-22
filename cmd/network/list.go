package network

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
		Usage: "List all networks",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "datacenter", Usage: "Filter by datacenter ID"},
			&cli.IntFlag{Name: "vlan", Usage: "Filter by VLAN ID"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			params := url.Values{}
			if dc := cmd.GetString("datacenter"); dc != "" {
				params.Set("datacenter_id", dc)
			}
			if vlan := cmd.GetInt("vlan"); vlan > 0 {
				params.Set("vlan_id", fmt.Sprintf("%d", vlan))
			}

			path := "/api/networks"
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

			var networks []map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&networks); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(networks)
			case "yaml":
				client.PrintYAML(networks)
			default:
				client.PrintNetworkTable(networks)
			}
			return nil
		},
	}
}
