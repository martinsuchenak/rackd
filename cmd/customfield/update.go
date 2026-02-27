package customfield

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func UpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update an existing custom field definition",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Custom field ID", Required: true},
			&cli.StringFlag{Name: "name", Usage: "Display name"},
			&cli.StringFlag{Name: "key", Usage: "Unique key (lowercase, numbers, underscores)"},
			&cli.StringFlag{Name: "type", Usage: "Field type (text, number, boolean, select)"},
			&cli.BoolFlag{Name: "required", Usage: "Mark field as required"},
			&cli.StringSliceFlag{Name: "options", Usage: "Options for select type"},
			&cli.StringFlag{Name: "description", Usage: "Field description"},
			&cli.StringFlag{Name: "output", Usage: "Output format (json/yaml)", DefaultValue: "yaml"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			req := make(map[string]interface{})

			if v := cmd.GetString("name"); v != "" {
				req["name"] = v
			}
			if v := cmd.GetString("key"); v != "" {
				req["key"] = v
			}
			if v := cmd.GetString("type"); v != "" {
				req["type"] = v
			}
			// For boolean, we always send it since there's no way to know if it was explicitly set
			req["required"] = cmd.GetBool("required")
			if v := cmd.GetString("description"); v != "" {
				req["description"] = v
			}
			if options := cmd.GetStringSlice("options"); len(options) > 0 {
				req["options"] = options
			}

			id := cmd.GetString("id")
			resp, err := c.DoRequest("PUT", "/api/custom-fields/"+id, req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
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
