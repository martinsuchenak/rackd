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

func CreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new IP reservation",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "pool", Usage: "Pool ID", Required: true},
			&cli.StringFlag{Name: "ip", Usage: "IP address (auto-assign if not specified)"},
			&cli.StringFlag{Name: "hostname", Usage: "Hostname for the reservation"},
			&cli.StringFlag{Name: "purpose", Usage: "Purpose of the reservation"},
			&cli.IntFlag{Name: "expires", Usage: "Days until expiration (0 for no expiration)"},
			&cli.StringFlag{Name: "notes", Usage: "Additional notes"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			req := map[string]interface{}{
				"pool_id": cmd.GetString("pool"),
			}

			if ip := cmd.GetString("ip"); ip != "" {
				req["ip_address"] = ip
			}
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

			body, err := json.Marshal(req)
			if err != nil {
				return err
			}

			resp, err := c.DoRequest("POST", "/api/reservations", bytes.NewReader(body))
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
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
				fmt.Printf("Reservation created successfully\n\n")
				printReservationDetail(reservation)
			}
			return nil
		},
	}
}
