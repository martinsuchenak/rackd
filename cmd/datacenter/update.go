package datacenter

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
		Usage: "Update a datacenter",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Datacenter ID", Required: true},
			&cli.StringFlag{Name: "name", Usage: "Datacenter name"},
			&cli.StringFlag{Name: "description", Usage: "Datacenter description"},
			&cli.StringFlag{Name: "location", Usage: "Location"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)
			dcID := cmd.GetString("id")

			updates := make(map[string]interface{})
			if v := cmd.GetString("name"); v != "" {
				updates["name"] = v
			}
			if v := cmd.GetString("description"); v != "" {
				updates["description"] = v
			}
			if v := cmd.GetString("location"); v != "" {
				updates["location"] = v
			}

			resp, err := c.DoRequest("PUT", "/api/datacenters/"+dcID, updates)
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
				fmt.Println("Datacenter updated successfully")
			}
			return nil
		},
	}
}
