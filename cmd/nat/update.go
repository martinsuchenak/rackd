package nat

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func UpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a NAT mapping",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "NAT mapping ID", Required: true},
			&cli.StringFlag{Name: "name", Usage: "Name of the NAT mapping"},
			&cli.StringFlag{Name: "external-ip", Usage: "External IP address"},
			&cli.IntFlag{Name: "external-port", Usage: "External port"},
			&cli.StringFlag{Name: "internal-ip", Usage: "Internal IP address"},
			&cli.IntFlag{Name: "internal-port", Usage: "Internal port"},
			&cli.StringFlag{Name: "protocol", Usage: "Protocol (tcp/udp/any)"},
			&cli.StringFlag{Name: "device", Usage: "Device ID"},
			&cli.StringFlag{Name: "datacenter", Usage: "Datacenter ID"},
			&cli.StringFlag{Name: "network", Usage: "Network ID"},
			&cli.StringFlag{Name: "description", Usage: "Description"},
			&cli.BoolFlag{Name: "enabled", Usage: "Enable the mapping"},
			&cli.BoolFlag{Name: "disabled", Usage: "Disable the mapping"},
			&cli.StringFlag{Name: "tags", Usage: "Comma-separated tags"},
			&cli.StringFlag{Name: "output", Usage: "Output format (json/yaml)", DefaultValue: "yaml"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			data := make(map[string]any)

			if name := cmd.GetString("name"); name != "" {
				data["name"] = name
			}
			if externalIP := cmd.GetString("external-ip"); externalIP != "" {
				data["external_ip"] = externalIP
			}
			if externalPort := cmd.GetInt("external-port"); externalPort > 0 {
				data["external_port"] = externalPort
			}
			if internalIP := cmd.GetString("internal-ip"); internalIP != "" {
				data["internal_ip"] = internalIP
			}
			if internalPort := cmd.GetInt("internal-port"); internalPort > 0 {
				data["internal_port"] = internalPort
			}
			if protocol := cmd.GetString("protocol"); protocol != "" {
				data["protocol"] = protocol
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
			if cmd.GetBool("enabled") {
				data["enabled"] = true
			} else if cmd.GetBool("disabled") {
				data["enabled"] = false
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

			id := cmd.GetString("id")
			resp, err := c.DoRequest("PUT", "/api/nat/"+id, data)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
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
