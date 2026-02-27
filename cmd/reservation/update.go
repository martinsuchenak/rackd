package reservation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func UpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update an existing reservation",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Reservation ID", Required: true},
			&cli.StringFlag{Name: "hostname", Usage: "Hostname for the reservation"},
			&cli.StringFlag{Name: "purpose", Usage: "Purpose of the reservation"},
			&cli.IntFlag{Name: "expires", Usage: "Days until expiration from now (0 to clear expiration)"},
			&cli.StringFlag{Name: "notes", Usage: "Additional notes"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)
			reservationID := cmd.GetString("id")

			req := map[string]interface{}{}

			if hostname := cmd.GetString("hostname"); hostname != "" {
				req["hostname"] = hostname
			}
			if purpose := cmd.GetString("purpose"); purpose != "" {
				req["purpose"] = purpose
			}
			if days := cmd.GetInt("expires"); days > 0 {
				req["expires_in_days"] = days
			}
			if notes := cmd.GetString("notes"); notes != "" {
				req["notes"] = notes
			}

			if len(req) == 0 {
				return fmt.Errorf("at least one field to update is required")
			}

			body, err := json.Marshal(req)
			if err != nil {
				return err
			}

			resp, err := c.DoRequest("PUT", "/api/reservations/"+reservationID, bytes.NewReader(body))
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
				fmt.Printf("Reservation updated successfully\n\n")
				printReservationDetail(reservation)
			}
			return nil
		},
	}
}
