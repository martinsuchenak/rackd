package reservation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func GetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a reservation by ID",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Reservation ID", Required: true},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)
			reservationID := cmd.GetString("id")

			resp, err := c.DoRequest("GET", "/api/reservations/"+reservationID, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var reservation map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&reservation); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(reservation)
			case "yaml":
				client.PrintYAML(reservation)
			default:
				printReservationDetail(reservation)
			}
			return nil
		},
	}
}

func printReservationDetail(r map[string]interface{}) {
	fmt.Printf("ID:          %s\n", getString(r, "id"))
	fmt.Printf("Pool ID:     %s\n", getString(r, "pool_id"))
	fmt.Printf("IP Address:  %s\n", getString(r, "ip_address"))
	fmt.Printf("Hostname:    %s\n", getString(r, "hostname"))
	fmt.Printf("Purpose:     %s\n", getString(r, "purpose"))
	fmt.Printf("Status:      %s\n", getString(r, "status"))
	fmt.Printf("Reserved By: %s\n", getString(r, "reserved_by"))
	fmt.Printf("Reserved At: %s\n", getString(r, "reserved_at"))

	if expiresAt := getString(r, "expires_at"); expiresAt != "" {
		fmt.Printf("Expires At:  %s\n", expiresAt)
	}

	if notes := getString(r, "notes"); notes != "" {
		fmt.Printf("Notes:       %s\n", notes)
	}

	fmt.Printf("Created At:  %s\n", getString(r, "created_at"))
	fmt.Printf("Updated At:  %s\n", getString(r, "updated_at"))
}
