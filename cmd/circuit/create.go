package circuit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func CreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new circuit",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Usage: "Circuit name", Required: true},
			&cli.StringFlag{Name: "circuit-id", Usage: "Provider's circuit identifier", Required: true},
			&cli.StringFlag{Name: "provider", Usage: "Provider name", Required: true},
			&cli.StringFlag{Name: "type", Usage: "Circuit type (fiber, copper, microwave, dark_fiber)", DefaultValue: "fiber"},
			&cli.StringFlag{Name: "status", Usage: "Circuit status", DefaultValue: "active"},
			&cli.IntFlag{Name: "capacity", Usage: "Capacity in Mbps", DefaultValue: 0},
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

			req := map[string]interface{}{
				"name":         cmd.GetString("name"),
				"circuit_id":   cmd.GetString("circuit-id"),
				"provider":     cmd.GetString("provider"),
				"type":         cmd.GetString("type"),
				"status":       cmd.GetString("status"),
				"capacity_mbps": cmd.GetInt("capacity"),
			}

			if v := cmd.GetString("datacenter-a"); v != "" {
				req["datacenter_a_id"] = v
			}
			if v := cmd.GetString("datacenter-b"); v != "" {
				req["datacenter_b_id"] = v
			}
			if v := cmd.GetString("device-a"); v != "" {
				req["device_a_id"] = v
			}
			if v := cmd.GetString("device-b"); v != "" {
				req["device_b_id"] = v
			}
			if v := cmd.GetString("port-a"); v != "" {
				req["port_a"] = v
			}
			if v := cmd.GetString("port-b"); v != "" {
				req["port_b"] = v
			}
			if v := cmd.GetString("description"); v != "" {
				req["description"] = v
			}

			resp, err := c.DoRequest("POST", "/api/circuits", req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
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
