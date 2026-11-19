package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func ScanCommand() *cli.Command {
	return &cli.Command{
		Name:  "scan",
		Usage: "Start a network discovery scan",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "network", Usage: "Network ID to scan", Required: true},
			&cli.StringFlag{Name: "type", Usage: "Scan type (quick/full/deep); ignored with --profile", DefaultValue: "full"},
			&cli.StringFlag{Name: "profile", Usage: "Scan profile ID; the profile sets the scan type, ports, timeouts and credentials"},
			&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be scanned without scanning"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)
			networkID := cmd.GetString("network")
			scanType := cmd.GetString("type")

			profileID := cmd.GetString("profile")

			if cmd.GetBool("dry-run") {
				fmt.Printf("Network: %s\n", networkID)
				if profileID != "" {
					fmt.Printf("Profile: %s\n", profileID)
					fmt.Printf("This would scan network %s with profile %s\n", networkID, profileID)
				} else {
					fmt.Printf("Scan type: %s\n", scanType)
					fmt.Printf("This would scan network %s with type %s\n", networkID, scanType)
				}
				return nil
			}

			reqBody := map[string]interface{}{
				"scan_type": scanType,
			}
			if profileID != "" {
				reqBody["profile_id"] = profileID
			}

			resp, err := c.DoRequest("POST", "/api/v1/discovery/networks/"+networkID+"/scan", reqBody)
			if err != nil {
				return err
			}
			defer func() { _ = resp.Body.Close() }()

			// The server answers 202 Accepted: the scan runs in the background.
			if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var scan map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&scan); err != nil {
				return err
			}

			fmt.Println("Discovery scan started")
			fmt.Printf("Scan ID: %s\n", scan["id"])
			fmt.Printf("Network: %s\n", scan["network_id"])
			fmt.Printf("Scan type: %s\n", scan["scan_type"])

			return nil
		},
	}
}
