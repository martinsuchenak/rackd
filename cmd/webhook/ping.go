package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func PingCommand() *cli.Command {
	return &cli.Command{
		Name:  "ping",
		Usage: "Send a test event to a webhook",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Webhook ID", Required: true},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			id := cmd.GetString("id")
			resp, err := c.DoRequest("POST", "/api/webhooks/"+id+"/ping", nil)
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

			fmt.Println("Webhook ping sent successfully")
			client.PrintJSON(result)
			return nil
		},
	}
}
