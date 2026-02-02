# Quick Start Guide

Get Rackd running in 5 minutes and perform essential operations.

## 1. Install and Start (1 minute)

```bash
# Download
curl -LO https://github.com/martinsuchenak/rackd/releases/latest/download/rackd-linux-amd64
chmod +x rackd-linux-amd64

# Start server
./rackd-linux-amd64 server
```

Access web UI at http://localhost:8080

## 2. Create First Datacenter (30 seconds)

```bash
# CLI
./rackd-linux-amd64 datacenter create --name "Main DC" --location "New York"

# Or via Web UI: Datacenters → Add Datacenter
```

## 3. Create First Network (30 seconds)

```bash
# Create network with CIDR
./rackd-linux-amd64 network create --name "Office LAN" --cidr "192.168.1.0/24" --datacenter-id 1

# Or via Web UI: Networks → Add Network
```

## 4. Add First Device (1 minute)

```bash
# Add a server
./rackd-linux-amd64 device create \
  --name "web-server-01" \
  --type "server" \
  --ip "192.168.1.10" \
  --datacenter-id 1

# Or via Web UI: Devices → Add Device
```

## 5. Run Discovery Scan (2 minutes)

```bash
# Scan network for devices
./rackd-linux-amd64 discovery scan --network "192.168.1.0/24"

# View discovered devices
./rackd-linux-amd64 device list
```

## 6. Basic Operations (1 minute)

```bash
# List all resources
./rackd-linux-amd64 datacenter list
./rackd-linux-amd64 network list
./rackd-linux-amd64 device list

# Get device details
./rackd-linux-amd64 device get --id 1

# Update device
./rackd-linux-amd64 device update --id 1 --description "Production web server"
```

## Next Steps

- Explore the Web UI for visual management
- Set up device relationships and dependencies
- Configure automated discovery schedules
- Review the [CLI Reference](cli.md) for advanced commands
- Check [API Reference](api.md) for integration options

You now have a working Rackd installation with your first datacenter, network, and device!