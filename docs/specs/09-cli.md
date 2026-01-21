# CLI Commands

This document covers the command-line interface structure, cmd/client package specification, error handling patterns, and complete examples for all subcommands.

## Main Entry Point

**File**: `main.go`

```go
package main

import (
    "context"
    "os"

    "github.com/paularlott/cli"
    "github.com/martinsuchenak/rackd/cmd/datacenter"
    "github.com/martinsuchenak/rackd/cmd/device"
    "github.com/martinsuchenak/rackd/cmd/discovery"
    "github.com/martinsuchenak/rackd/cmd/network"
    "github.com/martinsuchenak/rackd/cmd/server"
)

var (
    version = "dev"
    commit  = "unknown"
    date    = "unknown"
)

func main() {
    app := &cli.Command{
        Name:    "rackd",
        Usage:   "Device inventory and IPAM management",
        Version: version,
        SubCommands: []*cli.Command{
            server.Command(),
            device.Command(),
            network.Command(),
            datacenter.Command(),
            discovery.Command(),
            {
                Name:  "version",
                Usage: "Show version information",
                Run: func(ctx context.Context, cmd *cli.Command) error {
                    cmd.Printf("Version: %s\nCommit: %s\nBuilt: %s\n", version, commit, date)
                    return nil
                },
            },
        },
    }

    if err := app.Run(context.Background(), os.Args); err != nil {
        os.Exit(1)
    }
}
```

## Server Command

**File**: `cmd/server/server.go`

```go
package server

import (
    "context"

    "github.com/paularlott/cli"
    "github.com/martinsuchenak/rackd/internal/config"
    "github.com/martinsuchenak/rackd/internal/log"
    "github.com/martinsuchenak/rackd/internal/server"
    "github.com/martinsuchenak/rackd/internal/storage"
)

func Command() *cli.Command {
    return &cli.Command{
        Name:  "server",
        Usage: "Start the HTTP/MCP server",
        Flags: []*cli.Flag{
            {Name: "data-dir", Usage: "Data directory", Default: "./data"},
            {Name: "listen-addr", Usage: "Listen address", Default: ":8080"},
            {Name: "api-auth-token", Usage: "API authentication token"},
            {Name: "mcp-auth-token", Usage: "MCP authentication token"},
            {Name: "log-level", Usage: "Log level (trace/debug/info/warn/error)", Default: "info"},
            {Name: "log-format", Usage: "Log format (text/json)", Default: "text"},
            {Name: "discovery-enabled", Usage: "Enable network discovery", Default: "true"},
            {Name: "discovery-interval", Usage: "Discovery scan interval", Default: "24h"},
        },
        Run: func(ctx context.Context, cmd *cli.Command) error {
            cfg := config.Load()

            // Initialize logging
            log.Init(cfg.LogLevel, cfg.LogFormat)

            // Initialize storage
            store, err := storage.NewExtendedStorage(cfg.DataDir, "sqlite", "")
            if err != nil {
                return err
            }

            // Run server
            return server.Run(cfg, store)
        },
    }
}
```

## Device Commands

**File**: `cmd/device/device.go`

```go
package device

import "github.com/paularlott/cli"

func Command() *cli.Command {
    return &cli.Command{
        Name:  "device",
        Usage: "Device management commands",
        SubCommands: []*cli.Command{
            ListCommand(),
            GetCommand(),
            AddCommand(),
            UpdateCommand(),
            DeleteCommand(),
        },
    }
}
```

## Device List Command

**File**: `cmd/device/list.go`

```go
package device

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"

    "github.com/paularlott/cli"
    "github.com/martinsuchenak/rackd/cmd/client"
)

func ListCommand() *cli.Command {
    return &cli.Command{
        Name:  "list",
        Usage: "List all devices",
        Flags: []*cli.Flag{
            {Name: "query", Short: "q", Usage: "Search query"},
            {Name: "tags", Short: "t", Usage: "Filter by tags (comma-separated)"},
            {Name: "datacenter", Short: "d", Usage: "Filter by datacenter ID"},
            {Name: "network", Short: "n", Usage: "Filter by network ID"},
            {Name: "limit", Short: "l", Usage: "Limit number of results"},
            {Name: "output", Short: "o", Usage: "Output format (table/json/yaml)", Default: "table"},
        },
        Run: func(ctx context.Context, cmd *cli.Command) error {
            cfg := client.LoadConfig()

            // Build query parameters
            params := make(map[string]string)
            if query := cmd.GetString("query"); query != "" {
                params["q"] = query
            }
            if tags := cmd.GetString("tags"); tags != "" {
                params["tags"] = tags
            }
            if dc := cmd.GetString("datacenter"); dc != "" {
                params["datacenter_id"] = dc
            }
            if network := cmd.GetString("network"); network != "" {
                params["network_id"] = network
            }
            if limit := cmd.GetInt("limit"); limit > 0 {
                params["limit"] = fmt.Sprintf("%d", limit)
            }

            url := fmt.Sprintf("%s/api/devices?%s", cfg.ServerURL, buildQueryString(params))

            resp, err := client.DoRequest("GET", url, cfg.Token, nil)
            if err != nil {
                return err
            }
            defer resp.Body.Close()

            if resp.StatusCode != http.StatusOK {
                return client.HandleError(resp)
            }

            var devices []map[string]interface{}
            if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
                return err
            }

            switch cmd.GetString("output") {
            case "json":
                return client.PrintJSON(devices)
            case "yaml":
                return client.PrintYAML(devices)
            default:
                return client.PrintDeviceTable(devices)
            }
        },
    }
}

func buildQueryString(params map[string]string) string {
    var builder strings.Builder
    for key, value := range params {
        if builder.Len() > 0 {
            builder.WriteString("&")
        }
        builder.WriteString(fmt.Sprintf("%s=%s", key, value))
    }
    return builder.String()
}
```

