package device

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/paularlott/cli"
)

func AddCommand() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "Add a new device",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "name", Usage: "Device name", Required: true},
			&cli.StringFlag{Name: "description", Usage: "Device description"},
			&cli.StringFlag{Name: "make-model", Usage: "Device make and model"},
			&cli.StringFlag{Name: "os", Usage: "Operating system"},
			&cli.StringFlag{Name: "datacenter", Usage: "Datacenter ID"},
			&cli.StringFlag{Name: "username", Usage: "Login username"},
			&cli.StringFlag{Name: "location", Usage: "Physical location"},
			&cli.StringFlag{Name: "tags", Usage: "Tags (comma-separated)"},
			&cli.StringFlag{Name: "addresses", Usage: "IP addresses (ip:port:type,...)"},
			&cli.StringFlag{Name: "domains", Usage: "Domain names (comma-separated)"},
			&cli.StringFlag{Name: "input", Usage: "Read from file (JSON)"},
			&cli.StringFlag{Name: "output", Usage: "Output format (table/json)", DefaultValue: "table"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			var device model.Device
			if input := cmd.GetString("input"); input != "" {
				data, err := os.ReadFile(input)
				if err != nil {
					return fmt.Errorf("failed to read input file: %w", err)
				}
				if err := json.Unmarshal(data, &device); err != nil {
					return fmt.Errorf("failed to parse input: %w", err)
				}
			} else {
				device = parseDeviceFlags(cmd)
			}

			resp, err := c.DoRequest("POST", "/api/devices", device)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
				return client.HandleError(resp)
			}

			var created map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
				return err
			}

			if cmd.GetString("output") == "json" {
				client.PrintJSON(created)
			} else {
				fmt.Printf("Device created successfully\n")
				fmt.Printf("ID: %s\n", created["id"])
				fmt.Printf("Name: %s\n", created["name"])
			}
			return nil
		},
	}
}

func parseDeviceFlags(cmd *cli.Command) model.Device {
	device := model.Device{
		Name:         cmd.GetString("name"),
		Description:  cmd.GetString("description"),
		MakeModel:    cmd.GetString("make-model"),
		OS:           cmd.GetString("os"),
		DatacenterID: cmd.GetString("datacenter"),
		Username:     cmd.GetString("username"),
		Location:     cmd.GetString("location"),
	}

	if tags := cmd.GetString("tags"); tags != "" {
		device.Tags = strings.Split(tags, ",")
	}
	if addrs := cmd.GetString("addresses"); addrs != "" {
		device.Addresses = parseAddresses(addrs)
	}
	if domains := cmd.GetString("domains"); domains != "" {
		device.Domains = strings.Split(domains, ",")
	}

	return device
}

func parseAddresses(addrs string) []model.Address {
	var addresses []model.Address
	for _, addrStr := range strings.Split(addrs, ",") {
		parts := strings.Split(strings.TrimSpace(addrStr), ":")
		if len(parts) < 1 || parts[0] == "" {
			continue
		}
		addr := model.Address{IP: parts[0], Type: "ipv4"}
		if len(parts) > 1 {
			if p, err := strconv.Atoi(parts[1]); err == nil && p > 0 {
				addr.Port = &p
			}
		}
		if len(parts) > 2 {
			addr.Type = parts[2]
		}
		addresses = append(addresses, addr)
	}
	return addresses
}
