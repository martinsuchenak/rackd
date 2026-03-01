package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func ImportCommand() *cli.Command {
	return &cli.Command{
		Name:  "import",
		Usage: "Import DNS records from provider",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "zone", Usage: "Zone ID", Required: true},
			&cli.BoolFlag{Name: "delete", Usage: "Delete local records not found on provider"},
			&cli.StringFlag{Name: "output", Usage: "Output format (json/yaml)", DefaultValue: "yaml"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			zoneID := cmd.GetString("zone")

			req := map[string]interface{}{}
			if cmd.GetBool("delete") {
				req["delete_missing"] = true
			}

			resp, err := c.DoRequest("POST", "/api/dns/zones/"+zoneID+"/import", req)
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
			if recordsImported, ok := result["records_imported"].(float64); ok {
				fmt.Printf("Records imported: %.0f\n", recordsImported)
			}
			if recordsUpdated, ok := result["records_updated"].(float64); ok {
				fmt.Printf("Records updated: %.0f\n", recordsUpdated)
			}
			if recordsSkipped, ok := result["records_skipped"].(float64); ok {
				fmt.Printf("Records skipped: %.0f\n", recordsSkipped)
			}
			if recordsDeleted, ok := result["records_deleted"].(float64); ok {
				fmt.Printf("Records deleted: %.0f\n", recordsDeleted)
			}

			return nil
		},
	}
}
