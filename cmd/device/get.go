package device

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
		Usage: "Get a device by ID",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "Device ID", Required: true},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json/yaml)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)
			deviceID := cmd.GetString("id")

			resp, err := c.DoRequest("GET", "/api/devices/"+deviceID, nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var device map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&device); err != nil {
				return err
			}

			switch cmd.GetString("output") {
			case "json":
				client.PrintJSON(device)
			case "yaml":
				client.PrintYAML(device)
			default:
				printDeviceDetail(device)
			}
			return nil
		},
	}
}

func printDeviceDetail(d map[string]interface{}) {
	fmt.Printf("ID:          %s\n", getString(d, "id"))
	fmt.Printf("Name:        %s\n", getString(d, "name"))
	fmt.Printf("Description: %s\n", getString(d, "description"))
	fmt.Printf("Make/Model:  %s\n", getString(d, "make_model"))
	fmt.Printf("OS:          %s\n", getString(d, "os"))
	fmt.Printf("Datacenter:  %s\n", getString(d, "datacenter_id"))
	fmt.Printf("Location:    %s\n", getString(d, "location"))
	fmt.Printf("Username:    %s\n", getString(d, "username"))

	if tags, ok := d["tags"].([]interface{}); ok && len(tags) > 0 {
		fmt.Print("Tags:        ")
		for i, t := range tags {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(t)
		}
		fmt.Println()
	}

	if addrs, ok := d["addresses"].([]interface{}); ok && len(addrs) > 0 {
		fmt.Println("Addresses:")
		for _, a := range addrs {
			if addr, ok := a.(map[string]interface{}); ok {
				fmt.Printf("  - %s", getString(addr, "ip"))
				if port, ok := addr["port"].(float64); ok && port > 0 {
					fmt.Printf(":%d", int(port))
				}
				if t := getString(addr, "type"); t != "" {
					fmt.Printf(" (%s)", t)
				}
				fmt.Println()
			}
		}
	}
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
