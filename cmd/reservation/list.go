package reservation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"text/tabwriter"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func ListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all reservations",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "pool", Usage: "Filter by pool ID"},
			&cli.StringFlag{Name: "status", Usage: "Filter by status (active, expired, claimed, released)"},
			&cli.StringFlag{Name: "reserved-by", Usage: "Filter by user who reserved"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			params := url.Values{}
			if poolID := cmd.GetString("pool"); poolID != "" {
				params.Set("pool_id", poolID)
			}
			if status := cmd.GetString("status"); status != "" {
				params.Set("status", status)
			}
			if reservedBy := cmd.GetString("reserved-by"); reservedBy != "" {
				params.Set("reserved_by", reservedBy)
			}

			path := "/api/reservations"
			if len(params) > 0 {
				path += "?" + params.Encode()
			}

			resp, err := c.DoRequest("GET", path, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var reservations []map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&reservations); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(reservations)
			case "yaml":
				client.PrintYAML(reservations)
			default:
				printReservationTable(reservations)
			}
			return nil
		},
	}
}

func printReservationTable(reservations []map[string]interface{}) {
	if len(reservations) == 0 {
		println("No reservations found")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tIP ADDRESS\tPOOL\tHOSTNAME\tSTATUS\tRESERVED BY\tEXPIRES")
	for _, r := range reservations {
		id := getString(r, "id")
		if len(id) > 8 {
			id = id[:8]
		}
		poolID := getString(r, "pool_id")
		if len(poolID) > 8 {
			poolID = poolID[:8]
		}
		expires := getString(r, "expires_at")
		if expires != "" && len(expires) > 10 {
			expires = expires[:10]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			id,
			getString(r, "ip_address"),
			poolID,
			getString(r, "hostname"),
			getString(r, "status"),
			getString(r, "reserved_by"),
			expires)
	}
	w.Flush()
}
