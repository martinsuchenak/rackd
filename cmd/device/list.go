package device

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func ListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all devices",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "query", Usage: "Search devices by text (can't be combined with filters)"},
			&cli.StringFlag{Name: "tags", Usage: "Filter by tags (comma-separated)"},
			&cli.StringFlag{Name: "datacenter", Usage: "Filter by datacenter ID"},
			&cli.StringFlag{Name: "network", Usage: "Filter by network ID"},
			&cli.StringFlag{Name: "pool", Usage: "Filter by pool ID"},
			&cli.StringFlag{Name: "status", Usage: "Filter by status (planned, active, maintenance, decommissioned)"},
			&cli.IntFlag{Name: "limit", Usage: "Return at most this many devices (default: all)"},
			&cli.IntFlag{Name: "offset", Usage: "Skip this many devices"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			params := url.Values{}
			for flag, param := range map[string]string{
				"tags": "tags", "datacenter": "datacenter_id", "network": "network_id", "pool": "pool_id", "status": "status",
			} {
				if v := cmd.GetString(flag); v != "" {
					params.Set(param, v)
				}
			}

			var devices []map[string]interface{}
			var err error
			if q := cmd.GetString("query"); q != "" {
				// Search has its own endpoint, which takes no filters.
				if len(params) > 0 {
					return fmt.Errorf("--query can't be combined with --tags, --datacenter, --network, --pool or --status")
				}
				devices, err = getDevices(c, "/api/v1/devices/search?"+url.Values{"q": {q}}.Encode())
			} else {
				devices, err = listDevices(c, params, cmd.GetInt("limit"), cmd.GetInt("offset"))
			}
			if err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(devices)
			case "yaml":
				client.PrintYAML(devices)
			default:
				client.PrintDeviceTable(devices)
			}
			return nil
		},
	}
}

// devicePageSize is the API's largest page.
const devicePageSize = 1000

// listDevices fetches devices matching params. With limit 0 it fetches every
// page; the API returns 100 at a time by default, so a single request would
// silently stop there.
func listDevices(c *client.Client, params url.Values, limit, offset int) ([]map[string]interface{}, error) {
	var all []map[string]interface{}
	for {
		size := devicePageSize
		if limit > 0 && limit-len(all) < size {
			size = limit - len(all)
		}
		params.Set("limit", fmt.Sprintf("%d", size))
		params.Set("offset", fmt.Sprintf("%d", offset+len(all)))
		page, err := getDevices(c, "/api/v1/devices?"+params.Encode())
		if err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) != size || (limit > 0 && len(all) >= limit) {
			return all, nil
		}
	}
}

func getDevices(c *client.Client, path string) ([]map[string]interface{}, error) {
	resp, err := c.DoRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, client.HandleError(resp)
	}
	var devices []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		return nil, err
	}
	return devices, nil
}
