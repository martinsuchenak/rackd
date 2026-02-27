package conflict

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func ResolveCommand() *cli.Command {
	return &cli.Command{
		Name:  "resolve",
		Usage: "Resolve a conflict",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Conflict ID", Required: true},
			&cli.StringFlag{Name: "keep-device-id", Usage: "Device ID to keep (for duplicate_ip conflicts)"},
			&cli.StringFlag{Name: "keep-network-id", Usage: "Network ID to keep (for overlapping_subnet conflicts)"},
			&cli.StringFlag{Name: "notes", Usage: "Optional notes for the resolution"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)
			conflictID := cmd.GetString("id")

			// First, get the conflict to determine its type
			resp, err := c.DoRequest("GET", "/api/conflicts/"+conflictID, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var conflict map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&conflict); err != nil {
				return err
			}

			conflictType := getString(conflict, "type")

			// Build resolution request
			resolution := map[string]interface{}{
				"conflict_id": conflictID,
				"notes":       cmd.GetString("notes"),
			}

			if conflictType == "duplicate_ip" {
				keepDeviceID := cmd.GetString("keep-device-id")
				if keepDeviceID == "" {
					return fmt.Errorf("keep-device-id is required for duplicate_ip conflicts")
				}
				resolution["keep_device_id"] = keepDeviceID
			} else if conflictType == "overlapping_subnet" {
				keepNetworkID := cmd.GetString("keep-network-id")
				if keepNetworkID == "" {
					return fmt.Errorf("keep-network-id is required for overlapping_subnet conflicts")
				}
				resolution["keep_network_id"] = keepNetworkID
			}

			// Resolve the conflict
			resp2, err := c.DoRequest("POST", "/api/conflicts/resolve", resolution)
			if err != nil {
				return err
			}
			defer resp2.Body.Close()

			if resp2.StatusCode != http.StatusOK {
				return client.HandleError(resp2)
			}

			fmt.Printf("Conflict %s resolved successfully\n", conflictID)
			return nil
		},
	}
}
