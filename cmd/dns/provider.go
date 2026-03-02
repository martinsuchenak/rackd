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

func ProviderCommand() *cli.Command {
	return &cli.Command{
		Name:  "provider",
		Usage: "DNS provider management",
		Commands: []*cli.Command{
			providerListCommand(),
			providerGetCommand(),
			providerCreateCommand(),
			providerUpdateCommand(),
			providerDeleteCommand(),
			providerTestCommand(),
		},
	}
}

func providerListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List DNS providers",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "type", Usage: "Filter by provider type"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			url := "/api/dns/providers"
			params := ""
			if providerType := cmd.GetString("type"); providerType != "" {
				params += "&type=" + providerType
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

			var providers []map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&providers); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(providers)
			case "yaml":
				client.PrintYAML(providers)
			default:
				printProviderTable(providers)
			}
			return nil
		},
	}
}

func providerGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get a DNS provider by ID",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Provider ID", Required: true},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "yaml"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			id := cmd.GetString("id")
			resp, err := c.DoRequest("GET", "/api/dns/providers/"+id, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var provider map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&provider); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(provider)
			case "table":
				printProviderTable([]map[string]interface{}{provider})
			default:
				client.PrintYAML(provider)
			}
			return nil
		},
	}
}

func providerCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a new DNS provider",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Usage: "Provider name", Required: true},
			&cli.StringFlag{Name: "type", Usage: "Provider type (technitium, powerdns, bind)", Required: true},
			&cli.StringFlag{Name: "endpoint", Usage: "API endpoint URL"},
			&cli.StringFlag{Name: "token", Usage: "API token or credentials"},
			&cli.StringFlag{Name: "description", Usage: "Provider description"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			req := map[string]interface{}{
				"name": cmd.GetString("name"),
				"type": cmd.GetString("type"),
			}

			if v := cmd.GetString("endpoint"); v != "" {
				req["endpoint"] = v
			}
			if v := cmd.GetString("token"); v != "" {
				req["token"] = v
			}
			if v := cmd.GetString("description"); v != "" {
				req["description"] = v
			}

			resp, err := c.DoRequest("POST", "/api/dns/providers", req)
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
				fmt.Printf("DNS provider created successfully\n")
				fmt.Printf("ID: %s\n", created["id"])
				fmt.Printf("Name: %s\n", created["name"])
			}
			return nil
		},
	}
}

func providerUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Update a DNS provider",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Provider ID", Required: true},
			&cli.StringFlag{Name: "name", Usage: "Provider name"},
			&cli.StringFlag{Name: "type", Usage: "Provider type (technitium, powerdns, bind)"},
			&cli.StringFlag{Name: "endpoint", Usage: "API endpoint URL"},
			&cli.StringFlag{Name: "token", Usage: "API token or credentials"},
			&cli.StringFlag{Name: "description", Usage: "Provider description"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "yaml"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			updates := make(map[string]interface{})

			if v := cmd.GetString("name"); v != "" {
				updates["name"] = v
			}
			if v := cmd.GetString("type"); v != "" {
				updates["type"] = v
			}
			if v := cmd.GetString("endpoint"); v != "" {
				updates["endpoint"] = v
			}
			if v := cmd.GetString("token"); v != "" {
				updates["token"] = v
			}
			if v := cmd.GetString("description"); v != "" {
				updates["description"] = v
			}

			if len(updates) == 0 {
				fmt.Println("No updates specified")
				return nil
			}

			resp, err := c.DoRequest("PUT", "/api/dns/providers/"+cmd.GetString("id"), updates)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var provider map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&provider); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(provider)
			case "table":
				printProviderTable([]map[string]interface{}{provider})
			default:
				client.PrintYAML(provider)
			}
			return nil
		},
	}
}

func providerDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a DNS provider",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Provider ID", Required: true},
			&cli.BoolFlag{Name: "force", Usage: "Skip confirmation"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			id := cmd.GetString("id")

			if !cmd.GetBool("force") {
				fmt.Printf("Are you sure you want to delete DNS provider %s? [y/N]: ", id)
				reader := bufio.NewReader(os.Stdin)
				confirm, _ := reader.ReadString('\n')
				confirm = strings.TrimSpace(strings.ToLower(confirm))
				if confirm != "y" && confirm != "yes" {
					fmt.Println("Deletion cancelled")
					return nil
				}
			}

			resp, err := c.DoRequest("DELETE", "/api/dns/providers/"+id, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
				return client.HandleError(resp)
			}

			fmt.Println("DNS provider deleted successfully")
			return nil
		},
	}
}

func providerTestCommand() *cli.Command {
	return &cli.Command{
		Name:  "test",
		Usage: "Test DNS provider connection",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Provider ID", Required: true},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			id := cmd.GetString("id")
			resp, err := c.DoRequest("POST", "/api/dns/providers/"+id+"/test", nil)
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

			if success, ok := result["success"].(bool); ok && success {
				fmt.Println("DNS provider connection test successful")
			} else {
				fmt.Println("DNS provider connection test failed")
				if msg, ok := result["message"].(string); ok {
					fmt.Printf("Message: %s\n", msg)
				}
			}
			return nil
		},
	}
}

func printProviderTable(providers []map[string]interface{}) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tTYPE\tENDPOINT")
	for _, p := range providers {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			client.GetString(p, "id"),
			client.GetString(p, "name"),
			client.GetString(p, "type"),
			client.GetString(p, "endpoint"))
	}
	w.Flush()
}
