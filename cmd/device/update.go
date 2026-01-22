package device

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func UpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a device",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Device ID", Required: true},
			&cli.StringFlag{Name: "name", Usage: "Device name"},
			&cli.StringFlag{Name: "description", Usage: "Device description"},
			&cli.StringFlag{Name: "make-model", Usage: "Device make and model"},
			&cli.StringFlag{Name: "os", Usage: "Operating system"},
			&cli.StringFlag{Name: "datacenter", Usage: "Datacenter ID"},
			&cli.StringFlag{Name: "username", Usage: "Login username"},
			&cli.StringFlag{Name: "location", Usage: "Physical location"},
			&cli.StringFlag{Name: "tags", Usage: "Tags (comma-separated)"},
			&cli.StringFlag{Name: "domains", Usage: "Domain names (comma-separated)"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)
			deviceID := cmd.GetString("id")

			updates := make(map[string]interface{})
			if v := cmd.GetString("name"); v != "" {
				updates["name"] = v
			}
			if v := cmd.GetString("description"); v != "" {
				updates["description"] = v
			}
			if v := cmd.GetString("make-model"); v != "" {
				updates["make_model"] = v
			}
			if v := cmd.GetString("os"); v != "" {
				updates["os"] = v
			}
			if v := cmd.GetString("datacenter"); v != "" {
				updates["datacenter_id"] = v
			}
			if v := cmd.GetString("username"); v != "" {
				updates["username"] = v
			}
			if v := cmd.GetString("location"); v != "" {
				updates["location"] = v
			}
			if v := cmd.GetString("tags"); v != "" {
				updates["tags"] = strings.Split(v, ",")
			}
			if v := cmd.GetString("domains"); v != "" {
				updates["domains"] = strings.Split(v, ",")
			}

			resp, err := c.DoRequest("PUT", "/api/devices/"+deviceID, updates)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var updated map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
				return err
			}

			if cmd.GetString("output") == "json" {
				client.PrintJSON(updated)
			} else {
				fmt.Println("Device updated successfully")
			}
			return nil
		},
	}
}
