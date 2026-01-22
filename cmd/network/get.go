package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func GetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a network by ID",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Network ID", Required: true},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)
			networkID := cmd.GetString("id")

			resp, err := c.DoRequest("GET", "/api/networks/"+networkID, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var network map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&network); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(network)
			case "yaml":
				client.PrintYAML(network)
			default:
				printNetworkDetail(network)
			}
			return nil
		},
	}
}

func printNetworkDetail(n map[string]interface{}) {
	fmt.Printf("ID:          %s\n", getString(n, "id"))
	fmt.Printf("Name:        %s\n", getString(n, "name"))
	fmt.Printf("Description: %s\n", getString(n, "description"))
	fmt.Printf("Subnet:      %s\n", getString(n, "subnet"))
	if vlan, ok := n["vlan_id"].(float64); ok {
		fmt.Printf("VLAN:        %d\n", int(vlan))
	}
	fmt.Printf("Datacenter:  %s\n", getString(n, "datacenter_id"))
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
