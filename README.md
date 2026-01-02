# Device Manager

A Go-based device tracking application with MCP server support, web UI, and CLI.

## Features

- Track devices with detailed information (name, IP addresses, make/model, OS, location, tags, domains)
- RESTful API for CRUD operations
- Web UI for easy management
- CLI tool for command-line operations
- MCP (Model Context Protocol) server for AI integration
- File-based storage (JSON or TOML)
- Deploy to Nomad, Docker, or run locally

## Quick Start

### Local Development

```bash
# Run server directly
go run cmd/server/main.go

# Or build and run
go build -o devicemanager cmd/server/main.go
./devicemanager
```

The server will start on `http://localhost:8080`:
- Web UI: http://localhost:8080
- API: http://localhost:8080/api/
- MCP: http://localhost:8080/mcp

### Docker

```bash
# Build and run with docker-compose
docker-compose up -d

# Or build and run manually
docker build -t devicemanager .
docker run -p 8080:8080 -v $(pwd)/data:/app/data devicemanager
```

### Nomad Deployment

```bash
nomad job run deployment/nomad/devicemanager.nomad
```

## Configuration

Configure via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `DM_DATA_DIR` | `./data` | Directory for device data files |
| `DM_LISTEN_ADDR` | `:8080` | Server listen address |
| `DM_STORAGE_FORMAT` | `json` | Storage format: `json` or `toml` |
| `DM_BEARER_TOKEN` | (none) | MCP authentication token |

## CLI Usage

```bash
# Build CLI
go build -o dm-cli cmd/cli/main.go

# Add a device
./dm-cli add \
  --name "web-server-01" \
  --make-model "Dell PowerEdge R740" \
  --os "Ubuntu 22.04" \
  --location "Rack A1" \
  --tags "server,production,web" \
  --domains "example.com,www.example.com"

# List all devices
./dm-cli list

# Filter by tags
./dm-cli list --filter server,production

# Get device details
./dm-cli get web-server-01

# Search devices
./dm-cli search "dell"

# Update a device
./dm-cli update web-server-01 \
  --location "Rack B2" \
  --tags "server,production,web,backend"

# Delete a device
./dm-cli delete web-server-01

# Use local storage instead of server
./dm-cli --local add --name "local-device"
```

## REST API

### List Devices
```bash
GET /api/devices
GET /api/devices?tag=server&tag=production
```

### Get Device
```bash
GET /api/devices/{id}
```

### Create Device
```bash
POST /api/devices
Content-Type: application/json

{
  "name": "web-server-01",
  "description": "Main web server",
  "make_model": "Dell PowerEdge R740",
  "os": "Ubuntu 22.04",
  "location": "Rack A1",
  "tags": ["server", "production", "web"],
  "domains": ["example.com"],
  "addresses": [
    {
      "ip": "192.168.1.10",
      "port": 443,
      "type": "ipv4",
      "label": "management"
    }
  ]
}
```

### Update Device
```bash
PUT /api/devices/{id}
Content-Type: application/json

{
  "name": "web-server-01",
  "location": "Rack B2"
}
```

### Delete Device
```bash
DELETE /api/devices/{id}
```

### Search Devices
```bash
GET /api/search?q=dell
```

## MCP Server

The MCP server provides AI assistants with tools to manage devices:

### Tools

- `device_add` - Add a new device
- `device_update` - Update an existing device
- `device_get` - Get device by ID or name
- `device_list` - List devices with optional tag filtering
- `device_search` - Search devices
- `device_delete` - Delete a device

### MCP Client Configuration

Configure your MCP client (e.g., Claude Desktop) to connect:

```json
{
  "mcpServers": {
    "devicemanager": {
      "url": "http://localhost:8080/mcp",
      "headers": {
        "Authorization": "Bearer your-token-here"
      }
    }
  }
}
```

## Data Model

```go
type Device struct {
    ID          string       `json:"id"`
    Name        string       `json:"name"`
    Description string       `json:"description"`
    MakeModel   string       `json:"make_model"`
    OS          string       `json:"os"`
    Location    string       `json:"location"`
    Tags        []string     `json:"tags"`
    Addresses   []Address    `json:"addresses"`
    Domains     []string     `json:"domains"`
    CreatedAt   time.Time    `json:"created_at"`
    UpdatedAt   time.Time    `json:"updated_at"`
}

type Address struct {
    IP    string `json:"ip"`
    Port  int    `json:"port"`
    Type  string `json:"type"`   // "ipv4" or "ipv6"
    Label string `json:"label"`  // e.g., "management", "data"
}
```

## Development

### Project Structure

```
devicemanager/
├── cmd/
│   ├── server/          # HTTP server
│   └── cli/             # CLI tool
├── internal/
│   ├── config/          # Configuration
│   ├── storage/         # File storage
│   ├── model/           # Data models
│   ├── api/             # REST handlers
│   ├── mcp/             # MCP server
│   └── ui/              # Web UI
├── deployment/
│   └── nomad/           # Nomad jobs
├── data/                # Device data files
└── go.mod
```

### Dependencies

- `github.com/pelletier/go-toml/v2` - TOML support
- `github.com/paularlott/cli` - CLI framework
- `github.com/paularlott/mcp` - MCP server

### Building

```bash
# Build server
go build -o devicemanager ./cmd/server

# Build CLI
go build -o dm-cli ./cmd/cli

# Build for Linux (Docker)
GOOS=linux GOARCH=amd64 go build -o devicemanager ./cmd/server
```

## License

MIT License
