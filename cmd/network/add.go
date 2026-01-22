package network

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
		Usage: "Add a new network",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Usage: "Network name", Required: true},
			&cli.StringFlag{Name: "subnet", Usage: "Network subnet (CIDR)", Required: true},
			&cli.StringFlag{Name: "description", Usage: "Network description"},
			&cli.IntFlag{Name: "vlan", Usage: "VLAN ID"},
			&cli.StringFlag{Name: "datacenter", Usage: "Datacenter ID"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			network := model.Network{
				Name:         cmd.GetString("name"),
				Subnet:       cmd.GetString("subnet"),
				Description:  cmd.GetString("description"),
				VLANID:       cmd.GetInt("vlan"),
				DatacenterID: cmd.GetString("datacenter"),
			}

			resp, err := c.DoRequest("POST", "/api/networks", network)
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
				fmt.Printf("Network created successfully\n")
				fmt.Printf("ID: %s\n", created["id"])
				fmt.Printf("Name: %s\n", created["name"])
			}
			return nil
		},
	}
}
