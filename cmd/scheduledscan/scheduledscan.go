package scheduledscan

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"text/tabwriter"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "scheduled-scan",
		Usage: "Manage scheduled discovery scans",
		Commands: []*cli.Command{
			listCommand(),
			getCommand(),
			createCommand(),
			updateCommand(),
			deleteCommand(),
		},
	}
}

func listCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List scheduled scans",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "network", Usage: "Filter by network ID"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			c := client.NewClient(client.LoadConfig())

			path := "/api/scheduled-scans"
			if nid := cmd.GetString("network"); nid != "" {
				path += "?" + url.Values{"network_id": {nid}}.Encode()
			}

			resp, err := c.DoRequest("GET", path, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var scans []map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&scans)

			if cmd.GetString("output") == "json" {
				client.PrintJSON(scans)
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tNETWORK\tPROFILE\tCRON\tENABLED\tLAST RUN")
			for _, s := range scans {
				lastRun := "never"
				if v, ok := s["last_run_at"].(string); ok && v != "" {
					lastRun = v
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%v\t%s\n",
					client.GetString(s, "id"),
					client.GetString(s, "name"),
					client.GetString(s, "network_id"),
					client.GetString(s, "profile_id"),
					client.GetString(s, "cron_expression"),
					s["enabled"],
					lastRun)
			}
			w.Flush()
			return nil
		},
	}
}

func getCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a scheduled scan by ID",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Scheduled scan ID", Required: true},
			&cli.StringFlag{Name: "output", Usage: "Output format (json/yaml)", DefaultValue: "json"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			c := client.NewClient(client.LoadConfig())
			resp, err := c.DoRequest("GET", "/api/scheduled-scans/"+cmd.GetString("id"), nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var scan map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&scan)

			switch cmd.GetString("output") {
			case "yaml":
				client.PrintYAML(scan)
			default:
				client.PrintJSON(scan)
			}
			return nil
		},
	}
}

func createCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a scheduled scan",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Usage: "Schedule name", Required: true},
			&cli.StringFlag{Name: "network", Usage: "Network ID", Required: true},
			&cli.StringFlag{Name: "profile", Usage: "Scan profile ID", Required: true},
			&cli.StringFlag{Name: "cron", Usage: "Cron expression (e.g., '0 2 * * *' for daily at 2am)", Required: true},
			&cli.BoolFlag{Name: "enabled", Usage: "Enable the schedule", DefaultValue: true},
			&cli.StringFlag{Name: "description", Usage: "Description"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			c := client.NewClient(client.LoadConfig())

			body := map[string]interface{}{
				"name":            cmd.GetString("name"),
				"network_id":     cmd.GetString("network"),
				"profile_id":     cmd.GetString("profile"),
				"cron_expression": cmd.GetString("cron"),
				"enabled":        cmd.GetBool("enabled"),
			}
			if desc := cmd.GetString("description"); desc != "" {
				body["description"] = desc
			}

			resp, err := c.DoRequest("POST", "/api/scheduled-scans", body)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusCreated {
				return client.HandleError(resp)
			}

			var scan map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&scan)
			fmt.Printf("Scheduled scan created: %s\n", client.GetString(scan, "id"))
			return nil
		},
	}
}

func updateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a scheduled scan",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Scheduled scan ID", Required: true},
			&cli.StringFlag{Name: "name", Usage: "Schedule name"},
			&cli.StringFlag{Name: "network", Usage: "Network ID"},
			&cli.StringFlag{Name: "profile", Usage: "Scan profile ID"},
			&cli.StringFlag{Name: "cron", Usage: "Cron expression"},
			&cli.BoolFlag{Name: "enable", Usage: "Enable the schedule"},
			&cli.BoolFlag{Name: "disable", Usage: "Disable the schedule"},
			&cli.StringFlag{Name: "description", Usage: "Description"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			c := client.NewClient(client.LoadConfig())

			// Fetch current
			resp, err := c.DoRequest("GET", "/api/scheduled-scans/"+cmd.GetString("id"), nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var body map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&body)

			if v := cmd.GetString("name"); v != "" {
				body["name"] = v
			}
			if v := cmd.GetString("network"); v != "" {
				body["network_id"] = v
			}
			if v := cmd.GetString("profile"); v != "" {
				body["profile_id"] = v
			}
			if v := cmd.GetString("cron"); v != "" {
				body["cron_expression"] = v
			}
			if cmd.GetBool("enable") {
				body["enabled"] = true
			}
			if cmd.GetBool("disable") {
				body["enabled"] = false
			}
			if v := cmd.GetString("description"); v != "" {
				body["description"] = v
			}

			resp2, err := c.DoRequest("PUT", "/api/scheduled-scans/"+cmd.GetString("id"), body)
			if err != nil {
				return err
			}
			defer resp2.Body.Close()
			if resp2.StatusCode != http.StatusOK {
				return client.HandleError(resp2)
			}

			fmt.Println("Scheduled scan updated")
			return nil
		},
	}
}

func deleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a scheduled scan",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Scheduled scan ID", Required: true},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			c := client.NewClient(client.LoadConfig())
			resp, err := c.DoRequest("DELETE", "/api/scheduled-scans/"+cmd.GetString("id"), nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusNoContent {
				return client.HandleError(resp)
			}
			fmt.Println("Scheduled scan deleted")
			return nil
		},
	}
}
