package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func PromoteCommand() *cli.Command {
	return &cli.Command{
		Name:  "promote",
		Usage: "Promote a discovered device to inventory",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "discovered-id", Usage: "Discovered device ID", Required: true},
			&cli.StringFlag{Name: "name", Usage: "Device name", Required: true},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)
			discoveredID := cmd.GetString("discovered-id")
			name := cmd.GetString("name")

			reqBody := map[string]interface{}{
				"name": name,
			}

			resp, err := c.DoRequest("POST", "/api/discovery/devices/"+discoveredID+"/promote", reqBody)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var device map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&device); err != nil {
				return err
			}

			fmt.Println("Device promoted successfully")
			fmt.Printf("Device ID: %s\n", device["id"])
			fmt.Printf("Name: %s\n", device["name"])

			return nil
		},
	}
}
