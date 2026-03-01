package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func SyncCommand() *cli.Command {
	return &cli.Command{
		Name:  "sync",
		Usage: "Sync a DNS zone to the provider",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "zone", Usage: "Zone ID", Required: true},
			&cli.BoolFlag{Name: "force", Usage: "Force sync even if unchanged"},
			&cli.StringFlag{Name: "output", Usage: "Output format (json/yaml)", DefaultValue: "yaml"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			zoneID := cmd.GetString("zone")

			req := map[string]interface{}{}
			if cmd.GetBool("force") {
				req["force"] = true
			}

			resp, err := c.DoRequest("POST", "/api/dns/zones/"+zoneID+"/sync", req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(result)
			default:
				client.PrintYAML(result)
			}

			// Also print a human-readable summary
			fmt.Println()
			if status, ok := result["status"].(string); ok {
				fmt.Printf("Status: %s\n", status)
			}
			if recordsCreated, ok := result["records_created"].(float64); ok && recordsCreated > 0 {
				fmt.Printf("Records created: %.0f\n", recordsCreated)
			}
			if recordsUpdated, ok := result["records_updated"].(float64); ok && recordsUpdated > 0 {
				fmt.Printf("Records updated: %.0f\n", recordsUpdated)
			}
			if recordsDeleted, ok := result["records_deleted"].(float64); ok && recordsDeleted > 0 {
				fmt.Printf("Records deleted: %.0f\n", recordsDeleted)
			}
			if unchanged, ok := result["unchanged"].(float64); ok && unchanged > 0 {
				fmt.Printf("Unchanged: %.0f\n", unchanged)
			}

			return nil
		},
	}
}