## Device Get Command

**File**: `cmd/device/get.go`

```go
package device

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/paularlott/cli"
    "github.com/martinsuchenak/rackd/cmd/client"
)

func GetCommand() *cli.Command {
    return &cli.Command{
        Name:  "get",
        Usage: "Get a device by ID",
        Flags: []*cli.Flag{
            {Name: "id", Short: "i", Usage: "Device ID", Required: true},
            {Name: "output", Short: "o", Usage: "Output format (table/json/yaml)", Default: "table"},
        },
        Run: func(ctx context.Context, cmd *cli.Command) error {
            cfg := client.LoadConfig()
            deviceID := cmd.GetString("id")

            url := fmt.Sprintf("%s/api/devices/%s", cfg.ServerURL, deviceID)

            resp, err := client.DoRequest("GET", url, cfg.Token, nil)
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
                return client.PrintJSON(device)
            case "yaml":
                return client.PrintYAML(device)
            default:
                return client.PrintDeviceDetail(device)
            }
        },
    }
}
```

## Device Add Command

**File**: `cmd/device/add.go`

```go
package device

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "strings"

    "github.com/paularlott/cli"
    "github.com/martinsuchenak/rackd/cmd/client"
    "github.com/martinsuchenak/rackd/internal/model"
)

func AddCommand() *cli.Command {
    return &cli.Command{
        Name:  "add",
        Usage: "Add a new device",
        Flags: []*cli.Flag{
            {Name: "name", Short: "n", Usage: "Device name", Required: true},
            {Name: "description", Short: "d", Usage: "Device description"},
            {Name: "make-model", Short: "m", Usage: "Device make and model"},
            {Name: "os", Short: "o", Usage: "Operating system"},
            {Name: "datacenter", Short: "D", Usage: "Datacenter ID"},
            {Name: "username", Short: "u", Usage: "Login username"},
            {Name: "location", Short: "l", Usage: "Physical location"},
            {Name: "tags", Short: "t", Usage: "Tags (comma-separated)"},
            {Name: "addresses", Short: "a", Usage: "IP addresses (comma-separated: ip:port,type,label)"},
            {Name: "domains", Short: "D", Usage: "Domain names (comma-separated)"},
            {Name: "input", Short: "i", Usage: "Read from stdin or file"},
            {Name: "output", Short: "o", Usage: "Output format (table/json)", Default: "table"},
        },
        Run: func(ctx context.Context, cmd *cli.Command) error {
            cfg := client.LoadConfig()

            // Parse input
            var device model.Device
            var err error

            if cmd.GetString("input") != "" {
                data, err := os.ReadFile(cmd.GetString("input"))
                if err != nil {
                    return fmt.Errorf("failed to read input file: %w", err)
                }
                err = json.Unmarshal(data, &device)
            } else {
                device = parseDeviceFlags(cmd)
            }

            if err != nil {
                return fmt.Errorf("failed to parse device: %w", err)
            }

            url := fmt.Sprintf("%s/api/devices", cfg.ServerURL)

            resp, err := client.DoRequest("POST", url, cfg.Token, device)
            if err != nil {
                return err
            }
            defer resp.Body.Close()

            if resp.StatusCode != http.StatusOK {
                return client.HandleError(resp)
            }

            var createdDevice map[string]interface{}
            if err := json.NewDecoder(resp.Body).Decode(&createdDevice); err != nil {
                return err
            }

            switch cmd.GetString("output") {
            case "json":
                return client.PrintJSON(createdDevice)
            default:
                fmt.Printf("Device created successfully\n")
                fmt.Printf("ID: %s\n", createdDevice["id"])
                fmt.Printf("Name: %s\n", createdDevice["name"])
                return nil
            }
        },
    }
}

func parseDeviceFlags(cmd *cli.Command) model.Device {
    device := model.Device{
        Name:        cmd.GetString("name"),
        Description: cmd.GetString("description"),
        MakeModel:  cmd.GetString("make-model"),
        OS:          cmd.GetString("os"),
        DatacenterID: cmd.GetString("datacenter"),
        Username:     cmd.GetString("username"),
        Location:     cmd.GetString("location"),
    }

    if tags := cmd.GetString("tags"); tags != "" {
        device.Tags = strings.Split(tags, ",")
    }

    if addresses := cmd.GetString("addresses"); addresses != "" {
        device.Addresses = parseAddresses(addresses)
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
        if len(parts) < 2 {
            continue
        }
        addr := model.Address{
            IP:   strings.TrimSpace(parts[0]),
            Type: "ipv4",
        }
        if len(parts) > 2 {
            addr.Port, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
            addr.Type = strings.TrimSpace(parts[2])
        }
        addresses = append(addresses, addr)
    }
    return addresses
}
```

## Device Update Command

