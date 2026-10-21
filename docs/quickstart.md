# Quick Start Guide

Get Rackd running in 5 minutes and perform essential operations.

## 1. Install and Start (1 minute)

```bash
# Download and unpack a release (see Installation for other platforms and verification)
VERSION=x.y.z   # the release to install
curl -LO "https://github.com/martinsuchenak/rackd/releases/download/v${VERSION}/rackd_${VERSION}_linux_amd64.tar.gz"
tar -xzf "rackd_${VERSION}_linux_amd64.tar.gz" rackd

# Start the server. The first admin is created from these on first start only.
INITIAL_ADMIN_USERNAME=admin INITIAL_ADMIN_PASSWORD='<choose a password>' ./rackd server
```

Open the web UI at http://localhost:8080 and sign in as `admin`.

Session cookies are `Secure` by default, so browsers only keep them over HTTPS or on `localhost`. To try Rackd over plain HTTP from another machine, start it with `COOKIE_SECURE=false` as well (or `--dev-mode`, which also turns off rate limiting), and use HTTPS for real use.

## 2. Point the CLI at the Server (30 seconds)

The CLI authenticates with an API key. In the web UI, open **API Keys**, create a key and copy it (it is shown once), then in another terminal:

```bash
export RACKD_SERVER_URL=http://localhost:8080
export RACKD_TOKEN='<api key>'
```

## 3. Create First Datacenter (30 seconds)

```bash
# CLI
./rackd datacenter add --name "Main DC" --location "New York"

# Or via Web UI: Datacenters → Add Datacenter
```

## 4. Create First Network (30 seconds)

```bash
# Create network with CIDR
./rackd network add --name "Office LAN" --subnet "192.168.1.0/24"

# Or via Web UI: Networks → Add Network
```

## 5. Add First Device (1 minute)

```bash
# Add a server
./rackd device add \
  --name "web-server-01" \
  --description "Production web server"

# Or via Web UI: Devices → Add Device
```

## 6. Run Discovery Scan (2 minutes)

```bash
# Scan a network for devices (use the network ID printed in step 4, or from `network list`)
./rackd discovery scan --network <network-id>

# View discovered devices
./rackd discovery list --network <network-id>
```

## 7. Basic Operations (1 minute)

```bash
# List all resources
./rackd datacenter list
./rackd network list
./rackd device list

# Get device details
./rackd device get --id <device-id>

# Update device
./rackd device update --id <device-id> --description "Updated description"
```

## Next Steps

- Explore the Web UI for visual management
- Set up device relationships and dependencies
- Configure automated discovery schedules
- Set `ENCRYPTION_KEY` to enable credentials, scan profiles, scheduled scans and DNS (see [Configuration Reference](configuration-reference.md))
- Review the [CLI Reference](cli.md) for advanced commands
- Check [API Reference](api.md) for integration options

You now have a working Rackd installation with your first datacenter, network, and device!
