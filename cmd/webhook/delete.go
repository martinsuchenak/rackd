package webhook

import (
	"context"
	"fmt"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func DeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a webhook",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Webhook ID", Required: true},
			&cli.BoolFlag{Name: "force", Usage: "Skip confirmation"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			id := cmd.GetString("id")

			if !cmd.GetBool("force") {
				fmt.Printf("Are you sure you want to delete webhook %s? (y/N): ", id)
				var confirm string
				fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Cancelled")
					return nil
				}
			}

			resp, err := c.DoRequest("DELETE", "/api/webhooks/"+id, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			fmt.Printf("Webhook %s deleted successfully\n", id)
			return nil
		},
	}
}
