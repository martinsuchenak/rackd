package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func ListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all webhooks",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "active", Usage: "Show only active webhooks"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			path := "/api/webhooks"
			if cmd.GetBool("active") {
				path += "?active=true"
			}

			resp, err := c.DoRequest("GET", path, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var webhooks []map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&webhooks); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(webhooks)
			case "yaml":
				client.PrintYAML(webhooks)
			default:
				printWebhookTable(webhooks)
			}
			return nil
		},
	}
}

func printWebhookTable(webhooks []map[string]interface{}) {
	if len(webhooks) == 0 {
		println("No webhooks found")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tURL\tACTIVE\tEVENTS")
	for _, wh := range webhooks {
		id := getString(wh, "id")
		if len(id) > 8 {
			id = id[:8]
		}
		active := "no"
		if getBool(wh, "active") {
			active = "yes"
		}
		events := ""
		if ev, ok := wh["events"].([]interface{}); ok {
			for i, e := range ev {
				if i > 0 {
					events += ", "
				}
				events += e.(string)
			}
			if len(events) > 30 {
				events = events[:27] + "..."
			}
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			id,
			getString(wh, "name"),
			getString(wh, "url"),
			active,
			events)
	}
	w.Flush()
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
