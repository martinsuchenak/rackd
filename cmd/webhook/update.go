package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func UpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a webhook",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Webhook ID", Required: true},
			&cli.StringFlag{Name: "name", Usage: "Webhook name"},
			&cli.StringFlag{Name: "url", Usage: "Webhook URL"},
			&cli.StringFlag{Name: "secret", Usage: "Secret for HMAC signatures"},
			&cli.StringFlag{Name: "events", Usage: "Comma-separated list of events to subscribe to"},
			&cli.StringFlag{Name: "description", Usage: "Webhook description"},
			&cli.BoolFlag{Name: "active", Usage: "Set webhook active"},
			&cli.BoolFlag{Name: "inactive", Usage: "Set webhook inactive"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			body := make(map[string]interface{})

			if name := cmd.GetString("name"); name != "" {
				body["name"] = name
			}
			if url := cmd.GetString("url"); url != "" {
				body["url"] = url
			}
			if secret := cmd.GetString("secret"); secret != "" {
				body["secret"] = secret
			}
			if events := cmd.GetString("events"); events != "" {
				eventList := strings.Split(events, ",")
				for i, e := range eventList {
					eventList[i] = strings.TrimSpace(e)
				}
				body["events"] = eventList
			}
			if desc := cmd.GetString("description"); desc != "" {
				body["description"] = desc
			}

			// Handle active/inactive flags
			if cmd.GetBool("active") {
				body["active"] = true
			} else if cmd.GetBool("inactive") {
				body["active"] = false
			}

			if len(body) == 0 {
				return fmt.Errorf("no update fields provided")
			}

			jsonBody, err := json.Marshal(body)
			if err != nil {
				return err
			}

			id := cmd.GetString("id")
			resp, err := c.DoRequest("PUT", "/api/webhooks/"+id, bytes.NewBuffer(jsonBody))
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var webhook map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&webhook); err != nil {
				return err
			}

			fmt.Printf("Webhook updated successfully: %s\n", id)
			client.PrintJSON(webhook)
			return nil
		},
	}
}
