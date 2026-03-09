package mcp

import (
	"github.com/paularlott/mcp"
)

func (s *Server) registerDNSTools() {
	// Provider tools
	s.mcpServer.RegisterTool(
		mcp.NewTool("dns_provider_list", "List DNS providers",
			mcp.String("type", "Filter by provider type (technitium, powerdns, bind)"),
			mcp.Number("limit", "Max results to return (default 100, max 1000)"),
			mcp.Number("offset", "Number of results to skip for pagination"),
		).Discoverable("dns", "provider", "nameserver"),
		s.handleDNSProviderList,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("dns_provider_get", "Get a DNS provider by ID",
			mcp.String("id", "Provider ID", mcp.Required()),
		).Discoverable("dns", "provider"),
		s.handleDNSProviderGet,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("dns_provider_save", "Create or update a DNS provider",
			mcp.String("id", "Provider ID (omit for new)"),
			mcp.String("name", "Provider name", mcp.Required()),
			mcp.String("type", "Provider type (technitium, powerdns, bind)", mcp.Required()),
			mcp.String("endpoint", "Provider API endpoint URL"),
			mcp.String("token", "Provider API token/key"),
			mcp.String("description", "Description"),
		).Discoverable("dns", "provider", "create", "update", "nameserver"),
		s.handleDNSProviderSave,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("dns_provider_delete", "Delete a DNS provider",
			mcp.String("id", "Provider ID", mcp.Required()),
		).Discoverable("dns", "provider", "delete", "remove"),
		s.handleDNSProviderDelete,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("dns_provider_test", "Test a DNS provider connection",
			mcp.String("id", "Provider ID", mcp.Required()),
		).Discoverable("dns", "provider", "test", "check", "connection"),
		s.handleDNSProviderTest,
	)

	// Zone tools
	s.mcpServer.RegisterTool(
		mcp.NewTool("dns_zone_list", "List DNS zones",
			mcp.String("provider_id", "Filter by provider ID"),
			mcp.String("network_id", "Filter by network ID"),
			mcp.Number("limit", "Max results to return (default 100, max 1000)"),
			mcp.Number("offset", "Number of results to skip for pagination"),
		).Discoverable("dns", "zone", "domain"),
		s.handleDNSZoneList,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("dns_zone_get", "Get a DNS zone by ID",
			mcp.String("id", "Zone ID", mcp.Required()),
		).Discoverable("dns", "zone", "domain"),
		s.handleDNSZoneGet,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("dns_zone_save", "Create or update a DNS zone",
			mcp.String("id", "Zone ID (omit for new)"),
			mcp.String("name", "Zone name (e.g., example.com)", mcp.Required()),
			mcp.String("provider_id", "Provider ID (required for new zones)"),
			mcp.String("network_id", "Associated network ID"),
			mcp.Boolean("auto_sync", "Enable automatic sync to provider"),
			mcp.Boolean("create_ptr", "Auto-create PTR records"),
			mcp.String("ptr_zone", "PTR zone name override"),
			mcp.Number("ttl", "Default TTL in seconds"),
			mcp.String("description", "Description"),
		).Discoverable("dns", "zone", "domain", "create", "update"),
		s.handleDNSZoneSave,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("dns_zone_delete", "Delete a DNS zone",
			mcp.String("id", "Zone ID", mcp.Required()),
		).Discoverable("dns", "zone", "domain", "delete", "remove"),
		s.handleDNSZoneDelete,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("dns_zone_sync", "Sync a DNS zone to its provider",
			mcp.String("id", "Zone ID", mcp.Required()),
		).Discoverable("dns", "zone", "sync", "push", "deploy"),
		s.handleDNSZoneSync,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("dns_zone_import", "Import DNS records from provider into a zone",
			mcp.String("id", "Zone ID", mcp.Required()),
		).Discoverable("dns", "zone", "import", "pull", "fetch"),
		s.handleDNSZoneImport,
	)

	// Record tools
	s.mcpServer.RegisterTool(
		mcp.NewTool("dns_record_list", "List DNS records for a zone",
			mcp.String("zone_id", "Zone ID", mcp.Required()),
			mcp.String("type", "Filter by record type (A, AAAA, CNAME, MX, TXT, PTR, NS, SRV)"),
			mcp.String("device_id", "Filter by device ID"),
			mcp.String("sync_status", "Filter by sync status (synced, pending, failed)"),
			mcp.Number("limit", "Max results to return (default 100, max 1000)"),
			mcp.Number("offset", "Number of results to skip for pagination"),
		).Discoverable("dns", "record", "A", "AAAA", "CNAME", "MX", "TXT", "PTR"),
		s.handleDNSRecordList,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("dns_record_get", "Get a DNS record by ID",
			mcp.String("id", "Record ID", mcp.Required()),
		).Discoverable("dns", "record"),
		s.handleDNSRecordGet,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("dns_record_save", "Create or update a DNS record",
			mcp.String("id", "Record ID (omit for new)"),
			mcp.String("zone_id", "Zone ID (required for new records)"),
			mcp.String("device_id", "Associated device ID"),
			mcp.String("name", "Record name (e.g., www)", mcp.Required()),
			mcp.String("type", "Record type (A, AAAA, CNAME, MX, TXT, PTR, NS, SRV)", mcp.Required()),
			mcp.String("value", "Record value", mcp.Required()),
			mcp.Number("ttl", "TTL in seconds"),
		).Discoverable("dns", "record", "create", "update", "A", "AAAA", "CNAME"),
		s.handleDNSRecordSave,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("dns_record_delete", "Delete a DNS record",
			mcp.String("id", "Record ID", mcp.Required()),
		).Discoverable("dns", "record", "delete", "remove"),
		s.handleDNSRecordDelete,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("dns_record_link", "Link a DNS record to a device",
			mcp.String("id", "Record ID", mcp.Required()),
			mcp.String("device_id", "Device ID to link", mcp.Required()),
			mcp.String("address_id", "Specific address ID on the device"),
			mcp.Boolean("add_to_domains", "Add record name to device domains list"),
		).Discoverable("dns", "record", "link", "device", "associate"),
		s.handleDNSRecordLink,
	)
}
