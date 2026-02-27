package circuit

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func ListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all circuits",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "provider", Usage: "Filter by provider"},
			&cli.StringFlag{Name: "status", Usage: "Filter by status"},
			&cli.StringFlag{Name: "datacenter", Usage: "Filter by datacenter ID"},
			&cli.StringFlag{Name: "type", Usage: "Filter by circuit type"},
			&cli.StringFlag{Name: "output", Usage: "Output format (json/yaml)", DefaultValue: "yaml"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			url := "/api/circuits"
			params := ""
			if provider := cmd.GetString("provider"); provider != "" {
				params += "&provider=" + provider
			}
			if status := cmd.GetString("status"); status != "" {
				params += "&status=" + status
			}
			if datacenter := cmd.GetString("datacenter"); datacenter != "" {
				params += "&datacenter_id=" + datacenter
			}
			if circuitType := cmd.GetString("type"); circuitType != "" {
				params += "&type=" + circuitType
			}
			if params != "" {
				url += "?" + params[1:]
			}

			resp, err := c.DoRequest("GET", url, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var circuits []map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&circuits); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(circuits)
			default:
				client.PrintYAML(circuits)
			}
			return nil
		},
	}
}
