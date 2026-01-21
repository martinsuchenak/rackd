# Configuration Reference

This document covers all configuration options for Rackd.

## Configuration Structure

```go
type Config struct {
    // Core
    DataDir      string `env:"DATA_DIR"       default:"./data"`
    ListenAddr   string `env:"LISTEN_ADDR"    default:":8080"`
    APIAuthToken string `env:"API_AUTH_TOKEN" default:""`
    MCPAuthToken string `env:"MCP_AUTH_TOKEN" default:""`
    LogFormat    string `env:"LOG_FORMAT"     default:"text"`
    LogLevel     string `env:"LOG_LEVEL"      default:"info"`

    // Discovery
    DiscoveryEnabled       bool          `env:"DISCOVERY_ENABLED"        default:"true"`
    DiscoveryInterval      time.Duration `env:"DISCOVERY_INTERVAL"       default:"24h"`
    DiscoveryMaxConcurrent int           `env:"DISCOVERY_MAX_CONCURRENT" default:"10"`
    DiscoveryTimeout       time.Duration `env:"DISCOVERY_TIMEOUT"        default:"5s"`
    DiscoveryCleanupDays   int           `env:"DISCOVERY_CLEANUP_DAYS"   default:"30"`
    DiscoveryScanOnStartup bool          `env:"DISCOVERY_SCAN_ON_STARTUP" default:"false"`
}
```

## Environment Variables

### Core Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `DATA_DIR` | `./data` | Directory for SQLite database and data files |
| `LISTEN_ADDR` | `:8080` | HTTP server listen address |
| `API_AUTH_TOKEN` | `` | Bearer token for API authentication (empty = no auth) |
| `MCP_AUTH_TOKEN` | `` | Bearer token for MCP authentication (empty = no auth) |
| `LOG_FORMAT` | `text` | Log output format: `text` or `json` |
| `LOG_LEVEL` | `info` | Log level: `trace`, `debug`, `info`, `warn`, `error` |

### Discovery Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `DISCOVERY_ENABLED` | `true` | Enable network discovery features |
| `DISCOVERY_INTERVAL` | `24h` | Interval between scheduled discovery scans |
| `DISCOVERY_MAX_CONCURRENT` | `10` | Maximum concurrent host scans |
| `DISCOVERY_TIMEOUT` | `5s` | Timeout for individual host probes |
| `DISCOVERY_CLEANUP_DAYS` | `30` | Delete discovered devices not seen for N days |
| `DISCOVERY_SCAN_ON_STARTUP` | `false` | Run scheduled scans immediately on startup |

### Enterprise Settings (Enterprise Only)

Enterprise features have their own configuration in the rackd-enterprise repository. OSS does not include these settings.

## Example .env File

```bash
# Core settings
DATA_DIR=./data
LISTEN_ADDR=:8080
API_AUTH_TOKEN=your-secret-token
MCP_AUTH_TOKEN=your-mcp-token
LOG_FORMAT=text
LOG_LEVEL=info

# Discovery
DISCOVERY_ENABLED=true
DISCOVERY_INTERVAL=24h
DISCOVERY_MAX_CONCURRENT=10
DISCOVERY_TIMEOUT=5s
DISCOVERY_CLEANUP_DAYS=30
DISCOVERY_SCAN_ON_STARTUP=false
```

## CLI Flags

The server command supports the following flags which override environment variables:

```
rackd server [flags]

Flags:
  --data-dir string          Data directory (default "./data")
  --listen-addr string       Listen address (default ":8080")
  --api-auth-token string    API authentication token
  --mcp-auth-token string    MCP authentication token
  --log-level string         Log level (trace/debug/info/warn/error) (default "info")
  --log-format string        Log format (text/json) (default "text")
  --discovery-enabled        Enable network discovery (default true)
  --discovery-interval       Discovery scan interval (default "24h")
```

## Precedence

Configuration is loaded in this order (later overrides earlier):
1. Default values
2. Environment variables
3. `.env` file
4. CLI flags