**File**: `cmd/device/update.go`

```go
package device

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "os"

    "github.com/paularlott/cli"
    "github.com/martinsuchenak/rackd/cmd/client"
    "github.com/martinsuchenak/rackd/internal/model"
)

func UpdateCommand() *cli.Command {
    return &cli.Command{
        Name:  "update",
        Usage: "Update a device",
        Flags: []*cli.Flag{
            {Name: "id", Short: "i", Usage: "Device ID", Required: true},
            {Name: "name", Short: "n", Usage: "Device name"},
            {Name: "description", Short: "d", Usage: "Device description"},
            {Name: "make-model", Short: "m", Usage: "Device make and model"},
            {Name: "os", Short: "o", Usage: "Operating system"},
            {Name: "datacenter", Short: "D", Usage: "Datacenter ID"},
            {Name: "username", Short: "u", Usage: "Login username"},
            {Name: "location", Short: "l", Usage: "Physical location"},
            {Name: "tags", Short: "t", Usage: "Tags (comma-separated, use + to add, - to remove)"},
            {Name: "addresses", Short: "a", Usage: "IP addresses (comma-separated: ip:port,type,label, set to empty to remove)"},
            {Name: "domains", Short: "D", Usage: "Domain names (comma-separated)"},
            {Name: "input", Short: "i", Usage: "Read from stdin or file"},
            {Name: "output", Short: "o", Usage: "Output format (table/json)", Default: "table"},
        },
        Run: func(ctx context.Context, cmd *cli.Command) error {
            cfg := client.LoadConfig()

            // Get existing device
            deviceID := cmd.GetString("id")
            url := fmt.Sprintf("%s/api/devices/%s", cfg.ServerURL, deviceID)

            resp, err := client.DoRequest("GET", url, cfg.Token, nil)
            if err != nil {
                return err
            }
            defer resp.Body.Close()

            if resp.StatusCode != http.StatusOK {
                return client.HandleError(resp)
            }

            var existingDevice map[string]interface{}
            if err := json.NewDecoder(resp.Body).Decode(&existingDevice); err != nil {
                return err
            }

            // Build update
            updates := make(map[string]interface{})
            if name := cmd.GetString("name"); name != "" {
                updates["name"] = name
            }
            if description := cmd.GetString("description"); description != "" {
                updates["description"] = description
            }
            if makeModel := cmd.GetString("make-model"); makeModel != "" {
                updates["make_model"] = makeModel
            }
            if os := cmd.GetString("os"); os != "" {
                updates["os"] = os
            }
            if datacenter := cmd.GetString("datacenter"); datacenter != "" {
                updates["datacenter_id"] = datacenter
            }
            if username := cmd.GetString("username"); username != "" {
                updates["username"] = username
            }
            if location := cmd.GetString("location"); location != "" {
                updates["location"] = location
            }
            if tags := cmd.GetString("tags"); tags != "" {
                updates["tags"] = updateTags(existingDevice["tags"], tags)
            }
            if addresses := cmd.GetString("addresses"); addresses != "" {
                updates["addresses"] = updateAddresses(existingDevice["addresses"], addresses)
            }
            if domains := cmd.GetString("domains"); domains != "" {
                updates["domains"] = strings.Split(domains, ",")
            }

            // Send update
            url = fmt.Sprintf("%s/api/devices/%s", cfg.ServerURL, deviceID)
            resp, err = client.DoRequest("PUT", url, cfg.Token, updates)
            if err != nil {
                return err
            }
            defer resp.Body.Close()

            if resp.StatusCode != http.StatusOK {
                return client.HandleError(resp)
            }

            var updatedDevice map[string]interface{}
            if err := json.NewDecoder(resp.Body).Decode(&updatedDevice); err != nil {
                return err
            }

            switch cmd.GetString("output") {
            case "json":
                return client.PrintJSON(updatedDevice)
            case "yaml":
                return client.PrintYAML(updatedDevice)
            default:
                fmt.Printf("Device updated successfully\n")
                return nil
            }
        },
    }
}

func updateTags(existingTags []string, tagsStr string) []string {
    existing := make(map[string]bool)
    for _, tag := range existingTags {
        existing[tag] = true
    }

    var newTags []string
    tags := strings.Split(tagsStr, ",")
    for _, tag := range tags {
        tag = strings.TrimSpace(tag)
        if strings.HasPrefix(tag, "+") {
            // Add tag
            tagName := strings.TrimPrefix(tag, "+")
            newTags = append(newTags, tagName)
        } else if strings.HasPrefix(tag, "-") {
            // Remove tag
            tagName := strings.TrimPrefix(tag, "-")
            delete(existing, tagName)
        }
    }

    return newTags
}

func updateAddresses(existing []model.Address, addressesStr string) []model.Address {
    if addressesStr == "" {
        return existing
    }

    existingMap := make(map[string]model.Address)
    for _, addr := range existing {
        key := fmt.Sprintf("%s:%d", addr.IP, addr.Port)
        existingMap[key] = addr
    }

    addressStrs := strings.Split(addressesStr, ",")
    var newAddresses []model.Address
    for _, addrStr := range addressStrs {
        parts := strings.Split(strings.TrimSpace(addrStr), ":")
        if len(parts) < 2 {
            continue
        }

        ip := strings.TrimSpace(parts[0])
        port := 0
        addrType := "ipv4"
        label := ""

        if len(parts) > 2 {
            port, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
            addrType = strings.TrimSpace(parts[2])
        }

        if len(parts) > 3 {
            label = strings.TrimSpace(parts[3])
        }

        key := fmt.Sprintf("%s:%d", ip, port)
        if strings.HasPrefix(addrStr, "+") {
            // Add address
            newAddr := model.Address{IP: ip, Port: port, Type: addrType, Label: label}
            newAddresses = append(newAddresses, newAddr)
        } else if strings.HasPrefix(addrStr, "-") {
            // Remove address
            delete(existingMap, key)
        }
    }

    return append(newAddresses, existingMapToSlice(existingMap)...)
}

func existingMapToSlice(m map[string]model.Address) []model.Address {
    addrs := make([]model.Address, 0, len(m))
    for _, addr := range m {
        addrs = append(addrs, addr)
    }
    return addrs
}
```

