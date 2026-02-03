package importcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/paularlott/cli"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/martinsuchenak/rackd/internal/importdata"
	"github.com/martinsuchenak/rackd/internal/model"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "import",
		Usage: "Import data from CSV or JSON",
		Commands: []*cli.Command{
			DevicesCommand(),
			NetworksCommand(),
			DatacentersCommand(),
		},
	}
}

func DevicesCommand() *cli.Command {
	return &cli.Command{
		Name:  "devices",
		Usage: "Import devices from file",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "file", Usage: "Input file (JSON or CSV)", Required: true},
			&cli.StringFlag{Name: "format", Usage: "Input format (json/csv, auto-detected if omitted)"},
			&cli.BoolFlag{Name: "dry-run", Usage: "Validate without importing"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			filename := cmd.GetString("file")
			format := cmd.GetString("format")
			dryRun := cmd.GetBool("dry-run")

			// Auto-detect format from extension
			if format == "" {
				ext := strings.ToLower(filepath.Ext(filename))
				if ext == ".json" {
					format = "json"
				} else if ext == ".csv" {
					format = "csv"
				} else {
					return fmt.Errorf("cannot auto-detect format, please specify --format")
				}
			}

			// Read file
			f, err := os.Open(filename)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer f.Close()

			// Parse devices
			var devices []model.Device
			if format == "json" {
				devices, err = importdata.ImportDevicesJSON(f)
			} else {
				devices, err = importdata.ImportDevicesCSV(f)
			}
			if err != nil {
				return fmt.Errorf("failed to parse file: %w", err)
			}

			fmt.Printf("Parsed %d devices from %s\n", len(devices), filename)

			if dryRun {
				fmt.Println("Dry run - no changes made")
				return nil
			}

			// Import devices
			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			result := importdata.ImportResult{Total: len(devices)}

			for _, device := range devices {
				// Try to create device
				body, _ := json.Marshal(device)
				resp, err := c.DoRequest("POST", "/api/devices", bytes.NewReader(body))
				if err != nil {
					result.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", device.Name, err))
					continue
				}
				resp.Body.Close()

				if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
					result.Created++
				} else {
					result.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("%s: HTTP %d", device.Name, resp.StatusCode))
				}
			}

			// Print results
			fmt.Printf("\nImport complete:\n")
			fmt.Printf("  Total:   %d\n", result.Total)
			fmt.Printf("  Created: %d\n", result.Created)
			fmt.Printf("  Failed:  %d\n", result.Failed)

			if len(result.Errors) > 0 {
				fmt.Printf("\nErrors:\n")
				for _, err := range result.Errors {
					fmt.Printf("  - %s\n", err)
				}
			}

			if result.Failed > 0 {
				return fmt.Errorf("import completed with %d errors", result.Failed)
			}

			return nil
		},
	}
}

func NetworksCommand() *cli.Command {
	return &cli.Command{
		Name:  "networks",
		Usage: "Import networks from file",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "file", Usage: "Input file (JSON or CSV)", Required: true},
			&cli.StringFlag{Name: "format", Usage: "Input format (json/csv, auto-detected if omitted)"},
			&cli.BoolFlag{Name: "dry-run", Usage: "Validate without importing"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			filename := cmd.GetString("file")
			format := cmd.GetString("format")
			dryRun := cmd.GetBool("dry-run")

			if format == "" {
				ext := strings.ToLower(filepath.Ext(filename))
				if ext == ".json" {
					format = "json"
				} else if ext == ".csv" {
					format = "csv"
				} else {
					return fmt.Errorf("cannot auto-detect format, please specify --format")
				}
			}

			f, err := os.Open(filename)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer f.Close()

			var networks []model.Network
			if format == "json" {
				networks, err = importdata.ImportNetworksJSON(f)
			} else {
				networks, err = importdata.ImportNetworksCSV(f)
			}
			if err != nil {
				return fmt.Errorf("failed to parse file: %w", err)
			}

			fmt.Printf("Parsed %d networks from %s\n", len(networks), filename)

			if dryRun {
				fmt.Println("Dry run - no changes made")
				return nil
			}

			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			result := importdata.ImportResult{Total: len(networks)}

			for _, network := range networks {
				body, _ := json.Marshal(network)
				resp, err := c.DoRequest("POST", "/api/networks", bytes.NewReader(body))
				if err != nil {
					result.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", network.Name, err))
					continue
				}
				resp.Body.Close()

				if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
					result.Created++
				} else {
					result.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("%s: HTTP %d", network.Name, resp.StatusCode))
				}
			}

			fmt.Printf("\nImport complete:\n")
			fmt.Printf("  Total:   %d\n", result.Total)
			fmt.Printf("  Created: %d\n", result.Created)
			fmt.Printf("  Failed:  %d\n", result.Failed)

			if len(result.Errors) > 0 {
				fmt.Printf("\nErrors:\n")
				for _, err := range result.Errors {
					fmt.Printf("  - %s\n", err)
				}
			}

			if result.Failed > 0 {
				return fmt.Errorf("import completed with %d errors", result.Failed)
			}

			return nil
		},
	}
}

func DatacentersCommand() *cli.Command {
	return &cli.Command{
		Name:  "datacenters",
		Usage: "Import datacenters from file",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "file", Usage: "Input file (JSON or CSV)", Required: true},
			&cli.StringFlag{Name: "format", Usage: "Input format (json/csv, auto-detected if omitted)"},
			&cli.BoolFlag{Name: "dry-run", Usage: "Validate without importing"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			filename := cmd.GetString("file")
			format := cmd.GetString("format")
			dryRun := cmd.GetBool("dry-run")

			if format == "" {
				ext := strings.ToLower(filepath.Ext(filename))
				if ext == ".json" {
					format = "json"
				} else if ext == ".csv" {
					format = "csv"
				} else {
					return fmt.Errorf("cannot auto-detect format, please specify --format")
				}
			}

			f, err := os.Open(filename)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer f.Close()

			var datacenters []model.Datacenter
			if format == "json" {
				datacenters, err = importdata.ImportDatacentersJSON(f)
			} else {
				datacenters, err = importdata.ImportDatacentersCSV(f)
			}
			if err != nil {
				return fmt.Errorf("failed to parse file: %w", err)
			}

			fmt.Printf("Parsed %d datacenters from %s\n", len(datacenters), filename)

			if dryRun {
				fmt.Println("Dry run - no changes made")
				return nil
			}

			cfg := client.LoadConfig()
			c := client.NewClient(cfg)

			result := importdata.ImportResult{Total: len(datacenters)}

			for _, datacenter := range datacenters {
				body, _ := json.Marshal(datacenter)
				resp, err := c.DoRequest("POST", "/api/datacenters", bytes.NewReader(body))
				if err != nil {
					result.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", datacenter.Name, err))
					continue
				}
				resp.Body.Close()

				if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
					result.Created++
				} else {
					result.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("%s: HTTP %d", datacenter.Name, resp.StatusCode))
				}
			}

			fmt.Printf("\nImport complete:\n")
			fmt.Printf("  Total:   %d\n", result.Total)
			fmt.Printf("  Created: %d\n", result.Created)
			fmt.Printf("  Failed:  %d\n", result.Failed)

			if len(result.Errors) > 0 {
				fmt.Printf("\nErrors:\n")
				for _, err := range result.Errors {
					fmt.Printf("  - %s\n", err)
				}
			}

			if result.Failed > 0 {
				return fmt.Errorf("import completed with %d errors", result.Failed)
			}

			return nil
		},
	}
}
