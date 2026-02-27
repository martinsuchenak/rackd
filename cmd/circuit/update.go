package circuit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func UpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a circuit",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Circuit ID", Required: true},
			&cli.StringFlag{Name: "name", Usage: "Circuit name"},
			&cli.StringFlag{Name: "circuit-id", Usage: "Provider's circuit identifier"},
			&cli.StringFlag{Name: "provider", Usage: "Provider name"},
			&cli.StringFlag{Name: "type", Usage: "Circuit type (fiber, copper, microwave, dark_fiber)"},
			&cli.StringFlag{Name: "status", Usage: "Circuit status"},
			&cli.IntFlag{Name: "capacity", Usage: "Capacity in Mbps"},
			&cli.StringFlag{Name: "datacenter-a", Usage: "Endpoint A datacenter ID"},
			&cli.StringFlag{Name: "datacenter-b", Usage: "Endpoint B datacenter ID"},
			&cli.StringFlag{Name: "device-a", Usage: "Device at endpoint A"},
			&cli.StringFlag{Name: "device-b", Usage: "Device at endpoint B"},
			&cli.StringFlag{Name: "port-a", Usage: "Port at endpoint A"},
			&cli.StringFlag{Name: "port-b", Usage: "Port at endpoint B"},
			&cli.StringFlag{Name: "description", Usage: "Description"},
			&cli.StringFlag{Name: "output", Usage: "Output format (json/yaml)", DefaultValue: "yaml"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			// Build updates map with only provided fields
			updates := make(map[string]interface{})

			if v := cmd.GetString("name"); v != "" {
				updates["name"] = v
			}
			if v := cmd.GetString("circuit-id"); v != "" {
				updates["circuit_id"] = v
			}
			if v := cmd.GetString("provider"); v != "" {
				updates["provider"] = v
			}
			if v := cmd.GetString("type"); v != "" {
				updates["type"] = v
			}
			if v := cmd.GetString("status"); v != "" {
				updates["status"] = v
			}
			if cmd.GetInt("capacity") != 0 {
				updates["capacity_mbps"] = cmd.GetInt("capacity")
			}
			if v := cmd.GetString("datacenter-a"); v != "" {
				updates["datacenter_a_id"] = v
			}
			if v := cmd.GetString("datacenter-b"); v != "" {
				updates["datacenter_b_id"] = v
			}
			if v := cmd.GetString("device-a"); v != "" {
				updates["device_a_id"] = v
			}
			if v := cmd.GetString("device-b"); v != "" {
				updates["device_b_id"] = v
			}
			if v := cmd.GetString("port-a"); v != "" {
				updates["port_a"] = v
			}
			if v := cmd.GetString("port-b"); v != "" {
				updates["port_b"] = v
			}
			if v := cmd.GetString("description"); v != "" {
				updates["description"] = v
			}

			if len(updates) == 0 {
				fmt.Println("No updates specified")
				return nil
			}

			resp, err := c.DoRequest("PUT", "/api/circuits/"+cmd.GetString("id"), updates)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var circuit map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&circuit); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(circuit)
			default:
				client.PrintYAML(circuit)
			}
			return nil
		},
	}
}
