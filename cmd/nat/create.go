package nat

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func CreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new NAT mapping",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Usage: "Name of the NAT mapping", Required: true},
			&cli.StringFlag{Name: "external-ip", Usage: "External IP address", Required: true},
			&cli.IntFlag{Name: "external-port", Usage: "External port", Required: true},
			&cli.StringFlag{Name: "internal-ip", Usage: "Internal IP address", Required: true},
			&cli.IntFlag{Name: "internal-port", Usage: "Internal port", Required: true},
			&cli.StringFlag{Name: "protocol", Usage: "Protocol (tcp/udp/any)", DefaultValue: "tcp"},
			&cli.StringFlag{Name: "device", Usage: "Device ID"},
			&cli.StringFlag{Name: "datacenter", Usage: "Datacenter ID"},
			&cli.StringFlag{Name: "network", Usage: "Network ID"},
			&cli.StringFlag{Name: "description", Usage: "Description"},
			&cli.BoolFlag{Name: "disabled", Usage: "Create as disabled"},
			&cli.StringFlag{Name: "tags", Usage: "Comma-separated tags"},
			&cli.StringFlag{Name: "output", Usage: "Output format (json/yaml)", DefaultValue: "yaml"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			data := map[string]any{
				"name":          cmd.GetString("name"),
				"external_ip":   cmd.GetString("external-ip"),
				"external_port": cmd.GetInt("external-port"),
				"internal_ip":   cmd.GetString("internal-ip"),
				"internal_port": cmd.GetInt("internal-port"),
				"protocol":      cmd.GetString("protocol"),
				"enabled":       !cmd.GetBool("disabled"),
			}

			if device := cmd.GetString("device"); device != "" {
				data["device_id"] = device
			}
			if datacenter := cmd.GetString("datacenter"); datacenter != "" {
				data["datacenter_id"] = datacenter
			}
			if network := cmd.GetString("network"); network != "" {
				data["network_id"] = network
			}
			if description := cmd.GetString("description"); description != "" {
				data["description"] = description
			}
			if tags := cmd.GetString("tags"); tags != "" {
				var tagList []string
				for _, tag := range strings.Split(tags, ",") {
					if t := strings.TrimSpace(tag); t != "" {
						tagList = append(tagList, t)
					}
				}
				data["tags"] = tagList
			}

			resp, err := c.DoRequest("POST", "/api/nat", data)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				return client.HandleError(resp)
			}

			var mapping map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&mapping); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(mapping)
			default:
				client.PrintYAML(mapping)
			}
			return nil
		},
	}
}
