package scanprofile

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "scan-profile",
		Usage: "Manage scan profiles",
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
		Usage: "List scan profiles",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			c := client.NewClient(client.LoadConfig())
			resp, err := c.DoRequest("GET", "/api/scan-profiles", nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var profiles []map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
				return err
			}

			if cmd.GetString("output") == "json" {
				client.PrintJSON(profiles)
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tTYPE\tTIMEOUT\tWORKERS\tSNMP\tSSH")
			for _, p := range profiles {
				fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%v\t%v\t%v\n",
					client.GetString(p, "id"),
					client.GetString(p, "name"),
					client.GetString(p, "scan_type"),
					p["timeout_sec"],
					p["max_workers"],
					p["enable_snmp"],
					p["enable_ssh"])
			}
			w.Flush()
			return nil
		},
	}
}

func getCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a scan profile by ID",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Profile ID", Required: true},
			&cli.StringFlag{Name: "output", Usage: "Output format (json/yaml)", DefaultValue: "json"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			c := client.NewClient(client.LoadConfig())
			resp, err := c.DoRequest("GET", "/api/scan-profiles/"+cmd.GetString("id"), nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var profile map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&profile)

			switch cmd.GetString("output") {
			case "yaml":
				client.PrintYAML(profile)
			default:
				client.PrintJSON(profile)
			}
			return nil
		},
	}
}

func createCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a scan profile",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Usage: "Profile name", Required: true},
			&cli.StringFlag{Name: "type", Usage: "Scan type (quick, full, deep, custom)", Required: true},
			&cli.IntFlag{Name: "timeout", Usage: "Timeout in seconds", DefaultValue: 30},
			&cli.IntFlag{Name: "workers", Usage: "Max concurrent workers", DefaultValue: 10},
			&cli.BoolFlag{Name: "snmp", Usage: "Enable SNMP probing"},
			&cli.BoolFlag{Name: "ssh", Usage: "Enable SSH probing"},
			&cli.StringFlag{Name: "ports", Usage: "Comma-separated port list (for custom type)"},
			&cli.StringFlag{Name: "description", Usage: "Description"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			c := client.NewClient(client.LoadConfig())

			body := map[string]interface{}{
				"name":        cmd.GetString("name"),
				"scan_type":   cmd.GetString("type"),
				"timeout_sec": cmd.GetInt("timeout"),
				"max_workers": cmd.GetInt("workers"),
				"enable_snmp": cmd.GetBool("snmp"),
				"enable_ssh":  cmd.GetBool("ssh"),
			}
			if desc := cmd.GetString("description"); desc != "" {
				body["description"] = desc
			}
			if ports := cmd.GetString("ports"); ports != "" {
				body["ports"] = parsePorts(ports)
			}

			resp, err := c.DoRequest("POST", "/api/scan-profiles", body)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusCreated {
				return client.HandleError(resp)
			}

			var profile map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&profile)
			fmt.Printf("Scan profile created: %s\n", client.GetString(profile, "id"))
			return nil
		},
	}
}

func updateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a scan profile",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Profile ID", Required: true},
			&cli.StringFlag{Name: "name", Usage: "Profile name"},
			&cli.StringFlag{Name: "type", Usage: "Scan type (quick, full, deep, custom)"},
			&cli.IntFlag{Name: "timeout", Usage: "Timeout in seconds"},
			&cli.IntFlag{Name: "workers", Usage: "Max concurrent workers"},
			&cli.BoolFlag{Name: "snmp", Usage: "Enable SNMP probing"},
			&cli.BoolFlag{Name: "ssh", Usage: "Enable SSH probing"},
			&cli.StringFlag{Name: "ports", Usage: "Comma-separated port list"},
			&cli.StringFlag{Name: "description", Usage: "Description"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			c := client.NewClient(client.LoadConfig())

			// Fetch current profile first
			resp, err := c.DoRequest("GET", "/api/scan-profiles/"+cmd.GetString("id"), nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var body map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&body)

			// Override with provided flags
			if v := cmd.GetString("name"); v != "" {
				body["name"] = v
			}
			if v := cmd.GetString("type"); v != "" {
				body["scan_type"] = v
			}
			if v := cmd.GetInt("timeout"); v > 0 {
				body["timeout_sec"] = v
			}
			if v := cmd.GetInt("workers"); v > 0 {
				body["max_workers"] = v
			}
			if cmd.GetBool("snmp") {
				body["enable_snmp"] = true
			}
			if cmd.GetBool("ssh") {
				body["enable_ssh"] = true
			}
			if v := cmd.GetString("ports"); v != "" {
				body["ports"] = parsePorts(v)
			}
			if v := cmd.GetString("description"); v != "" {
				body["description"] = v
			}

			resp2, err := c.DoRequest("PUT", "/api/scan-profiles/"+cmd.GetString("id"), body)
			if err != nil {
				return err
			}
			defer resp2.Body.Close()
			if resp2.StatusCode != http.StatusOK {
				return client.HandleError(resp2)
			}

			fmt.Println("Scan profile updated")
			return nil
		},
	}
}

func deleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a scan profile",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Profile ID", Required: true},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			c := client.NewClient(client.LoadConfig())
			resp, err := c.DoRequest("DELETE", "/api/scan-profiles/"+cmd.GetString("id"), nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusNoContent {
				return client.HandleError(resp)
			}
			fmt.Println("Scan profile deleted")
			return nil
		},
	}
}

func parsePorts(s string) []int {
	var ports []int
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if n, err := strconv.Atoi(p); err == nil {
			ports = append(ports, n)
		}
	}
	return ports
}