## Device Delete Command

**File**: `cmd/device/delete.go`

```go
package device

import (
    "context"
    "fmt"
    "net/http"

    "github.com/paularlott/cli"
    "github.com/martinsuchenak/rackd/cmd/client"
)

func DeleteCommand() *cli.Command {
    return &cli.Command{
        Name:  "delete",
        Usage: "Delete a device",
        Flags: []*cli.Flag{
            {Name: "id", Short: "i", Usage: "Device ID", Required: true},
            {Name: "force", Short: "f", Usage: "Skip confirmation"},
        },
        Run: func(ctx context.Context, cmd *cli.Command) error {
            cfg := client.LoadConfig()
            deviceID := cmd.GetString("id")

            // Confirm deletion unless force flag
            if !cmd.GetBool("force") {
                fmt.Printf("Are you sure you want to delete device %s? [y/N]: ", deviceID)
                var confirm string
                fmt.Scanln(&confirm)
                if confirm != "y" && confirm != "Y" {
                    fmt.Println("Deletion cancelled")
                    return nil
                }
            }

            url := fmt.Sprintf("%s/api/devices/%s", cfg.ServerURL, deviceID)

            resp, err := client.DoRequest("DELETE", url, cfg.Token, nil)
            if err != nil {
                return err
            }
            defer resp.Body.Close()

            if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
                return client.HandleError(resp)
            }

            fmt.Printf("Device deleted successfully\n")
            return nil
        },
    }
}
```

## Network Commands

**File**: `cmd/network/network.go`

```go
package network

import "github.com/paularlott/cli"

func Command() *cli.Command {
    return &cli.Command{
        Name:  "network",
        Usage: "Network management commands",
        SubCommands: []*cli.Command{
            ListCommand(),
            GetCommand(),
            AddCommand(),
            DeleteCommand(),
            PoolCommand(),
        },
    }
}

// cmd/network/list.go, get.go, add.go, delete.go, pool.go
// Implementation similar to device commands with appropriate models and endpoints
```

## Datacenter Commands

**File**: `cmd/datacenter/datacenter.go`

```go
package datacenter

import "github.com/paularlott/cli"

func Command() *cli.Command {
    return &cli.Command{
        Name:  "datacenter",
        Usage: "Datacenter management commands",
        SubCommands: []*cli.Command{
            ListCommand(),
            GetCommand(),
            AddCommand(),
            UpdateCommand(),
            DeleteCommand(),
        },
    }
}

// cmd/datacenter/list.go, get.go, add.go, update.go, delete.go
// Implementation similar to device commands with appropriate models and endpoints
```

## Discovery Commands

**File**: `cmd/discovery/discovery.go`

```go
package discovery

import "github.com/paularlott/cli"

func Command() *cli.Command {
    return &cli.Command{
        Name:  "discovery",
        Usage: "Network discovery commands",
        SubCommands: []*cli.Command{
            ScanCommand(),
            ListCommand(),
            PromoteCommand(),
        },
    }
}
```

## Discovery Scan Command

**File**: `cmd/discovery/scan.go`

```go
package discovery

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/paularlott/cli"
    "github.com/martinsuchenak/rackd/cmd/client"
)

func ScanCommand() *cli.Command {
    return &cli.Command{
        Name:  "scan",
        Usage: "Start a network discovery scan",
        Flags: []*cli.Flag{
            {Name: "network", Short: "n", Usage: "Network ID to scan", Required: true},
            {Name: "type", Short: "t", Usage: "Scan type (quick/full/deep)", Default: "full"},
            {Name: "dry-run", Usage: "Show what would be scanned without scanning"},
        },
        Run: func(ctx context.Context, cmd *cli.Command) error {
            cfg := client.LoadConfig()
            networkID := cmd.GetString("network")
            scanType := cmd.GetString("type")

            if cmd.GetBool("dry-run") {
                fmt.Printf("Network: %s\n", networkID)
                fmt.Printf("Scan type: %s\n", scanType)
                fmt.Printf("This would scan network %s with type %s\n", networkID, scanType)
                return nil
            }

            url := fmt.Sprintf("%s/api/discovery/networks/%s/scan", cfg.ServerURL, networkID)

            reqBody := map[string]interface{}{
                "scan_type": scanType,
            }

            resp, err := client.DoRequest("POST", url, cfg.Token, reqBody)
            if err != nil {
                return err
            }
            defer resp.Body.Close()

            if resp.StatusCode != http.StatusCreated {
                return client.HandleError(resp)
            }

            var scan map[string]interface{}
            if err := json.NewDecoder(resp.Body).Decode(&scan); err != nil {
                return err
            }

            fmt.Printf("Discovery scan started\n")
            fmt.Printf("Scan ID: %s\n", scan["id"])
            fmt.Printf("Network: %s\n", scan["network_id"])
            fmt.Printf("Scan type: %s\n", scan["scan_type"])

            return nil
        },
    }
}
```

