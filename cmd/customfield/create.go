package customfield

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func CreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new custom field definition",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Usage: "Display name", Required: true},
			&cli.StringFlag{Name: "key", Usage: "Unique key (lowercase, numbers, underscores)", Required: true},
			&cli.StringFlag{Name: "type", Usage: "Field type (text, number, boolean, select)", DefaultValue: "text"},
			&cli.BoolFlag{Name: "required", Usage: "Mark field as required"},
			&cli.StringSliceFlag{Name: "options", Usage: "Options for select type (comma-separated or multiple flags)"},
			&cli.StringFlag{Name: "description", Usage: "Field description"},
			&cli.StringFlag{Name: "output", Usage: "Output format (json/yaml)", DefaultValue: "yaml"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			req := map[string]interface{}{
				"name":        cmd.GetString("name"),
				"key":         cmd.GetString("key"),
				"type":        cmd.GetString("type"),
				"required":    cmd.GetBool("required"),
				"description": cmd.GetString("description"),
			}

			options := cmd.GetStringSlice("options")
			if len(options) > 0 {
				req["options"] = options
			}

			resp, err := c.DoRequest("POST", "/api/custom-fields", req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				return client.HandleError(resp)
			}

			var field map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&field); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(field)
			default:
				client.PrintYAML(field)
			}
			return nil
		},
	}
}
