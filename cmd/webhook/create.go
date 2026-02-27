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

func CreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new webhook",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Usage: "Webhook name", Required: true},
			&cli.StringFlag{Name: "url", Usage: "Webhook URL", Required: true},
			&cli.StringFlag{Name: "secret", Usage: "Secret for HMAC signatures"},
			&cli.StringFlag{Name: "events", Usage: "Comma-separated list of events to subscribe to", Required: true},
			&cli.StringFlag{Name: "description", Usage: "Webhook description"},
			&cli.BoolFlag{Name: "inactive", Usage: "Create as inactive (default is active)"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			events := strings.Split(cmd.GetString("events"), ",")
			for i, e := range events {
				events[i] = strings.TrimSpace(e)
			}

			active := !cmd.GetBool("inactive")

			body := map[string]interface{}{
				"name":        cmd.GetString("name"),
				"url":         cmd.GetString("url"),
				"events":      events,
				"active":      active,
				"description": cmd.GetString("description"),
			}
			if secret := cmd.GetString("secret"); secret != "" {
				body["secret"] = secret
			}

			jsonBody, err := json.Marshal(body)
			if err != nil {
				return err
			}

			resp, err := c.DoRequest("POST", "/api/webhooks", bytes.NewBuffer(jsonBody))
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				return client.HandleError(resp)
			}

			var webhook map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&webhook); err != nil {
				return err
			}

			fmt.Printf("Webhook created successfully: %s\n", getString(webhook, "id"))
			client.PrintJSON(webhook)
			return nil
		},
	}
}
