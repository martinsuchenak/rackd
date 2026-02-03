package export

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/paularlott/cli"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/martinsuchenak/rackd/internal/export"
	"github.com/martinsuchenak/rackd/internal/model"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "export",
		Usage: "Export data to CSV or JSON",
		Commands: []*cli.Command{
			DevicesCommand(),
			NetworksCommand(),
			DatacentersCommand(),
			AllCommand(),
		},
	}
}

func DevicesCommand() *cli.Command {
	return &cli.Command{
		Name:  "devices",
		Usage: "Export devices",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "format", Usage: "Output format (json/csv)", DefaultValue: "json"},
			&cli.StringFlag{Name: "output", Usage: "Output file (default: stdout)"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			resp, err := c.DoRequest("GET", "/api/devices", nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var devices []model.Device
			if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			format := export.Format(cmd.GetString("format"))
			output := cmd.GetString("output")

			var writer *os.File
			if output == "" {
				writer = os.Stdout
			} else {
				f, err := os.Create(output)
				if err != nil {
					return fmt.Errorf("failed to create output file: %w", err)
				}
				defer f.Close()
				writer = f
			}

			if err := export.ExportDevices(devices, format, writer); err != nil {
				return fmt.Errorf("export failed: %w", err)
			}

			if output != "" {
				fmt.Fprintf(os.Stderr, "Exported %d devices to %s\n", len(devices), output)
			}

			return nil
		},
	}
}

func NetworksCommand() *cli.Command {
	return &cli.Command{
		Name:  "networks",
		Usage: "Export networks",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "format", Usage: "Output format (json/csv)", DefaultValue: "json"},
			&cli.StringFlag{Name: "output", Usage: "Output file (default: stdout)"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			resp, err := c.DoRequest("GET", "/api/networks", nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var networks []model.Network
			if err := json.NewDecoder(resp.Body).Decode(&networks); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			format := export.Format(cmd.GetString("format"))
			output := cmd.GetString("output")

			var writer *os.File
			if output == "" {
				writer = os.Stdout
			} else {
				f, err := os.Create(output)
				if err != nil {
					return fmt.Errorf("failed to create output file: %w", err)
				}
				defer f.Close()
				writer = f
			}

			if err := export.ExportNetworks(networks, format, writer); err != nil {
				return fmt.Errorf("export failed: %w", err)
			}

			if output != "" {
				fmt.Fprintf(os.Stderr, "Exported %d networks to %s\n", len(networks), output)
			}

			return nil
		},
	}
}

func DatacentersCommand() *cli.Command {
	return &cli.Command{
		Name:  "datacenters",
		Usage: "Export datacenters",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "format", Usage: "Output format (json/csv)", DefaultValue: "json"},
			&cli.StringFlag{Name: "output", Usage: "Output file (default: stdout)"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			resp, err := c.DoRequest("GET", "/api/datacenters", nil)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return client.HandleError(resp)
			}

			var datacenters []model.Datacenter
			if err := json.NewDecoder(resp.Body).Decode(&datacenters); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}

			format := export.Format(cmd.GetString("format"))
			output := cmd.GetString("output")

			var writer *os.File
			if output == "" {
				writer = os.Stdout
			} else {
				f, err := os.Create(output)
				if err != nil {
					return fmt.Errorf("failed to create output file: %w", err)
				}
				defer f.Close()
				writer = f
			}

			if err := export.ExportDatacenters(datacenters, format, writer); err != nil {
				return fmt.Errorf("export failed: %w", err)
			}

			if output != "" {
				fmt.Fprintf(os.Stderr, "Exported %d datacenters to %s\n", len(datacenters), output)
			}

			return nil
		},
	}
}

func AllCommand() *cli.Command {
	return &cli.Command{
		Name:  "all",
		Usage: "Export all data (devices, networks, datacenters)",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "format", Usage: "Output format (json only)", DefaultValue: "json"},
			&cli.StringFlag{Name: "output", Usage: "Output file (default: stdout)"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.GetString("format") != "json" {
				return fmt.Errorf("only JSON format is supported for 'all' export")
			}

			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			// Fetch all data
			var devices []model.Device
			var networks []model.Network
			var datacenters []model.Datacenter

			// Get devices
			resp, err := c.DoRequest("GET", "/api/devices", nil)
			if err != nil {
				return err
			}
			if resp.StatusCode == http.StatusOK {
				json.NewDecoder(resp.Body).Decode(&devices)
			}
			resp.Body.Close()

			// Get networks
			resp, err = c.DoRequest("GET", "/api/networks", nil)
			if err != nil {
				return err
			}
			if resp.StatusCode == http.StatusOK {
				json.NewDecoder(resp.Body).Decode(&networks)
			}
			resp.Body.Close()

			// Get datacenters
			resp, err = c.DoRequest("GET", "/api/datacenters", nil)
			if err != nil {
				return err
			}
			if resp.StatusCode == http.StatusOK {
				json.NewDecoder(resp.Body).Decode(&datacenters)
			}
			resp.Body.Close()

			// Create combined export
			data := map[string]interface{}{
				"devices":     devices,
				"networks":    networks,
				"datacenters": datacenters,
				"exported_at": ctx.Value("time"),
			}

			output := cmd.GetString("output")
			var writer *os.File
			if output == "" {
				writer = os.Stdout
			} else {
				f, err := os.Create(output)
				if err != nil {
					return fmt.Errorf("failed to create output file: %w", err)
				}
				defer f.Close()
				writer = f
			}

			encoder := json.NewEncoder(writer)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(data); err != nil {
				return fmt.Errorf("export failed: %w", err)
			}

			if output != "" {
				fmt.Fprintf(os.Stderr, "Exported %d devices, %d networks, %d datacenters to %s\n",
					len(devices), len(networks), len(datacenters), output)
			}

			return nil
		},
	}
}