## Discovery List Command

**File**: `cmd/discovery/list.go`

```go
package discovery

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"

    "github.com/paularlott/cli"
    "github.com/martinsuchenak/rackd/cmd/client"
)

func ListCommand() *cli.Command {
    return &cli.Command{
        Name:  "list",
        Usage: "List discovered devices",
        Flags: []*cli.Flag{
            {Name: "network", Short: "n", Usage: "Filter by network ID"},
            {Name: "status", Short: "s", Usage: "Filter by status (online/offline/unknown)"},
            {Name: "limit", Short: "l", Usage: "Limit number of results"},
            {Name: "output", Short: "o", Usage: "Output format (table/json)", Default: "table"},
        },
        Run: func(ctx context.Context, cmd *cli.Command) error {
            cfg := client.LoadConfig()

            params := make(map[string]string)
            if network := cmd.GetString("network"); network != "" {
                params["network_id"] = network
            }
            if status := cmd.GetString("status"); status != "" {
                params["status"] = status
            }
            if limit := cmd.GetInt("limit"); limit > 0 {
                params["limit"] = fmt.Sprintf("%d", limit)
            }

            url := fmt.Sprintf("%s/api/discovery/devices?%s", cfg.ServerURL, buildQueryString(params))

            resp, err := client.DoRequest("GET", url, cfg.Token, nil)
            if err != nil {
                return err
            }
            defer resp.Body.Close()

            if resp.StatusCode != http.StatusOK {
                return client.HandleError(resp)
            }

            var devices []map[string]interface{}
            if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
                return err
            }

            switch cmd.GetString("output") {
            case "json":
                return client.PrintJSON(devices)
            default:
                return client.PrintDiscoveredDevicesTable(devices)
            }
        },
    }
}
```

## Discovery Promote Command

**File**: `cmd/discovery/promote.go`

```go
package discovery

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/paularlott/cli"
    "github.com/martinsuchenak/rackd/cmd/client"
)

func PromoteCommand() *cli.Command {
    return &cli.Command{
        Name:  "promote",
        Usage: "Promote a discovered device to inventory",
        Flags: []*cli.Flag{
            {Name: "discovered-id", Short: "i", Usage: "Discovered device ID", Required: true},
            {Name: "name", Short: "n", Usage: "Device name", Required: true},
        },
        Run: func(ctx context.Context, cmd *cli.Command) error {
            cfg := client.LoadConfig()
            discoveredID := cmd.GetString("discovered-id")
            name := cmd.GetString("name")

            reqBody := map[string]interface{}{
                "discovered_id": discoveredID,
                "name": name,
            }

            url := fmt.Sprintf("%s/api/discovery/devices/%s/promote", cfg.ServerURL, discoveredID)

            resp, err := client.DoRequest("POST", url, cfg.Token, reqBody)
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

            fmt.Printf("Device promoted successfully\n")
            fmt.Printf("Device ID: %s\n", device["id"])
            fmt.Printf("Name: %s\n", device["name"])

            return nil
        },
    }
}
```

## cmd/client Package Specification

### Configuration File Structure

**File**: `~/.config/rackd/config.json`

```json
{
  "server_url": "http://localhost:8080",
  "token": "your-api-token",
  "timeout": "30s",
  "output": "table",
  "verify_ssl": true
}
```

### Configuration Loader

**File**: `cmd/client/config.go`

```go
package client

import (
    "encoding/json"
    "os"
    "path/filepath"
)

type Config struct {
    ServerURL string `json:"server_url"`
    Token     string `json:"token"`
    Timeout   string `json:"timeout"`
    Output    string `json:"output"`
    VerifySSL bool   `json:"verify_ssl"`
}

var defaultConfig = Config{
    ServerURL: "http://localhost:8080",
    Timeout:  "30s",
    Output:  "table",
    VerifySSL: true,
}

func LoadConfig() *Config {
    // Load from config file
    configPath := filepath.Join(getConfigDir(), "config.json")
    config := defaultConfig

    if data, err := os.ReadFile(configPath); err == nil {
        json.Unmarshal(data, &config)
    }

    // Override with environment variables
    if url := os.Getenv("RACKD_SERVER_URL"); url != "" {
        config.ServerURL = url
    }
    if token := os.Getenv("RACKD_TOKEN"); token != "" {
        config.Token = token
    }

    return &config
}

func getConfigDir() string {
    configDir := os.Getenv("XDG_CONFIG_HOME")
    if configDir == "" {
        configDir = filepath.Join(os.Getenv("HOME"), ".config")
    }
    return filepath.Join(configDir, "rackd")
}

func SaveConfig(cfg *Config) error {
    configPath := filepath.Join(getConfigDir(), "config.json")
    data, err := json.MarshalIndent(cfg, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(configPath, data, 0644)
}
```

