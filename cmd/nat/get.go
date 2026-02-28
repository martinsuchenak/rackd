package nat

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func GetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a NAT mapping by ID",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "NAT mapping ID", Required: true},
			&cli.StringFlag{Name: "output", Usage: "Output format (json/yaml)", DefaultValue: "yaml"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			id := cmd.GetString("id")
			resp, err := c.DoRequest("GET", "/api/nat/"+id, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var mapping map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&mapping); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(mapping)
			default:
				client.PrintYAML(mapping)
			}
			return nil
		},
	}
}
