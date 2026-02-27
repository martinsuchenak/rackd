package conflict

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
		Usage: "Get a conflict by ID",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Conflict ID", Required: true},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)
			conflictID := cmd.GetString("id")

			resp, err := c.DoRequest("GET", "/api/conflicts/"+conflictID, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var conflict map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&conflict); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(conflict)
			case "yaml":
				client.PrintYAML(conflict)
			default:
				printConflictDetail(conflict)
			}
			return nil
		},
	}
}

func printConflictDetail(c map[string]interface{}) {
	fmt.Printf("ID:          %s\n", getString(c, "id"))
	fmt.Printf("Type:         %s\n", getString(c, "type"))
	fmt.Printf("Status:       %s\n", getString(c, "status"))
	fmt.Printf("Description:  %s\n", getString(c, "description"))

	if ip := getString(c, "ip_address"); ip != "" {
		fmt.Printf("IP Address:   %s\n", ip)
	}

	if deviceIDs, ok := c["device_ids"].([]interface{}); ok && len(deviceIDs) > 0 {
		fmt.Print("Devices:      ")
		for i, id := range deviceIDs {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%v", id)
		}
		fmt.Println()
	}

	if deviceNames, ok := c["device_names"].([]interface{}); ok && len(deviceNames) > 0 {
		fmt.Print("Device Names: ")
		for i, name := range deviceNames {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%v", name)
		}
		fmt.Println()
	}

	if networkIDs, ok := c["network_ids"].([]interface{}); ok && len(networkIDs) > 0 {
		fmt.Print("Networks:     ")
		for i, id := range networkIDs {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%v", id)
		}
		fmt.Println()
	}

	if subnets, ok := c["subnets"].([]interface{}); ok && len(subnets) > 0 {
		fmt.Print("Subnets:      ")
		for i, subnet := range subnets {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%v", subnet)
		}
		fmt.Println()
	}

	fmt.Printf("Detected At:  %s\n", getString(c, "detected_at"))

	if resolvedAt := getString(c, "resolved_at"); resolvedAt != "" {
		fmt.Printf("Resolved At:  %s\n", resolvedAt)
	}

	if resolvedBy := getString(c, "resolved_by"); resolvedBy != "" {
		fmt.Printf("Resolved By:  %s\n", resolvedBy)
	}

	if notes := getString(c, "notes"); notes != "" {
		fmt.Printf("Notes:        %s\n", notes)
	}
}
