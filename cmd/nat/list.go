package nat

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func ListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all NAT mappings",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "external-ip", Usage: "Filter by external IP"},
			&cli.StringFlag{Name: "internal-ip", Usage: "Filter by internal IP"},
			&cli.StringFlag{Name: "protocol", Usage: "Filter by protocol (tcp/udp/any)"},
			&cli.StringFlag{Name: "device", Usage: "Filter by device ID"},
			&cli.StringFlag{Name: "datacenter", Usage: "Filter by datacenter ID"},
			&cli.StringFlag{Name: "output", Usage: "Output format (json/yaml)", DefaultValue: "yaml"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			url := "/api/nat"
			params := ""
			if externalIP := cmd.GetString("external-ip"); externalIP != "" {
				params += "&external_ip=" + externalIP
			}
			if internalIP := cmd.GetString("internal-ip"); internalIP != "" {
				params += "&internal_ip=" + internalIP
			}
			if protocol := cmd.GetString("protocol"); protocol != "" {
				params += "&protocol=" + protocol
			}
			if device := cmd.GetString("device"); device != "" {
				params += "&device_id=" + device
			}
			if datacenter := cmd.GetString("datacenter"); datacenter != "" {
				params += "&datacenter_id=" + datacenter
			}
			if params != "" {
				url += "?" + params[1:]
			}

			resp, err := c.DoRequest("GET", url, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var mappings []map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&mappings); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(mappings)
			default:
				client.PrintYAML(mappings)
			}
			return nil
		},
	}
}
