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
			&cli.BoolFlag{Name: "delete", Usage: "Also delete local records that were synced to the provider but are no longer there"},
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

			resp, err := c.DoRequest("POST", "/api/v1/dns/zones/"+zoneID+"/import", req)
			if err != nil {
				return err
			}
			defer func() { _ = resp.Body.Close() }()

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
			fmt.Printf("Records on provider: %.0f\n", num(result, "total"))
			fmt.Printf("Imported or updated: %.0f (%.0f linked to devices)\n", num(result, "imported"), num(result, "linked"))
			fmt.Printf("Unchanged: %.0f\n", num(result, "skipped"))
			if cmd.GetBool("delete") {
				fmt.Printf("Deleted: %.0f\n", num(result, "deleted"))
			}
			if failed := num(result, "failed"); failed > 0 {
				fmt.Printf("Failed: %.0f\n", failed)
			}
			if msg, ok := result["error"].(string); ok && msg != "" {
				fmt.Printf("Error: %s\n", msg)
			}

			return nil
		},
	}
}