### HTTP Client

**File**: `cmd/client/http.go`

```go
package client

import (
    "bytes"
    "context"
    "crypto/tls"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type Client struct {
    serverURL  string
    token     string
    timeout   time.Duration
    verifySSL bool
    httpClient *http.Client
}

func NewClient(cfg *Config) *Client {
    timeout, _ := time.ParseDuration(cfg.Timeout)
    client := &http.Client{
        Timeout: timeout,
    }

    return &Client{
        serverURL:  cfg.ServerURL,
        token:     cfg.Token,
        timeout:   timeout,
        verifySSL: cfg.VerifySSL,
        httpClient: client,
    }
}

func (c *Client) DoRequest(method, path string, body interface{}) (*http.Response, error) {
    var reqBody io.Reader
    if body != nil {
        data, err := json.Marshal(body)
        if err != nil {
            return nil, err
        }
        reqBody = bytes.NewReader(data)
    }

    url := c.serverURL + path
    req, err := http.NewRequest(method, url, reqBody)
    if err != nil {
        return nil, err
    }

    // Add auth header
    if c.token != "" {
        req.Header.Set("Authorization", "Bearer "+c.token)
    }

    return c.httpClient.Do(req)
}
```

### Error Handling

**File**: `cmd/client/errors.go`

```go
package client

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

func (c *Client) HandleError(resp *http.Response) error {
    var errResp struct {
        Error   string `json:"error"`
        Code    string `json:"code"`
        Details string `json:"details,omitempty"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
        return fmt.Errorf("failed to parse error response: %w", err)
    }

    return fmt.Errorf("API error: %s (code: %s)", errResp.Error, errResp.Code)
}

func PrintError(code, message string) {
    fmt.Printf("Error [%s]: %s\n", code, message)
    os.Exit(1)
}
```

### Output Formatters

**File**: `cmd/client/table.go`

```go
package client

import (
    "encoding/json"
    "fmt"
    "os"
    "text/tabwriter"

    "github.com/olekukonko/tablewriter"
)

func PrintDeviceTable(devices []map[string]interface{}) error {
    w := tablewriter.NewWriter(os.Stdout)
    w.SetHeader([]string{"ID", "Name", "Make/Model", "OS", "Datacenter", "Location"})

    for _, device := range devices {
        addresses := device["addresses"].([]interface{})
        addrStr := formatAddresses(addresses)

        w.Append([]string{
            device["id"].(string),
            device["name"].(string),
            device["make_model"].(string),
            device["os"].(string),
            getNestedField(device, "datacenter", "name"),
            getNestedField(device, "location", ""),
            addrStr,
        })
    }

    w.Render()
    return nil
}

func PrintDiscoveredDevicesTable(devices []map[string]interface{}) error {
    w := tablewriter.NewWriter(os.Stdout)
    w.SetHeader([]string{"ID", "IP", "MAC", "Hostname", "Status", "Last Seen"})

    for _, device := range devices {
        w.Append([]string{
            device["id"].(string),
            device["ip"].(string),
            device["mac_address"].(string),
            device["hostname"].(string),
            device["status"].(string),
            device["last_seen"].(string),
        })
    }

    w.Render()
    return nil
}

func formatAddresses(addresses []interface{}) string {
    var builder strings.Builder
    for i, addr := range addresses {
        if i > 0 {
            builder.WriteString("; ")
        }
        addrMap := addr.(map[string]interface{})
        builder.WriteString(fmt.Sprintf("%s:%s",
            addrMap["ip"], addrMap["port"]))
    }
    return builder.String()
}

func getNestedField(m map[string]interface{}, fields ...string) string {
    var current = m
    for _, field := range fields {
        if val, ok := current[field]; ok {
            if str, ok := val.(string); ok {
                return str
            }
        }
        if val, ok := current[field]; ok {
            if sub, ok := val.(map[string]interface{}); ok {
                current = sub
            }
        }
    }
    return ""
}

func PrintJSON(data interface{}) error {
    json.NewEncoder(os.Stdout).Encode(data)
    return nil
}

func PrintYAML(data interface{}) error {
    // Use YAML library or custom formatter
    fmt.Printf("---\n")
    printYAMLMap(data, 0)
    return nil
}

func printYAMLMap(m map[string]interface{}, indent int) {
    for key, value := range m {
        fmt.Printf("%s%s: ", strings.Repeat(" ", indent), key)
        printYAMLValue(value, indent+2)
    }
}

func printYAMLSlice(s []interface{}, indent int) {
    for _, value := range s {
        printYAMLValue(value, indent)
    }
}

