package dns

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/paularlott/cli"
)

func ZoneCommand() *cli.Command {
	return &cli.Command{
		Name:  "zone",
		Usage: "DNS zone management",
		Commands: []*cli.Command{
			zoneListCommand(),
			zoneGetCommand(),
			zoneCreateCommand(),
			zoneUpdateCommand(),
			zoneDeleteCommand(),
		},
	}
}

func zoneListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List DNS zones",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "provider", Usage: "Filter by provider ID"},
			&cli.StringFlag{Name: "network", Usage: "Filter by network ID"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			url := "/api/dns/zones"
			params := ""
			if provider := cmd.GetString("provider"); provider != "" {
				params += "&provider_id=" + provider
			}
			if network := cmd.GetString("network"); network != "" {
				params += "&network_id=" + network
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

			var zones []map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&zones); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(zones)
			case "yaml":
				client.PrintYAML(zones)
			default:
				printZoneTable(zones)
			}
			return nil
		},
	}
}

func zoneGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a DNS zone by ID",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Zone ID", Required: true},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "yaml"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			id := cmd.GetString("id")
			resp, err := c.DoRequest("GET", "/api/dns/zones/"+id, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var zone map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&zone); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(zone)
			case "table":
				printZoneTable([]map[string]interface{}{zone})
			default:
				client.PrintYAML(zone)
			}
			return nil
		},
	}
}

func zoneCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new DNS zone",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Usage: "Zone name (e.g., example.com)", Required: true},
			&cli.StringFlag{Name: "provider", Usage: "Provider ID", Required: true},
			&cli.StringFlag{Name: "network", Usage: "Network ID (for auto PTR records)"},
			&cli.BoolFlag{Name: "enable-auto-sync", Usage: "Enable automatic sync"},
			&cli.BoolFlag{Name: "disable-auto-sync", Usage: "Disable automatic sync"},
			&cli.BoolFlag{Name: "create-ptr", Usage: "Create PTR records"},
			&cli.IntFlag{Name: "ttl", Usage: "Default TTL"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			req := map[string]interface{}{
				"name":        cmd.GetString("name"),
				"provider_id": cmd.GetString("provider"),
			}

			if v := cmd.GetString("network"); v != "" {
				req["network_id"] = v
			}
			if cmd.GetBool("enable-auto-sync") {
				req["auto_sync"] = true
			} else if cmd.GetBool("disable-auto-sync") {
				req["auto_sync"] = false
			}
			if cmd.GetBool("create-ptr") {
				req["create_ptr"] = true
			}
			if v := cmd.GetInt("ttl"); v > 0 {
				req["default_ttl"] = v
			}

			resp, err := c.DoRequest("POST", "/api/dns/zones", req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated {
				return client.HandleError(resp)
			}

			var created map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(created)
			case "yaml":
				client.PrintYAML(created)
			default:
				fmt.Printf("DNS zone created successfully\n")
				fmt.Printf("ID: %s\n", created["id"])
				fmt.Printf("Name: %s\n", created["name"])
			}
			return nil
		},
	}
}

func zoneUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a DNS zone",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Zone ID", Required: true},
			&cli.StringFlag{Name: "name", Usage: "Zone name (e.g., example.com)"},
			&cli.StringFlag{Name: "provider", Usage: "Provider ID"},
			&cli.StringFlag{Name: "network", Usage: "Network ID (for auto PTR records)"},
			&cli.BoolFlag{Name: "enable-auto-sync", Usage: "Enable automatic sync"},
			&cli.BoolFlag{Name: "disable-auto-sync", Usage: "Disable automatic sync"},
			&cli.BoolFlag{Name: "create-ptr", Usage: "Enable PTR record creation"},
			&cli.BoolFlag{Name: "no-create-ptr", Usage: "Disable PTR record creation"},
			&cli.IntFlag{Name: "ttl", Usage: "Default TTL"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "yaml"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			updates := make(map[string]interface{})

			if v := cmd.GetString("name"); v != "" {
				updates["name"] = v
			}
			if v := cmd.GetString("provider"); v != "" {
				updates["provider_id"] = v
			}
			if v := cmd.GetString("network"); v != "" {
				updates["network_id"] = v
			}
			if cmd.GetBool("enable-auto-sync") {
				updates["auto_sync"] = true
			} else if cmd.GetBool("disable-auto-sync") {
				updates["auto_sync"] = false
			}
			if cmd.GetBool("create-ptr") {
				updates["create_ptr"] = true
			} else if cmd.GetBool("no-create-ptr") {
				updates["create_ptr"] = false
			}
			if v := cmd.GetInt("ttl"); v > 0 {
				updates["default_ttl"] = v
			}

			if len(updates) == 0 {
				fmt.Println("No updates specified")
				return nil
			}

			resp, err := c.DoRequest("PUT", "/api/dns/zones/"+cmd.GetString("id"), updates)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var zone map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&zone); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(zone)
			case "table":
				printZoneTable([]map[string]interface{}{zone})
			default:
				client.PrintYAML(zone)
			}
			return nil
		},
	}
}

func zoneDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a DNS zone",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Zone ID", Required: true},
			&cli.BoolFlag{Name: "force", Usage: "Skip confirmation"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			id := cmd.GetString("id")

			if !cmd.GetBool("force") {
				fmt.Printf("Are you sure you want to delete DNS zone %s? [y/N]: ", id)
				reader := bufio.NewReader(os.Stdin)
				confirm, _ := reader.ReadString('\n')
				confirm = strings.TrimSpace(strings.ToLower(confirm))
				if confirm != "y" && confirm != "yes" {
					fmt.Println("Deletion cancelled")
					return nil
				}
			}

			resp, err := c.DoRequest("DELETE", "/api/dns/zones/"+id, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
				return client.HandleError(resp)
			}

			fmt.Println("DNS zone deleted successfully")
			return nil
		},
	}
}

func printZoneTable(zones []map[string]interface{}) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tPROVIDER\tNETWORK\tAUTO-SYNC")
	for _, z := range zones {
		autoSync := "false"
		if v, ok := z["auto_sync"].(bool); ok && v {
			autoSync = "true"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			client.GetString(z, "id"),
			client.GetString(z, "name"),
			client.GetString(z, "provider_id"),
			client.GetString(z, "network_id"),
			autoSync)
	}
	w.Flush()
}
