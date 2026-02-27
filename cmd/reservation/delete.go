package reservation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func DeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a reservation",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Reservation ID", Required: true},
			&cli.BoolFlag{Name: "force", Usage: "Skip confirmation"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)
			reservationID := cmd.GetString("id")

			if !cmd.GetBool("force") {
				fmt.Printf("Are you sure you want to delete reservation %s? [y/N]: ", reservationID)
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Cancelled")
					return nil
				}
			}

			resp, err := c.DoRequest("DELETE", "/api/reservations/"+reservationID, nil)
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

			fmt.Printf("Reservation %s deleted successfully\n", reservationID)
			return nil
		},
	}
}
