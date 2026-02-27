package conflict

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func DetectCommand() *cli.Command {
	return &cli.Command{
		Name:  "detect",
		Usage: "Detect conflicts in the infrastructure",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "type", Usage: "Type of conflict to detect (duplicate_ip, overlapping_subnet), omit for both"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			conflictType := cmd.GetString("type")

			params := url.Values{}
			if conflictType != "" {
				params.Set("type", conflictType)
			}

			path := "/api/conflicts/detect"
			if len(params) > 0 {
				path += "?" + params.Encode()
			}

			resp, err := c.DoRequest("POST", path, nil)
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

			conflictsJSON, _ := json.Marshal(result["conflicts"])
			var conflicts []map[string]interface{}
			json.Unmarshal(conflictsJSON, &conflicts)

			fmt.Printf("Detection complete. Found %d conflict(s).\n", len(conflicts))

			if len(conflicts) > 0 {
				switch cmd.GetString("output") {
				case "json":
					client.PrintJSON(conflicts)
				case "yaml":
					client.PrintYAML(conflicts)
				default:
					client.PrintConflictTable(conflicts)
				}
			}
			return nil
		},
	}
}
