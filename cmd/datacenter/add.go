package datacenter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/paularlott/cli"
)

func AddCommand() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "Add a new datacenter",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Usage: "Datacenter name", Required: true},
			&cli.StringFlag{Name: "description", Usage: "Datacenter description"},
			&cli.StringFlag{Name: "location", Usage: "Location"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			dc := model.Datacenter{
				Name:        cmd.GetString("name"),
				Description: cmd.GetString("description"),
				Location:    cmd.GetString("location"),
			}

			resp, err := c.DoRequest("POST", "/api/datacenters", dc)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
				return client.HandleError(resp)
			}

			var created map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
				return err
			}

			if cmd.GetString("output") == "json" {
				client.PrintJSON(created)
			} else {
				fmt.Printf("Datacenter created successfully\n")
				fmt.Printf("ID: %s\n", created["id"])
				fmt.Printf("Name: %s\n", created["name"])
			}
			return nil
		},
	}
}
