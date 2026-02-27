package customfield

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
		Usage: "List all custom field definitions",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "type", Usage: "Filter by field type (text/number/boolean/select)"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			path := "/api/custom-fields"
			if cmd.GetString("type") != "" {
				path += "?type=" + cmd.GetString("type")
			}

			resp, err := c.DoRequest("GET", path, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var fields []map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&fields); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(fields)
			case "yaml":
				client.PrintYAML(fields)
			default:
				printFieldTable(fields)
			}
			return nil
		},
	}
}

func printFieldTable(fields []map[string]interface{}) {
	if len(fields) == 0 {
		println("No custom fields found")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tKEY\tTYPE\tREQUIRED")
	for _, f := range fields {
		id := getString(f, "id")
		if len(id) > 8 {
			id = id[:8]
		}
		required := "no"
		if getBool(f, "required") {
			required = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			id,
			getString(f, "name"),
			getString(f, "key"),
			getString(f, "type"),
			required,
		)
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
