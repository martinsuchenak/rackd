package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func RecordsCommand() *cli.Command {
	return &cli.Command{
		Name:  "records",
		Usage: "List DNS records for a zone",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "zone", Usage: "Zone ID", Required: true},
			&cli.StringFlag{Name: "type", Usage: "Filter by record type (A, AAAA, CNAME, MX, TXT, PTR, NS, SRV)"},
			&cli.StringFlag{Name: "device", Usage: "Filter by device ID"},
			&cli.StringFlag{Name: "name", Usage: "Filter by record name"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			zoneID := cmd.GetString("zone")
			url := "/api/dns/zones/" + zoneID + "/records"
			params := ""

			if recordType := cmd.GetString("type"); recordType != "" {
				params += "&type=" + recordType
			}
			if device := cmd.GetString("device"); device != "" {
				params += "&device_id=" + device
			}
			if name := cmd.GetString("name"); name != "" {
				params += "&name=" + name
			}
			if params != "" {
				url += "?" + params[1:]
			}

			resp, err := c.DoRequest("GET", url, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var records []map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(records)
			case "yaml":
				client.PrintYAML(records)
			default:
				printRecordsTable(records)
			}
			return nil
		},
	}
}

func printRecordsTable(records []map[string]interface{}) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tVALUE\tTTL\tDEVICE")
	for _, r := range records {
		value := client.GetString(r, "value")
		// Truncate long values
		if len(value) > 40 {
			value = value[:37] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%s\n",
			client.GetString(r, "name"),
			client.GetString(r, "type"),
			value,
			r["ttl"],
			client.GetString(r, "device_id"))
	}
	w.Flush()

	// Print summary
	fmt.Printf("\nTotal records: %d\n", len(records))
}