func printYAMLValue(v interface{}, indent int) {
    switch val := v.(type) {
    case string:
        fmt.Printf("'%s'\n", val)
    case []interface{}:
        printYAMLSlice(val, indent)
    case map[string]interface{}:
        printYAMLMap(val.(map[string]interface{}), indent)
    case bool:
        fmt.Printf("%t\n", val)
    case float64:
        fmt.Printf("%v\n", val)
    case float32:
        fmt.Printf("%v\n", val)
    case int:
        fmt.Printf("%d\n", val)
    case nil:
        fmt.Printf("null\n")
    default:
        fmt.Printf("%v\n", val)
    }
}
```

### Exit Codes

| Exit Code | Meaning | When Used |
|-----------|----------|-----------|
| 0 | Success | Command completed successfully |
| 1 | Generic Error | Unclassified error occurred |
| 2 | Invalid Usage | Wrong arguments, invalid flags, missing required arguments |
| 3 | Network Error | Network connectivity issues (connection refused, timeout) |
| 4 | Authentication Error | Failed authentication (invalid token, missing token) |
| 5 | Server Error | Server returned 5xx error |

### Error Display Examples

```go
// Display errors in a user-friendly way
func DisplayError(err error) {
    var errCode string
    var message string

    switch e := err.(type) {
    case *client.APIError:
        errCode = e.Code
        message = e.Message
    case *net.OpError:
        errCode = "NETWORK_ERROR"
        message = "Network error: " + e.Error()
    case *json.SyntaxError:
        errCode = "INVALID_INPUT"
        message = "Invalid input: " + e.Error()
    default:
        errCode = "UNKNOWN_ERROR"
        message = err.Error()
    }

    fmt.Printf("Error: %s\n", message)
    fmt.Printf("Code: %s\n", errCode)
    os.Exit(getExitCode(errCode))
}

func getExitCode(code string) int {
    switch code {
    case "NETWORK_ERROR":
        return 3
    case "INVALID_INPUT":
        return 2
    case "UNAUTHORIZED":
        return 4
    case "SERVER_ERROR":
        return 5
    default:
        return 1
    }
}
```

### Retry Logic Examples

```go
func (c *Client) DoRequestWithRetry(method, path string, body interface{}, maxRetries int) (*http.Response, error) {
    var lastErr error
    for i := 0; i < maxRetries; i++ {
        resp, err := c.DoRequest(method, path, body)
        if err != nil {
            lastErr = err
            // Check if error is retryable
            if !isRetryableError(err) {
                return resp, err
            }
            // Exponential backoff
            time.Sleep(time.Duration(i*i) * time.Second)
            continue
        }

        // Check for success
        if resp.StatusCode >= 200 && resp.StatusCode < 300 {
            return resp, nil
        }

        // Check for rate limit
        if resp.StatusCode == 429 {
            time.Sleep(c.calculateRetryAfter(resp.Header))
        }
    }

    return nil, lastErr
}

func isRetryableError(err error) bool {
    // Network errors are retryable
    if _, ok := err.(*net.OpError); ok {
        return true
    }
    // Timeout errors are retryable
    if err == context.DeadlineExceeded {
        return true
    }
    return false
}
```

### Complete CLI Examples

**Listing Devices:**
```bash
# List all devices
rackd device list

# Filter by tag
rackd device list --tags web,prod

# Filter by datacenter
rackd device list --datacenter dc-001

# Search devices
rackd device list --query "server"

# JSON output
rackd device list --output json
```

**Getting Device Details:**
```bash
# Get device by ID
rackd device get --id dev-001

# Output in YAML format
rackd device get --id dev-001 --output yaml
```

**Adding Devices:**
```bash
# Add device with required fields
rackd device add --name "server-01" --os "Ubuntu 22.04" --datacenter dc-001

# Add device with addresses
rackd device add --name "server-01" \
  --addresses "192.168.1.10:22:ipv4:management,192.168.1.11:80:ipv4:data"

# Add device with tags
rackd device add --name "server-01" --tags web,prod,database

# Add device from JSON file
rackd device add --input device.json

# Add device from stdin
echo '{"name":"server-02","os":"Ubuntu 24.04"}' | rackd device add --input -
```

**Updating Devices:**
```bash
# Update device name
rackd device update --id dev-001 --name "web-server-01"

# Update with multiple fields
rackd device update --id dev-001 \
  --name "web-server-01" \
  --description "Primary web server" \
  --os "Ubuntu 24.04"

# Add tags
rackd device update --id dev-001 --tags "+web,+prod,+api"

# Remove tags
rackd device update --id dev-001 --tags "-old-tag"

# Update addresses
rackd device update --id dev-001 --addresses "192.168.1.20:22:ipv4:management"

# Clear addresses
rackd device update --id dev-001 --addresses ""

# Set addresses from JSON
rackd device update --id dev-001 --input update.json
```

**Deleting Devices:**
```bash
# Delete device (with confirmation)
rackd device delete --id dev-001

# Delete device (force, no confirmation)
rackd device delete --id dev-001 --force
```

**Network Operations:**
```bash
# List networks
rackd network list

# Get network
rackd network get --id net-001

# Add network
rackd network add --name "production" --subnet "192.168.1.0/24" --datacenter dc-001

# Delete network
rackd network delete --id net-001

# List pools
rackd network pool list --network net-001

# Create pool
rackd network pool add --network net-001 --name "production-pool" \
  --start-ip "192.168.1.100" --end-ip "192.168.1.200"
```

**Datacenter Operations:**
```bash
# List datacenters
rackd datacenter list

# Get datacenter
rackd datacenter get --id dc-001

# Add datacenter
rackd datacenter add --name "Data Center West" --location "San Francisco, CA"

# Update datacenter
rackd datacenter update --id dc-001 --name "DC West"

# Delete datacenter
rackd datacenter delete --id dc-001
```

**Discovery Operations:**
```bash
# List discovered devices
rackd discovery list

# Filter by network
rackd discovery list --network net-001

# Filter by status
rackd discovery list --status online

