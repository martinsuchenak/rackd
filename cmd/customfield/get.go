package customfield

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func GetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a custom field definition by ID",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Custom field ID", Required: true},
			&cli.StringFlag{Name: "output", Usage: "Output format (json/yaml)", DefaultValue: "yaml"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			id := cmd.GetString("id")
			resp, err := c.DoRequest("GET", "/api/custom-fields/"+id, nil)
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

func TypesCommand() *cli.Command {
	return &cli.Command{
		Name:  "types",
		Usage: "List available custom field types",
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			resp, err := c.DoRequest("GET", "/api/custom-fields/types", nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var types []map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&types); err != nil {
				return err
			}

			fmt.Println("Available custom field types:")
			for _, t := range types {
				fmt.Printf("  - %s (%s)\n", t["value"], t["label"])
			}
			return nil
		},
	}
}
