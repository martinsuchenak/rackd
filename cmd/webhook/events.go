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

func EventsCommand() *cli.Command {
	return &cli.Command{
		Name:  "events",
		Usage: "List all available event types",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			resp, err := c.DoRequest("GET", "/api/webhooks/events", nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var events []map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(events)
			case "yaml":
				client.PrintYAML(events)
			default:
				printEventsTable(events)
			}
			return nil
		},
	}
}

func printEventsTable(events []map[string]interface{}) {
	if len(events) == 0 {
		println("No events available")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "VALUE\tLABEL")
	for _, e := range events {
		fmt.Fprintf(w, "%s\t%s\n",
			getString(e, "value"),
			getString(e, "label"))
	}
	w.Flush()
}