# Start discovery scan
rackd discovery scan --network net-001 --type full

# Start quick scan
rackd discovery scan --network net-001 --type quick

# Start deep scan
rackd discovery scan --network net-001 --type deep

# Promote discovered device
rackd discovery promote --discovered-id disc-001 --name "web-server-02"

# Dry-run scan
rackd discovery scan --network net-001 --dry-run
```

    if err := app.Run(context.Background(), os.Args); err != nil {
        os.Exit(1)
    }
}
```

## Server Command

**File**: `cmd/server/server.go`

```go
package server

import (
    "context"

    "github.com/paularlott/cli"
    "github.com/martinsuchenak/rackd/internal/config"
    "github.com/martinsuchenak/rackd/internal/log"
    "github.com/martinsuchenak/rackd/internal/server"
    "github.com/martinsuchenak/rackd/internal/storage"
)

func Command() *cli.Command {
    return &cli.Command{
        Name:  "server",
        Usage: "Start the HTTP/MCP server",
        Flags: []*cli.Flag{
            {Name: "data-dir", Usage: "Data directory", Default: "./data"},
            {Name: "listen-addr", Usage: "Listen address", Default: ":8080"},
            {Name: "api-auth-token", Usage: "API authentication token"},
            {Name: "mcp-auth-token", Usage: "MCP authentication token"},
            {Name: "log-level", Usage: "Log level (trace/debug/info/warn/error)", Default: "info"},
            {Name: "log-format", Usage: "Log format (text/json)", Default: "text"},
            {Name: "discovery-enabled", Usage: "Enable network discovery", Default: "true"},
            {Name: "discovery-interval", Usage: "Discovery scan interval", Default: "24h"},
        },
        Run: func(ctx context.Context, cmd *cli.Command) error {
            cfg := config.Load()

            // Initialize logging
            log.Init(cfg.LogLevel, cfg.LogFormat)

            // Initialize storage
            store, err := storage.NewExtendedStorage(cfg.DataDir, "sqlite", "")
            if err != nil {
                return err
            }

            // Run server
            return server.Run(cfg, store)
        },
    }
}
```

## Device Commands

**File**: `cmd/device/device.go`

```go
package device

import "github.com/paularlott/cli"

func Command() *cli.Command {
    return &cli.Command{
        Name:  "device",
        Usage: "Device management commands",
        SubCommands: []*cli.Command{
            ListCommand(),
            GetCommand(),
            AddCommand(),
            UpdateCommand(),
            DeleteCommand(),
        },
    }
}
```

**File**: `cmd/device/list.go`

```go
package device

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"

    "github.com/paularlott/cli"
    "github.com/martinsuchenak/rackd/cmd/client"
)

func ListCommand() *cli.Command {
    return &cli.Command{
        Name:  "list",
        Usage: "List all devices",
        Flags: []*cli.Flag{
            {Name: "query", Short: "q", Usage: "Search query"},
            {Name: "tags", Short: "t", Usage: "Filter by tags (comma-separated)"},
            {Name: "datacenter", Short: "d", Usage: "Filter by datacenter ID"},
            {Name: "output", Short: "o", Usage: "Output format (table/json)", Default: "table"},
        },
        Run: func(ctx context.Context, cmd *cli.Command) error {
            cfg := client.LoadConfig()

            url := fmt.Sprintf("%s/api/devices", cfg.ServerURL)
            if query := cmd.GetString("query"); query != "" {
                url = fmt.Sprintf("%s/api/devices/search?q=%s", cfg.ServerURL, query)
            }

            resp, err := client.DoRequest("GET", url, cfg.Token, nil)
            if err != nil {
                return err
            }
            defer resp.Body.Close()

            if resp.StatusCode != http.StatusOK {
                return client.HandleError(resp)
            }

            var devices []map[string]interface{}
            if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
                return err
            }

            if cmd.GetString("output") == "json" {
                return client.PrintJSON(devices)
            }

            return client.PrintDeviceTable(devices)
        },
    }
}
```

## Network Commands

**File**: `cmd/network/network.go`

```go
package network

import "github.com/paularlott/cli"

func Command() *cli.Command {
    return &cli.Command{
        Name:  "network",
        Usage: "Network management commands",
        SubCommands: []*cli.Command{
            ListCommand(),
            GetCommand(),
            AddCommand(),
            DeleteCommand(),
            PoolCommand(), // Subcommand for pool operations
        },
    }
}
```

## Discovery Commands

**File**: `cmd/discovery/discovery.go`

```go
package discovery

import "github.com/paularlott/cli"

func Command() *cli.Command {
    return &cli.Command{
        Name:  "discovery",
        Usage: "Network discovery commands",
        SubCommands: []*cli.Command{
            ScanCommand(),
            ListCommand(),
            PromoteCommand(),
        },
    }
}

// cmd/discovery/scan.go
func ScanCommand() *cli.Command {
    return &cli.Command{
        Name:  "scan",
        Usage: "Start a network discovery scan",
        Flags: []*cli.Flag{
            {Name: "network", Short: "n", Usage: "Network ID to scan", Required: true},
            {Name: "type", Short: "t", Usage: "Scan type (quick/full/deep)", Default: "full"},
        },
        Run: func(ctx context.Context, cmd *cli.Command) error {
            // Implementation
            return nil
        },
    }
}
```
