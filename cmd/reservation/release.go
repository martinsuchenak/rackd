package reservation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func ReleaseCommand() *cli.Command {
	return &cli.Command{
		Name:  "release",
		Usage: "Release a reservation (marks it as released without deleting)",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Reservation ID", Required: true},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)
			reservationID := cmd.GetString("id")

			resp, err := c.DoRequest("POST", "/api/reservations/"+reservationID+"/release", nil)
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

			fmt.Printf("Reservation %s released successfully\n", reservationID)
			return nil
		},
	}
}
