# User Guide and Use Cases

This document provides a high-level overview of how users interact with Rackd, covering common use cases for both the Web UI and CLI. It serves as a foundation for more detailed user documentation.

## 1. Core Functionality Overview

Rackd is an IP Address Management (IPAM) and Device Inventory System. Its primary functions include:
- **Device Tracking**: Registering and managing information about network devices (servers, switches, routers).
- **Location Management**: Organizing devices within datacenters and physical locations.
- **Network Management**: Defining subnets, VLANs, and IP address pools.
- **IP Allocation**: Managing the assignment and utilization of IP addresses.
- **Network Discovery**: Automatically finding devices on the network.
- **Relationship Mapping**: Tracking how devices are connected or dependent on each other.

## 2. Web UI Usage

The Web UI (built with Alpine.js) provides a graphical interface for managing all aspects of Rackd.

### 2.1. Navigating the UI

The main navigation will typically include sections for:
- **Devices**: View, add, edit, and delete devices.
- **Networks**: Manage network configurations, subnets, and VLANs.
- **Datacenters**: Organize physical locations and view devices within them.
- **IP Pools**: Manage IP address ranges within networks.
- **Discovery**: Initiate scans, view discovered devices, and promote them to inventory.
- **Search**: Global search functionality to quickly find any entity.

### 2.2. Common Web UI Workflows

#### Use Case: Registering a New Server

1.  Navigate to the "Devices" section.
2.  Click "Add New Device".
3.  Fill in details: Name, Make/Model, OS, Datacenter, Tags.
4.  Add IP Addresses, specifying Network and optional Switch Port.
5.  Save the device.

#### Use Case: Creating a New Network Subnet

1.  Navigate to the "Networks" section.
2.  Click "Add New Network".
3.  Provide Name, CIDR Subnet (e.g., `192.168.1.0/24`), VLAN ID, and associate with a Datacenter.
4.  Optionally, define IP Pools within this network.

#### Use Case: Initiating a Network Discovery Scan

1.  Navigate to the "Discovery" section.
2.  Select an existing network.
3.  Choose a scan type (Quick, Full, Deep).
4.  Start the scan.
5.  Monitor scan progress and review discovered devices.
6.  Promote discovered devices to the main inventory as needed.

## 3. CLI Usage

The Rackd CLI provides command-line access to manage the system, ideal for scripting and automation.

### 3.1. General Structure

All commands follow the pattern: `rackd [command] [subcommand] [flags]`

Example: `rackd device list --datacenter my-dc-id`

### 3.2. Common CLI Workflows

#### Use Case: Listing All Devices

`rackd device list`
`rackd device list -o json` (for JSON output)

#### Use Case: Adding a New Device via CLI

`rackd device add --name "webserver-01" --os "Ubuntu 22.04" --datacenter "dc-atl"`
(Further commands would be needed to add IP addresses and other details, or these could be integrated into the `add` command's flags or a configuration file import.)

#### Use Case: Starting a Discovery Scan

`rackd discovery scan --network "mynet-id-123" --type full`

#### Use Case: Promoting a Discovered Device

`rackd discovery promote --discovered-id "disc-abc-123" --name "new-found-server"`

### 3.3. Configuration via CLI

The `rackd server` command and other subcommands accept flags that override environment variables for specific runs.

Example: `rackd server --listen-addr :9000 --api-auth-token my-cli-token`

## 4. MCP (Model Context Protocol) Usage

The MCP server is designed for integration with AI/automation tools. It exposes a single HTTP endpoint (`/mcp`) that allows these tools to invoke Rackd's internal functions (tools).

### 4.1. Common MCP Use Cases

- **Automated IP Allocation**: An AI tool requests the next available IP from a pool for a new VM deployment.
- **Inventory Updates**: An automation script updates device status or properties based on external system events.
- **Discovery Triggering**: An external monitoring system triggers a deep network scan in response to anomalous network behavior.

## 5. Future Enhancements

- **Mobile Application**: The `webui/src/core` separation allows for future development of native mobile applications leveraging the same API client.
- **Advanced Integrations**: The MCP and Feature Injection points are designed for future integrations with more complex external systems.
