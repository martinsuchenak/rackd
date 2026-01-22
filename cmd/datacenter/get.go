package datacenter

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
		Usage: "Get a datacenter by ID",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Datacenter ID", Required: true},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)
			dcID := cmd.GetString("id")

			resp, err := c.DoRequest("GET", "/api/datacenters/"+dcID, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var dc map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&dc); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(dc)
			case "yaml":
				client.PrintYAML(dc)
			default:
				printDatacenterDetail(dc)
			}
			return nil
		},
	}
}

func printDatacenterDetail(dc map[string]interface{}) {
	fmt.Printf("ID:          %s\n", getString(dc, "id"))
	fmt.Printf("Name:        %s\n", getString(dc, "name"))
	fmt.Printf("Description: %s\n", getString(dc, "description"))
	fmt.Printf("Location:    %s\n", getString(dc, "location"))
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
