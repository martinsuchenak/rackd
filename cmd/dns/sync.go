package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func SyncCommand() *cli.Command {
	return &cli.Command{
		Name:  "sync",
		Usage: "Sync a DNS zone to the provider",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "zone", Usage: "Zone ID", Required: true},
			&cli.StringFlag{Name: "output", Usage: "Output format (json/yaml)", DefaultValue: "yaml"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			zoneID := cmd.GetString("zone")

			// Every record is checked against the provider and only written
			// when its value differs, so there is nothing to force.
			resp, err := c.DoRequest("POST", "/api/v1/dns/zones/"+zoneID+"/sync", nil)
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

			// A human-readable summary, on stderr so --output json stays parseable
			fmt.Fprintln(os.Stderr)
			fmt.Fprintf(os.Stderr, "Synced: %.0f of %.0f\n", num(result, "synced"), num(result, "total"))
			if failed := num(result, "failed"); failed > 0 {
				fmt.Fprintf(os.Stderr, "Failed: %.0f\n", failed)
			}
			if msg, ok := result["error"].(string); ok && msg != "" {
				fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
			}

			return nil
		},
	}
}

// num reads a numeric field from a decoded JSON object, or 0.
func num(m map[string]interface{}, key string) float64 {
	v, _ := m[key].(float64)
	return v
}
