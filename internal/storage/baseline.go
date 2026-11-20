package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/martinsuchenak/xdal/pkg/migrate"
	"github.com/martinsuchenak/xdal/pkg/xdal"
)

// SchemaMigrationNamespace is the history namespace of the portable migration
// track. Every schema change from the baseline on is written here, with xdal's
// schema builder, so it applies to SQLite, PostgreSQL and MySQL alike.
//
// The legacy "core" track (migrations.go) is frozen: it is the SQLite-only
// history that built the schema up to the baseline, and still runs first on
// SQLite so existing databases upgrade exactly as before.
const SchemaMigrationNamespace = "schema"

// Column sizes for the baseline. MySQL can only index bounded strings, so key,
// foreign key and indexed columns are VARCHAR rather than TEXT.
const (
	idSize  = 64  // primary and foreign key ids
	keySize = 255 // other indexed or unique text: names, tokens, IPs
)

// baselineVersion is the first migration of the portable track.
const baselineVersion = "20261121000000"

// SchemaMigrations returns the portable migration track, in order. On a
// PostgreSQL or MySQL database it builds the whole core schema, so tests of
// stores outside this package can apply it with dialecttest.Run; on SQLite
// it builds nothing (the legacy track does that).
func SchemaMigrations() []migrate.Migration { return portableMigrations() }

// portableMigrations is the portable migration track, in order.
func portableMigrations() []migrate.Migration {
	return []migrate.Migration{
		{Version: baselineVersion, Name: "baseline", Up: baselineUp},
	}
}

// baselineUp creates the schema the legacy track had built by its last
// migration, on a server database. On SQLite the legacy track has already
// built it, so the baseline is recorded without changing anything.
func baselineUp(ctx context.Context, tx *migrate.Tx) error {
	if tx.Dialect().Name() == xdal.DialectSQLite {
		return nil
	}
	return createBaseline(ctx, tx)
}

// createBaseline creates every core table, index and seed row. It is the
// portable equivalent of the legacy track; TestBaselineMatchesLegacySchema
// keeps the two in step. Full-text search tables are not part of it: search
// on each backend is set up separately.
func createBaseline(ctx context.Context, tx *migrate.Tx) error {
	if err := tx.Apply(ctx, baselineStatements(tx.Schema())...); err != nil {
		return err
	}
	if err := createPartialIndexes(ctx, tx); err != nil {
		return err
	}
	return seedBaseline(ctx, tx)
}

// baselineStatements returns the tables in foreign key order, each followed
// by its indexes.
//
// Differences from the legacy SQLite schema, all deliberate:
//   - flag columns are BOOLEAN rather than INTEGER;
//   - custom_field_values.string_value has no index (idx_cfv_string): values
//     are unbounded text, and lookups already go through idx_cfv_field.
func baselineStatements(s *xdal.Schema) []xdal.Statement {
	var stmts []xdal.Statement

	datacenters := s.CreateTable("datacenters")
	datacenters.String("id", idSize).PrimaryKey()
	datacenters.Text("name").NotNull()
	datacenters.Text("location")
	datacenters.Text("description")
	datacenters.Timestamp("created_at").DefaultNow()
	datacenters.Timestamp("updated_at").DefaultNow()
	stmts = append(stmts, datacenters)

	networks := s.CreateTable("networks")
	networks.String("id", idSize).PrimaryKey()
	networks.Text("name").NotNull()
	networks.Text("subnet").NotNull()
	networks.Integer("vlan_id")
	networks.String("datacenter_id", idSize).References("datacenters", "id")
	networks.Text("description")
	networks.Timestamp("created_at").DefaultNow()
	networks.Timestamp("updated_at").DefaultNow()
	stmts = append(stmts, networks)
	stmts = append(stmts, s.CreateIndex("idx_networks_datacenter", "networks", "datacenter_id"))

	networkpools := s.CreateTable("network_pools")
	networkpools.String("id", idSize).PrimaryKey()
	networkpools.String("network_id", idSize).NotNull().References("networks", "id").OnDelete(xdal.Cascade)
	networkpools.Text("name").NotNull()
	networkpools.Text("start_ip").NotNull()
	networkpools.Text("end_ip").NotNull()
	networkpools.Text("description")
	networkpools.Timestamp("created_at").DefaultNow()
	networkpools.Timestamp("updated_at").DefaultNow()
	stmts = append(stmts, networkpools)
	stmts = append(stmts, s.CreateIndex("idx_network_pools_network", "network_pools", "network_id"))

	devices := s.CreateTable("devices")
	devices.String("id", idSize).PrimaryKey()
	devices.String("name", keySize).NotNull()
	devices.Text("description")
	devices.Text("make_model")
	devices.Text("os")
	devices.String("datacenter_id", idSize).References("datacenters", "id")
	devices.Text("username")
	devices.Text("location")
	devices.Timestamp("created_at").DefaultNow()
	devices.Timestamp("updated_at").DefaultNow()
	devices.Text("hostname").Default("")
	devices.String("status", keySize).NotNull().Default("active")
	devices.Timestamp("decommission_date")
	devices.Timestamp("status_changed_at")
	devices.Text("status_changed_by")
	stmts = append(stmts, devices)
	stmts = append(stmts, s.CreateIndex("idx_devices_datacenter", "devices", "datacenter_id"))
	stmts = append(stmts, s.CreateIndex("idx_devices_name", "devices", "name"))
	stmts = append(stmts, s.CreateIndex("idx_devices_status", "devices", "status"))

	addresses := s.CreateTable("addresses")
	addresses.String("id", idSize).PrimaryKey()
	addresses.String("device_id", idSize).NotNull().References("devices", "id").OnDelete(xdal.Cascade)
	addresses.String("ip", keySize).NotNull()
	addresses.Integer("port")
	addresses.Text("type").Default("ipv4")
	addresses.Text("label")
	addresses.String("network_id", idSize).References("networks", "id")
	addresses.Text("switch_port")
	addresses.String("pool_id", idSize).References("network_pools", "id")
	stmts = append(stmts, addresses)
	stmts = append(stmts, s.CreateIndex("idx_addresses_device", "addresses", "device_id"))
	stmts = append(stmts, s.CreateIndex("idx_addresses_ip", "addresses", "ip"))
	stmts = append(stmts, s.CreateIndex("idx_addresses_network_ip", "addresses", "network_id", "ip"))
	stmts = append(stmts, s.CreateIndex("idx_addresses_pool", "addresses", "pool_id"))

	users := s.CreateTable("users")
	users.String("id", idSize).PrimaryKey()
	users.String("username", keySize).NotNull().Unique()
	users.String("email", keySize).Unique()
	users.Text("full_name")
	users.Text("password_hash").NotNull()
	users.Boolean("is_active").NotNull().Default(true)
	users.Boolean("is_admin").NotNull().Default(false)
	users.Timestamp("created_at").NotNull()
	users.Timestamp("updated_at").NotNull()
	users.Timestamp("last_login_at")
	stmts = append(stmts, users)
	stmts = append(stmts, s.CreateIndex("idx_users_email", "users", "email"))
	stmts = append(stmts, s.CreateIndex("idx_users_is_active", "users", "is_active"))
	stmts = append(stmts, s.CreateIndex("idx_users_username", "users", "username"))

	apikeys := s.CreateTable("api_keys")
	apikeys.String("id", idSize).PrimaryKey()
	apikeys.String("name", keySize).NotNull().Unique()
	apikeys.String("key", keySize).NotNull().Unique()
	apikeys.Text("description")
	apikeys.Timestamp("created_at").NotNull()
	apikeys.Timestamp("last_used_at")
	apikeys.Timestamp("expires_at")
	apikeys.String("user_id", idSize).References("users", "id")
	apikeys.Text("scopes")
	stmts = append(stmts, apikeys)
	stmts = append(stmts, s.CreateIndex("idx_api_keys_key", "api_keys", "key"))
	stmts = append(stmts, s.CreateIndex("idx_api_keys_name", "api_keys", "name"))
	stmts = append(stmts, s.CreateIndex("idx_api_keys_user_id", "api_keys", "user_id"))

	auditlogs := s.CreateTable("audit_logs")
	auditlogs.String("id", idSize).PrimaryKey()
	auditlogs.Timestamp("timestamp").NotNull()
	auditlogs.String("action", keySize).NotNull()
	auditlogs.String("resource", keySize).NotNull()
	auditlogs.String("resource_id", idSize)
	auditlogs.String("user_id", idSize)
	auditlogs.Text("username")
	auditlogs.Text("ip_address")
	auditlogs.Text("changes")
	auditlogs.Text("status").NotNull()
	auditlogs.Text("error")
	auditlogs.String("source", keySize)
	stmts = append(stmts, auditlogs)
	stmts = append(stmts, s.CreateIndex("idx_audit_logs_action", "audit_logs", "action"))
	stmts = append(stmts, s.CreateIndex("idx_audit_logs_resource", "audit_logs", "resource", "resource_id"))
	stmts = append(stmts, s.CreateIndex("idx_audit_logs_source", "audit_logs", "source"))
	stmts = append(stmts, s.CreateIndex("idx_audit_logs_timestamp", "audit_logs", "timestamp"))
	stmts = append(stmts, s.CreateIndex("idx_audit_logs_user", "audit_logs", "user_id"))

	circuits := s.CreateTable("circuits")
	circuits.String("id", idSize).PrimaryKey()
	circuits.Text("name").NotNull()
	circuits.String("circuit_id", idSize).NotNull().Unique()
	circuits.String("provider", keySize).NotNull()
	circuits.Text("type").NotNull().Default("fiber")
	circuits.String("status", keySize).NotNull().Default("active")
	circuits.Integer("capacity_mbps").NotNull().Default(0)
	circuits.String("datacenter_a_id", idSize).References("datacenters", "id").OnDelete(xdal.SetNull)
	circuits.String("datacenter_b_id", idSize).References("datacenters", "id").OnDelete(xdal.SetNull)
	circuits.String("device_a_id", idSize).References("devices", "id").OnDelete(xdal.SetNull)
	circuits.String("device_b_id", idSize).References("devices", "id").OnDelete(xdal.SetNull)
	circuits.Text("port_a").Default("")
	circuits.Text("port_b").Default("")
	circuits.Text("ip_address_a").Default("")
	circuits.Text("ip_address_b").Default("")
	circuits.Integer("vlan_id").Default(0)
	circuits.Text("description").Default("")
	circuits.Timestamp("install_date")
	circuits.Timestamp("terminate_date")
	circuits.Float("monthly_cost").Default(0)
	circuits.Text("contract_number").Default("")
	circuits.Text("contact_name").Default("")
	circuits.Text("contact_phone").Default("")
	circuits.Text("contact_email").Default("")
	circuits.Text("tags").Default("[]")
	circuits.Timestamp("created_at").NotNull()
	circuits.Timestamp("updated_at").NotNull()
	stmts = append(stmts, circuits)
	stmts = append(stmts, s.CreateIndex("idx_circuits_datacenter_a", "circuits", "datacenter_a_id"))
	stmts = append(stmts, s.CreateIndex("idx_circuits_datacenter_b", "circuits", "datacenter_b_id"))
	stmts = append(stmts, s.CreateIndex("idx_circuits_device_a", "circuits", "device_a_id"))
	stmts = append(stmts, s.CreateIndex("idx_circuits_device_b", "circuits", "device_b_id"))
	stmts = append(stmts, s.CreateIndex("idx_circuits_provider", "circuits", "provider"))
	stmts = append(stmts, s.CreateIndex("idx_circuits_status", "circuits", "status"))

	// idx_conflicts_active_duplicate_ip: partial index, created per dialect below
	conflicts := s.CreateTable("conflicts")
	conflicts.String("id", idSize).PrimaryKey()
	conflicts.String("type", keySize).NotNull()
	conflicts.String("status", keySize).NotNull().Default("active")
	conflicts.Text("description")
	conflicts.String("ip_address", keySize)
	conflicts.Text("device_ids")
	conflicts.Text("device_names")
	conflicts.Text("network_ids")
	conflicts.Text("network_names")
	conflicts.Text("subnets")
	conflicts.Timestamp("detected_at").NotNull().DefaultNow()
	conflicts.Timestamp("resolved_at")
	conflicts.Text("resolved_by")
	conflicts.Text("notes")
	stmts = append(stmts, conflicts)
	stmts = append(stmts, s.CreateIndex("idx_conflicts_detected", "conflicts", "detected_at"))
	stmts = append(stmts, s.CreateIndex("idx_conflicts_ip", "conflicts", "ip_address"))
	stmts = append(stmts, s.CreateIndex("idx_conflicts_status", "conflicts", "status"))
	stmts = append(stmts, s.CreateIndex("idx_conflicts_type", "conflicts", "type"))

	credentials := s.CreateTable("credentials")
	credentials.String("id", idSize).PrimaryKey()
	credentials.Text("name").NotNull()
	credentials.Text("type").NotNull()
	credentials.Text("snmp_community")
	credentials.Text("snmp_v3_user")
	credentials.Text("snmp_v3_auth")
	credentials.Text("snmp_v3_priv")
	credentials.Text("ssh_username")
	credentials.Text("ssh_key_id")
	credentials.Text("datacenter_id")
	credentials.Text("description")
	credentials.Timestamp("created_at").DefaultNow()
	credentials.Timestamp("updated_at").DefaultNow()
	stmts = append(stmts, credentials)

	customfielddefinitions := s.CreateTable("custom_field_definitions")
	customfielddefinitions.String("id", idSize).PrimaryKey()
	customfielddefinitions.Text("name").NotNull()
	customfielddefinitions.String("key", keySize).NotNull().Unique()
	customfielddefinitions.Text("type").NotNull()
	customfielddefinitions.Boolean("required").NotNull().Default(false)
	customfielddefinitions.Text("options").Default("[]")
	customfielddefinitions.Text("description").Default("")
	customfielddefinitions.Timestamp("created_at").NotNull()
	customfielddefinitions.Timestamp("updated_at").NotNull()
	stmts = append(stmts, customfielddefinitions)
	stmts = append(stmts, s.CreateIndex("idx_cfd_key", "custom_field_definitions", "key"))

	customfieldvalues := s.CreateTable("custom_field_values")
	customfieldvalues.String("id", idSize).PrimaryKey()
	customfieldvalues.String("device_id", idSize).NotNull().References("devices", "id").OnDelete(xdal.Cascade)
	customfieldvalues.String("field_id", idSize).NotNull().References("custom_field_definitions", "id").OnDelete(xdal.Cascade)
	customfieldvalues.Text("string_value").Default("")
	customfieldvalues.Integer("number_value")
	customfieldvalues.Boolean("bool_value")
	customfieldvalues.Unique("device_id", "field_id")
	stmts = append(stmts, customfieldvalues)
	stmts = append(stmts, s.CreateIndex("idx_cfv_device", "custom_field_values", "device_id"))
	stmts = append(stmts, s.CreateIndex("idx_cfv_field", "custom_field_values", "field_id"))

	devicerelationships := s.CreateTable("device_relationships")
	devicerelationships.String("parent_id", idSize).NotNull().References("devices", "id").OnDelete(xdal.Cascade)
	devicerelationships.String("child_id", idSize).NotNull().References("devices", "id").OnDelete(xdal.Cascade)
	devicerelationships.String("type", keySize).NotNull()
	devicerelationships.Timestamp("created_at").DefaultNow()
	devicerelationships.Text("notes").Default("")
	devicerelationships.PrimaryKey("parent_id", "child_id", "type")
	stmts = append(stmts, devicerelationships)
	stmts = append(stmts, s.CreateIndex("idx_device_relationships_child", "device_relationships", "child_id"))
	stmts = append(stmts, s.CreateIndex("idx_device_relationships_parent", "device_relationships", "parent_id"))

	discovereddevices := s.CreateTable("discovered_devices")
	discovereddevices.String("id", idSize).PrimaryKey()
	discovereddevices.String("ip", keySize).NotNull()
	discovereddevices.Text("mac_address")
	discovereddevices.Text("hostname")
	discovereddevices.String("network_id", idSize).References("networks", "id")
	discovereddevices.Text("status").Default("unknown")
	discovereddevices.Integer("confidence").Default(0)
	discovereddevices.Text("os_guess")
	discovereddevices.Text("vendor")
	discovereddevices.Text("open_ports")
	discovereddevices.Text("services")
	discovereddevices.Timestamp("first_seen")
	discovereddevices.Timestamp("last_seen")
	discovereddevices.String("promoted_to_device_id", idSize).References("devices", "id")
	discovereddevices.Timestamp("promoted_at")
	discovereddevices.Timestamp("created_at").DefaultNow()
	discovereddevices.Timestamp("updated_at").DefaultNow()
	stmts = append(stmts, discovereddevices)
	stmts = append(stmts, s.CreateIndex("idx_discovered_devices_ip", "discovered_devices", "ip"))
	stmts = append(stmts, s.CreateIndex("idx_discovered_devices_network", "discovered_devices", "network_id"))
	stmts = append(stmts, s.CreateIndex("idx_discovered_devices_network_ip", "discovered_devices", "network_id", "ip").Unique())
	stmts = append(stmts, s.CreateIndex("idx_discovered_devices_promoted", "discovered_devices", "promoted_to_device_id"))

	discoveryrules := s.CreateTable("discovery_rules")
	discoveryrules.String("id", idSize).PrimaryKey()
	discoveryrules.String("network_id", idSize).Unique().References("networks", "id")
	discoveryrules.Boolean("enabled").Default(true)
	discoveryrules.Text("scan_type").Default("full")
	discoveryrules.Integer("interval_hours").Default(24)
	discoveryrules.Text("exclude_ips")
	discoveryrules.Timestamp("created_at").DefaultNow()
	discoveryrules.Timestamp("updated_at").DefaultNow()
	stmts = append(stmts, discoveryrules)

	discoveryscans := s.CreateTable("discovery_scans")
	discoveryscans.String("id", idSize).PrimaryKey()
	discoveryscans.String("network_id", idSize).References("networks", "id")
	discoveryscans.Text("status").Default("pending")
	discoveryscans.Text("scan_type").Default("full")
	discoveryscans.Integer("total_hosts").Default(0)
	discoveryscans.Integer("scanned_hosts").Default(0)
	discoveryscans.Integer("found_hosts").Default(0)
	discoveryscans.Float("progress_percent").Default(0)
	discoveryscans.Text("error_message")
	discoveryscans.Timestamp("started_at")
	discoveryscans.Timestamp("completed_at")
	discoveryscans.Timestamp("created_at").DefaultNow()
	discoveryscans.Timestamp("updated_at").DefaultNow()
	stmts = append(stmts, discoveryscans)
	stmts = append(stmts, s.CreateIndex("idx_discovery_scans_network", "discovery_scans", "network_id"))

	dnsproviderconfigs := s.CreateTable("dns_provider_configs")
	dnsproviderconfigs.String("id", idSize).PrimaryKey()
	dnsproviderconfigs.String("name", keySize).NotNull().Unique()
	dnsproviderconfigs.Text("type").NotNull()
	dnsproviderconfigs.Text("endpoint").NotNull()
	dnsproviderconfigs.Text("token").NotNull()
	dnsproviderconfigs.Text("description")
	dnsproviderconfigs.Timestamp("created_at").NotNull()
	dnsproviderconfigs.Timestamp("updated_at").NotNull()
	stmts = append(stmts, dnsproviderconfigs)

	dnszones := s.CreateTable("dns_zones")
	dnszones.String("id", idSize).PrimaryKey()
	dnszones.String("name", keySize).NotNull().Unique()
	dnszones.String("provider_id", idSize).NotNull().References("dns_provider_configs", "id").OnDelete(xdal.Cascade)
	dnszones.String("network_id", idSize).References("networks", "id").OnDelete(xdal.SetNull)
	dnszones.Boolean("auto_sync").NotNull().Default(false)
	dnszones.Boolean("create_ptr").NotNull().Default(true)
	dnszones.Text("ptr_zone")
	dnszones.Integer("ttl").NotNull().Default(3600)
	dnszones.Text("description")
	dnszones.Timestamp("last_sync_at")
	dnszones.Text("last_sync_status")
	dnszones.Text("last_sync_error")
	dnszones.Timestamp("created_at").NotNull()
	dnszones.Timestamp("updated_at").NotNull()
	stmts = append(stmts, dnszones)
	stmts = append(stmts, s.CreateIndex("idx_dns_zones_network", "dns_zones", "network_id"))
	stmts = append(stmts, s.CreateIndex("idx_dns_zones_provider", "dns_zones", "provider_id"))

	dnsrecords := s.CreateTable("dns_records")
	dnsrecords.String("id", idSize).PrimaryKey()
	dnsrecords.String("zone_id", idSize).NotNull().References("dns_zones", "id").OnDelete(xdal.Cascade)
	dnsrecords.String("device_id", idSize).References("devices", "id").OnDelete(xdal.SetNull)
	dnsrecords.String("name", keySize).NotNull()
	dnsrecords.String("type", keySize).NotNull()
	dnsrecords.Text("value").NotNull()
	dnsrecords.Integer("ttl").NotNull()
	dnsrecords.Text("sync_status").NotNull().Default("pending")
	dnsrecords.Timestamp("last_sync_at")
	dnsrecords.Text("error_message")
	dnsrecords.Timestamp("created_at").NotNull()
	dnsrecords.Timestamp("updated_at").NotNull()
	dnsrecords.String("address_id", idSize).References("addresses", "id").OnDelete(xdal.SetNull)
	dnsrecords.Unique("zone_id", "name", "type")
	stmts = append(stmts, dnsrecords)
	stmts = append(stmts, s.CreateIndex("idx_dns_records_address", "dns_records", "address_id"))
	stmts = append(stmts, s.CreateIndex("idx_dns_records_device", "dns_records", "device_id"))
	stmts = append(stmts, s.CreateIndex("idx_dns_records_zone", "dns_records", "zone_id"))

	domains := s.CreateTable("domains")
	domains.String("device_id", idSize).NotNull().References("devices", "id").OnDelete(xdal.Cascade)
	domains.String("domain", keySize).NotNull()
	domains.PrimaryKey("device_id", "domain")
	stmts = append(stmts, domains)
	stmts = append(stmts, s.CreateIndex("idx_domains_device", "domains", "device_id"))

	natmappings := s.CreateTable("nat_mappings")
	natmappings.String("id", idSize).PrimaryKey()
	natmappings.Text("name").NotNull()
	natmappings.String("external_ip", keySize).NotNull()
	natmappings.Integer("external_port").NotNull()
	natmappings.String("internal_ip", keySize).NotNull()
	natmappings.Integer("internal_port").NotNull()
	natmappings.String("protocol", keySize).NotNull().Default("tcp")
	natmappings.String("device_id", idSize).References("devices", "id").OnDelete(xdal.SetNull)
	natmappings.Text("description").Default("")
	natmappings.Boolean("enabled").NotNull().Default(true)
	natmappings.String("datacenter_id", idSize).References("datacenters", "id").OnDelete(xdal.SetNull)
	natmappings.String("network_id", idSize).References("networks", "id").OnDelete(xdal.SetNull)
	natmappings.Text("tags").Default("[]")
	natmappings.Timestamp("created_at").NotNull()
	natmappings.Timestamp("updated_at").NotNull()
	stmts = append(stmts, natmappings)
	stmts = append(stmts, s.CreateIndex("idx_nat_datacenter", "nat_mappings", "datacenter_id"))
	stmts = append(stmts, s.CreateIndex("idx_nat_device", "nat_mappings", "device_id"))
	stmts = append(stmts, s.CreateIndex("idx_nat_enabled", "nat_mappings", "enabled"))
	stmts = append(stmts, s.CreateIndex("idx_nat_external_ip", "nat_mappings", "external_ip"))
	stmts = append(stmts, s.CreateIndex("idx_nat_internal_ip", "nat_mappings", "internal_ip"))
	stmts = append(stmts, s.CreateIndex("idx_nat_network", "nat_mappings", "network_id"))
	stmts = append(stmts, s.CreateIndex("idx_nat_protocol", "nat_mappings", "protocol"))

	oauthclients := s.CreateTable("oauth_clients")
	oauthclients.String("id", idSize).PrimaryKey()
	oauthclients.Text("name").NotNull()
	oauthclients.Text("secret_hash")
	oauthclients.Text("redirect_uris").NotNull().Default("[]")
	oauthclients.Text("grant_types").NotNull().Default("[]")
	oauthclients.Text("response_types").NotNull().Default("[]")
	oauthclients.Text("token_endpoint_auth").NotNull().Default("none")
	oauthclients.Text("scope")
	oauthclients.Text("client_uri")
	oauthclients.Text("logo_uri")
	oauthclients.Boolean("is_confidential").NotNull().Default(false)
	oauthclients.String("created_by_user_id", idSize).References("users", "id")
	oauthclients.Timestamp("created_at").NotNull()
	oauthclients.Timestamp("updated_at").NotNull()
	stmts = append(stmts, oauthclients)
	stmts = append(stmts, s.CreateIndex("idx_oauth_clients_created_by", "oauth_clients", "created_by_user_id"))

	oauthauthorizationcodes := s.CreateTable("oauth_authorization_codes")
	oauthauthorizationcodes.String("code_hash", keySize).PrimaryKey()
	oauthauthorizationcodes.String("client_id", idSize).NotNull().References("oauth_clients", "id").OnDelete(xdal.Cascade)
	oauthauthorizationcodes.String("user_id", idSize).NotNull().References("users", "id").OnDelete(xdal.Cascade)
	oauthauthorizationcodes.Text("redirect_uri").NotNull()
	oauthauthorizationcodes.Text("scope")
	oauthauthorizationcodes.Text("code_challenge").NotNull()
	oauthauthorizationcodes.Text("code_challenge_method").NotNull().Default("S256")
	oauthauthorizationcodes.Timestamp("expires_at").NotNull()
	oauthauthorizationcodes.Timestamp("created_at").NotNull()
	oauthauthorizationcodes.Boolean("used").NotNull().Default(false)
	stmts = append(stmts, oauthauthorizationcodes)
	stmts = append(stmts, s.CreateIndex("idx_oauth_codes_client", "oauth_authorization_codes", "client_id"))
	stmts = append(stmts, s.CreateIndex("idx_oauth_codes_expires", "oauth_authorization_codes", "expires_at"))
	stmts = append(stmts, s.CreateIndex("idx_oauth_codes_user", "oauth_authorization_codes", "user_id"))

	oauthtokens := s.CreateTable("oauth_tokens")
	oauthtokens.String("id", idSize).PrimaryKey()
	oauthtokens.Text("token_type").NotNull()
	oauthtokens.String("token_hash", keySize).NotNull().Unique()
	oauthtokens.String("client_id", idSize).NotNull().References("oauth_clients", "id").OnDelete(xdal.Cascade)
	oauthtokens.String("user_id", idSize).NotNull().References("users", "id").OnDelete(xdal.Cascade)
	oauthtokens.Text("scope")
	oauthtokens.Timestamp("expires_at").NotNull()
	oauthtokens.Timestamp("created_at").NotNull()
	oauthtokens.Timestamp("revoked_at")
	oauthtokens.String("parent_token_id", idSize)
	stmts = append(stmts, oauthtokens)
	stmts = append(stmts, s.CreateIndex("idx_oauth_tokens_client", "oauth_tokens", "client_id"))
	stmts = append(stmts, s.CreateIndex("idx_oauth_tokens_expires", "oauth_tokens", "expires_at"))
	stmts = append(stmts, s.CreateIndex("idx_oauth_tokens_hash", "oauth_tokens", "token_hash"))
	stmts = append(stmts, s.CreateIndex("idx_oauth_tokens_parent", "oauth_tokens", "parent_token_id"))
	stmts = append(stmts, s.CreateIndex("idx_oauth_tokens_user", "oauth_tokens", "user_id"))

	permissions := s.CreateTable("permissions")
	permissions.String("id", idSize).PrimaryKey()
	permissions.String("name", keySize).NotNull().Unique()
	permissions.String("resource", keySize).NotNull()
	permissions.String("action", keySize).NotNull()
	permissions.Timestamp("created_at").NotNull()
	stmts = append(stmts, permissions)
	stmts = append(stmts, s.CreateIndex("idx_permissions_resource", "permissions", "resource", "action"))

	pooltags := s.CreateTable("pool_tags")
	pooltags.String("pool_id", idSize).NotNull().References("network_pools", "id").OnDelete(xdal.Cascade)
	pooltags.String("tag", keySize).NotNull()
	pooltags.PrimaryKey("pool_id", "tag")
	stmts = append(stmts, pooltags)
	stmts = append(stmts, s.CreateIndex("idx_pool_tags_pool", "pool_tags", "pool_id"))

	// idx_reservations_pool_ip_unique: partial index, created per dialect below
	reservations := s.CreateTable("reservations")
	reservations.String("id", idSize).PrimaryKey()
	reservations.String("pool_id", idSize).NotNull().References("network_pools", "id").OnDelete(xdal.Cascade)
	reservations.String("ip_address", keySize).NotNull()
	reservations.Text("hostname")
	reservations.Text("purpose")
	reservations.String("reserved_by", keySize).NotNull()
	reservations.Timestamp("reserved_at").NotNull().DefaultNow()
	reservations.Timestamp("expires_at")
	reservations.String("status", keySize).NotNull().Default("active")
	reservations.Text("notes")
	reservations.Timestamp("created_at").NotNull().DefaultNow()
	reservations.Timestamp("updated_at").NotNull().DefaultNow()
	stmts = append(stmts, reservations)
	stmts = append(stmts, s.CreateIndex("idx_reservations_expires_at", "reservations", "expires_at"))
	stmts = append(stmts, s.CreateIndex("idx_reservations_ip_address", "reservations", "pool_id", "ip_address"))
	stmts = append(stmts, s.CreateIndex("idx_reservations_pool_id", "reservations", "pool_id"))
	stmts = append(stmts, s.CreateIndex("idx_reservations_reserved_by", "reservations", "reserved_by"))
	stmts = append(stmts, s.CreateIndex("idx_reservations_status", "reservations", "status"))

	roles := s.CreateTable("roles")
	roles.String("id", idSize).PrimaryKey()
	roles.String("name", keySize).NotNull().Unique()
	roles.Text("description")
	roles.Boolean("is_system").NotNull().Default(false)
	roles.Timestamp("created_at").NotNull()
	roles.Timestamp("updated_at").NotNull()
	stmts = append(stmts, roles)

	rolepermissions := s.CreateTable("role_permissions")
	rolepermissions.String("role_id", idSize).NotNull().References("roles", "id").OnDelete(xdal.Cascade)
	rolepermissions.String("permission_id", idSize).NotNull().References("permissions", "id").OnDelete(xdal.Cascade)
	rolepermissions.Timestamp("created_at").NotNull()
	rolepermissions.PrimaryKey("role_id", "permission_id")
	stmts = append(stmts, rolepermissions)
	stmts = append(stmts, s.CreateIndex("idx_role_permissions_permission", "role_permissions", "permission_id"))
	stmts = append(stmts, s.CreateIndex("idx_role_permissions_role", "role_permissions", "role_id"))

	scanprofiles := s.CreateTable("scan_profiles")
	scanprofiles.String("id", idSize).PrimaryKey()
	scanprofiles.String("name", keySize).NotNull().Unique()
	scanprofiles.Text("scan_type").NotNull()
	scanprofiles.Text("ports")
	scanprofiles.Boolean("enable_snmp").Default(false)
	scanprofiles.Boolean("enable_ssh").Default(false)
	scanprofiles.Integer("timeout_sec").Default(30)
	scanprofiles.Integer("max_workers").Default(10)
	scanprofiles.Text("description")
	scanprofiles.Timestamp("created_at").DefaultNow()
	scanprofiles.Timestamp("updated_at").DefaultNow()
	scanprofiles.Text("snmp_credential_id")
	scanprofiles.Text("ssh_credential_id")
	stmts = append(stmts, scanprofiles)

	scheduledscans := s.CreateTable("scheduled_scans")
	scheduledscans.String("id", idSize).PrimaryKey()
	scheduledscans.Text("network_id").NotNull()
	scheduledscans.Text("profile_id").NotNull()
	scheduledscans.Text("name").NotNull()
	scheduledscans.Text("cron_expression").NotNull()
	scheduledscans.Boolean("enabled").Default(true)
	scheduledscans.Text("description")
	scheduledscans.Timestamp("last_run_at")
	scheduledscans.Timestamp("next_run_at")
	scheduledscans.Timestamp("created_at").DefaultNow()
	scheduledscans.Timestamp("updated_at").DefaultNow()
	stmts = append(stmts, scheduledscans)

	sessions := s.CreateTable("sessions")
	sessions.String("token", keySize).PrimaryKey()
	sessions.String("user_id", idSize).NotNull().References("users", "id").OnDelete(xdal.Cascade)
	sessions.Text("username").NotNull()
	sessions.Boolean("is_admin").NotNull().Default(false)
	sessions.Timestamp("created_at").NotNull().DefaultNow()
	sessions.Timestamp("expires_at").NotNull()
	stmts = append(stmts, sessions)
	stmts = append(stmts, s.CreateIndex("idx_sessions_expires", "sessions", "expires_at"))
	stmts = append(stmts, s.CreateIndex("idx_sessions_user", "sessions", "user_id"))

	sshhostkeys := s.CreateTable("ssh_host_keys")
	sshhostkeys.String("host", keySize).PrimaryKey()
	sshhostkeys.Blob("key_data").NotNull()
	sshhostkeys.Timestamp("created_at").DefaultNow()
	sshhostkeys.Timestamp("updated_at").DefaultNow()
	stmts = append(stmts, sshhostkeys)

	tags := s.CreateTable("tags")
	tags.String("device_id", idSize).NotNull().References("devices", "id").OnDelete(xdal.Cascade)
	tags.String("tag", keySize).NotNull()
	tags.PrimaryKey("device_id", "tag")
	stmts = append(stmts, tags)
	stmts = append(stmts, s.CreateIndex("idx_tags_device", "tags", "device_id"))

	userroles := s.CreateTable("user_roles")
	userroles.String("user_id", idSize).NotNull().References("users", "id").OnDelete(xdal.Cascade)
	userroles.String("role_id", idSize).NotNull().References("roles", "id").OnDelete(xdal.Cascade)
	userroles.Timestamp("created_at").NotNull()
	userroles.PrimaryKey("user_id", "role_id")
	stmts = append(stmts, userroles)
	stmts = append(stmts, s.CreateIndex("idx_user_roles_role", "user_roles", "role_id"))
	stmts = append(stmts, s.CreateIndex("idx_user_roles_user", "user_roles", "user_id"))

	utilizationsnapshots := s.CreateTable("utilization_snapshots")
	utilizationsnapshots.String("id", idSize).PrimaryKey()
	utilizationsnapshots.String("type", keySize).NotNull()
	utilizationsnapshots.String("resource_id", idSize).NotNull()
	utilizationsnapshots.Text("resource_name").NotNull()
	utilizationsnapshots.Integer("total_ips").NotNull()
	utilizationsnapshots.Integer("used_ips").NotNull()
	utilizationsnapshots.Float("utilization").NotNull()
	utilizationsnapshots.Timestamp("timestamp").NotNull()
	utilizationsnapshots.Timestamp("created_at").NotNull().DefaultNow()
	stmts = append(stmts, utilizationsnapshots)
	stmts = append(stmts, s.CreateIndex("idx_snapshots_timestamp", "utilization_snapshots", "timestamp"))
	stmts = append(stmts, s.CreateIndex("idx_snapshots_type_resource", "utilization_snapshots", "type", "resource_id"))
	stmts = append(stmts, s.CreateIndex("idx_snapshots_type_timestamp", "utilization_snapshots", "type", "timestamp"))

	webhooks := s.CreateTable("webhooks")
	webhooks.String("id", idSize).PrimaryKey()
	webhooks.Text("name").NotNull()
	webhooks.Text("url").NotNull()
	webhooks.Text("secret")
	webhooks.Text("events").NotNull()
	webhooks.Boolean("active").NotNull().Default(true)
	webhooks.Text("description")
	webhooks.Timestamp("created_at").NotNull()
	webhooks.Timestamp("updated_at").NotNull()
	webhooks.Text("created_by")
	stmts = append(stmts, webhooks)
	stmts = append(stmts, s.CreateIndex("idx_webhooks_active", "webhooks", "active"))

	webhookdeliveries := s.CreateTable("webhook_deliveries")
	webhookdeliveries.String("id", idSize).PrimaryKey()
	webhookdeliveries.String("webhook_id", idSize).NotNull().References("webhooks", "id").OnDelete(xdal.Cascade)
	webhookdeliveries.Text("event_type").NotNull()
	webhookdeliveries.Text("payload").NotNull()
	webhookdeliveries.Integer("response_code")
	webhookdeliveries.Text("response_body")
	webhookdeliveries.Text("error")
	webhookdeliveries.Integer("duration_ms")
	webhookdeliveries.String("status", keySize).NotNull()
	webhookdeliveries.Integer("attempt_number").NotNull().Default(1)
	webhookdeliveries.Timestamp("next_retry")
	webhookdeliveries.Timestamp("created_at").NotNull()
	stmts = append(stmts, webhookdeliveries)
	stmts = append(stmts, s.CreateIndex("idx_webhook_deliveries_created_at", "webhook_deliveries", "created_at"))
	stmts = append(stmts, s.CreateIndex("idx_webhook_deliveries_next_retry", "webhook_deliveries", "next_retry"))
	stmts = append(stmts, s.CreateIndex("idx_webhook_deliveries_status", "webhook_deliveries", "status"))
	stmts = append(stmts, s.CreateIndex("idx_webhook_deliveries_webhook_id", "webhook_deliveries", "webhook_id"))

	return stmts
}

// createPartialIndexes adds the two unique indexes that only cover some rows.
// SQLite and PostgreSQL support partial indexes; MySQL gets the same effect
// from a functional key part that is NULL outside the covered rows, since a
// unique index allows any number of NULLs.
func createPartialIndexes(ctx context.Context, tx *migrate.Tx) error {
	d := tx.Dialect()
	q := d.QuoteIdentifier
	var stmts []string
	switch d.Name() {
	case xdal.DialectMySQL:
		stmts = []string{
			"CREATE UNIQUE INDEX " + q("idx_conflicts_active_duplicate_ip") + " ON " + q("conflicts") +
				" ((CASE WHEN " + q("type") + " = 'duplicate_ip' AND " + q("status") + " = 'active' THEN " + q("ip_address") + " END))",
			"CREATE UNIQUE INDEX " + q("idx_reservations_pool_ip_unique") + " ON " + q("reservations") +
				" ((CASE WHEN " + q("status") + " = 'active' THEN CONCAT(" + q("pool_id") + ", '/', " + q("ip_address") + ") END))",
		}
	default:
		stmts = []string{
			"CREATE UNIQUE INDEX " + q("idx_conflicts_active_duplicate_ip") + " ON " + q("conflicts") + " (" + q("ip_address") + ")" +
				" WHERE " + q("type") + " = 'duplicate_ip' AND " + q("status") + " = 'active'",
			"CREATE UNIQUE INDEX " + q("idx_reservations_pool_ip_unique") + " ON " + q("reservations") + " (" + q("pool_id") + ", " + q("ip_address") + ")" +
				" WHERE " + q("status") + " = 'active'",
		}
	}
	return tx.ExecSQL(ctx, stmts...)
}

// baselinePermissions are the permissions a new database starts with:
// name, resource, action.
var baselinePermissions = [][3]string{
	{"apikey:create", "apikeys", "create"},
	{"apikey:delete", "apikeys", "delete"},
	{"apikey:list", "apikeys", "list"},
	{"apikey:read", "apikeys", "read"},
	{"apikey:update", "apikeys", "update"},
	{"audit:export", "audit", "export"},
	{"audit:list", "audit", "list"},
	{"audit:read", "audit", "read"},
	{"circuits:create", "circuits", "create"},
	{"circuits:delete", "circuits", "delete"},
	{"circuits:list", "circuits", "list"},
	{"circuits:read", "circuits", "read"},
	{"circuits:update", "circuits", "update"},
	{"conflict:delete", "conflicts", "delete"},
	{"conflict:detect", "conflicts", "detect"},
	{"conflict:list", "conflicts", "list"},
	{"conflict:read", "conflicts", "read"},
	{"conflict:resolve", "conflicts", "resolve"},
	{"credential:create", "credentials", "create"},
	{"credential:delete", "credentials", "delete"},
	{"credential:list", "credentials", "list"},
	{"credential:read", "credentials", "read"},
	{"credential:update", "credentials", "update"},
	{"custom-fields:create", "custom-fields", "create"},
	{"custom-fields:delete", "custom-fields", "delete"},
	{"custom-fields:list", "custom-fields", "list"},
	{"custom-fields:read", "custom-fields", "read"},
	{"custom-fields:update", "custom-fields", "update"},
	{"dashboard:read", "dashboard", "read"},
	{"datacenter:create", "datacenters", "create"},
	{"datacenter:delete", "datacenters", "delete"},
	{"datacenter:list", "datacenters", "list"},
	{"datacenter:read", "datacenters", "read"},
	{"datacenter:update", "datacenters", "update"},
	{"device:create", "devices", "create"},
	{"device:delete", "devices", "delete"},
	{"device:list", "devices", "list"},
	{"device:read", "devices", "read"},
	{"device:update", "devices", "update"},
	{"discovery:create", "discovery", "create"},
	{"discovery:delete", "discovery", "delete"},
	{"discovery:list", "discovery", "list"},
	{"discovery:read", "discovery", "read"},
	{"discovery:update", "discovery", "update"},
	{"dns-provider:create", "dns-provider", "create"},
	{"dns-provider:delete", "dns-provider", "delete"},
	{"dns-provider:list", "dns-provider", "list"},
	{"dns-provider:read", "dns-provider", "read"},
	{"dns-provider:test", "dns-provider", "test"},
	{"dns-provider:update", "dns-provider", "update"},
	{"dns-zone:create", "dns-zone", "create"},
	{"dns-zone:delete", "dns-zone", "delete"},
	{"dns-zone:import", "dns-zone", "import"},
	{"dns-zone:list", "dns-zone", "list"},
	{"dns-zone:read", "dns-zone", "read"},
	{"dns-zone:sync", "dns-zone", "sync"},
	{"dns-zone:update", "dns-zone", "update"},
	{"dns:create", "dns", "create"},
	{"dns:delete", "dns", "delete"},
	{"dns:list", "dns", "list"},
	{"dns:read", "dns", "read"},
	{"dns:sync", "dns", "sync"},
	{"dns:update", "dns", "update"},
	{"logs:export", "logs", "export"},
	{"logs:list", "logs", "list"},
	{"logs:read", "logs", "read"},
	{"metrics:read", "metrics", "read"},
	{"nat:create", "nat", "create"},
	{"nat:delete", "nat", "delete"},
	{"nat:list", "nat", "list"},
	{"nat:read", "nat", "read"},
	{"nat:update", "nat", "update"},
	{"network:create", "networks", "create"},
	{"network:delete", "networks", "delete"},
	{"network:list", "networks", "list"},
	{"network:read", "networks", "read"},
	{"network:update", "networks", "update"},
	{"pool:create", "pools", "create"},
	{"pool:delete", "pools", "delete"},
	{"pool:list", "pools", "list"},
	{"pool:read", "pools", "read"},
	{"pool:update", "pools", "update"},
	{"relationship:create", "relationships", "create"},
	{"relationship:delete", "relationships", "delete"},
	{"relationship:list", "relationships", "list"},
	{"relationship:read", "relationships", "read"},
	{"relationship:update", "relationships", "update"},
	{"reservation:create", "reservations", "create"},
	{"reservation:delete", "reservations", "delete"},
	{"reservation:list", "reservations", "list"},
	{"reservation:read", "reservations", "read"},
	{"reservation:update", "reservations", "update"},
	{"role:create", "roles", "create"},
	{"role:delete", "roles", "delete"},
	{"role:list", "roles", "list"},
	{"role:read", "roles", "read"},
	{"role:update", "roles", "update"},
	{"scan-profile:create", "scan-profiles", "create"},
	{"scan-profile:delete", "scan-profiles", "delete"},
	{"scan-profile:list", "scan-profiles", "list"},
	{"scan-profile:read", "scan-profiles", "read"},
	{"scan-profile:update", "scan-profiles", "update"},
	{"scheduled-scan:create", "scheduled-scans", "create"},
	{"scheduled-scan:delete", "scheduled-scans", "delete"},
	{"scheduled-scan:list", "scheduled-scans", "list"},
	{"scheduled-scan:read", "scheduled-scans", "read"},
	{"scheduled-scan:update", "scheduled-scans", "update"},
	{"search:read", "search", "read"},
	{"user:create", "users", "create"},
	{"user:delete", "users", "delete"},
	{"user:list", "users", "list"},
	{"user:read", "users", "read"},
	{"user:update", "users", "update"},
	{"webhooks:create", "webhooks", "create"},
	{"webhooks:delete", "webhooks", "delete"},
	{"webhooks:list", "webhooks", "list"},
	{"webhooks:read", "webhooks", "read"},
	{"webhooks:update", "webhooks", "update"},
}

// baselineRoles are the built-in roles: name, description. The admin role is
// granted every permission.
var baselineRoles = [][2]string{
	{"admin", "Full administrative access"},
	{"operator", "Can manage devices, networks, and discovery"},
	{"viewer", "Read-only access"},
}

// baselineRolePermissions grants the other built-in roles their permissions.
var baselineRolePermissions = map[string][]string{
	"operator": {
		"audit:list", "audit:read", "circuits:create", "circuits:list", "circuits:read",
		"circuits:update", "conflict:detect", "conflict:list", "conflict:read", "credential:create",
		"credential:list", "credential:read", "credential:update", "custom-fields:list",
		"custom-fields:read", "dashboard:read", "datacenter:list", "datacenter:read", "device:create",
		"device:list", "device:read", "device:update", "discovery:create", "discovery:list",
		"discovery:read", "discovery:update", "dns-provider:list", "dns-provider:read",
		"dns-provider:test", "dns-zone:list", "dns-zone:read", "dns-zone:sync", "dns:list", "dns:read",
		"dns:sync", "logs:list", "logs:read", "metrics:read", "nat:create", "nat:list", "nat:read",
		"nat:update", "network:create", "network:list", "network:read", "network:update", "pool:create",
		"pool:list", "pool:read", "pool:update", "relationship:create", "relationship:list",
		"relationship:read", "relationship:update", "reservation:create", "reservation:list",
		"reservation:read", "reservation:update", "scan-profile:create", "scan-profile:list",
		"scan-profile:read", "scan-profile:update", "scheduled-scan:create", "scheduled-scan:list",
		"scheduled-scan:read", "scheduled-scan:update", "search:read", "webhooks:list", "webhooks:read",
	},
	"viewer": {
		"circuits:list", "circuits:read", "conflict:list", "conflict:read", "credential:list",
		"credential:read", "custom-fields:list", "custom-fields:read", "dashboard:read",
		"datacenter:list", "datacenter:read", "device:list", "device:read", "discovery:list",
		"discovery:read", "dns-provider:list", "dns-provider:read", "dns-zone:list", "dns-zone:read",
		"dns:list", "dns:read", "metrics:read", "nat:list", "nat:read", "network:list", "network:read",
		"pool:list", "pool:read", "relationship:list", "relationship:read", "reservation:list",
		"reservation:read", "scan-profile:list", "scan-profile:read", "scheduled-scan:list",
		"scheduled-scan:read", "search:read", "webhooks:list", "webhooks:read",
	},
}

// baselineScanProfiles are the default discovery scan profiles.
var baselineScanProfiles = []struct {
	id, name, scanType, ports string
	snmp, ssh                 bool
	timeout, workers          int
}{
	{"default-quick", "Quick Scan", "quick", "[22,80,443,3389]", false, false, 10, 50},
	{"default-full", "Full Scan", "full", "[22,80,443,3389,8080,8443]", true, false, 30, 20},
	{"default-deep", "Deep Scan", "deep", "[22,80,443,3389,8080,8443,3306,5432]", true, true, 60, 10},
}

// seedBaseline inserts the rows the legacy track seeded.
func seedBaseline(ctx context.Context, tx *migrate.Tx) error {
	qb := xdal.NewQueryBuilder(tx.Dialect())
	now := time.Now().UTC()
	exec := func(b interface {
		Build() (string, []interface{}, error)
	}) error {
		q, args, err := b.Build()
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, q, args...)
		return err
	}

	permIDs := make(map[string]string, len(baselinePermissions))
	for _, p := range baselinePermissions {
		id := newUUID()
		permIDs[p[0]] = id
		if err := exec(qb.Insert("permissions").Set("id", id).Set("name", p[0]).Set("resource", p[1]).
			Set("action", p[2]).Set("created_at", now)); err != nil {
			return fmt.Errorf("seed permission %s: %w", p[0], err)
		}
	}
	grant := func(roleID, perm string) error {
		pid, ok := permIDs[perm]
		if !ok {
			return fmt.Errorf("seed: unknown permission %s", perm)
		}
		return exec(qb.Insert("role_permissions").Set("role_id", roleID).Set("permission_id", pid).Set("created_at", now))
	}
	for _, r := range baselineRoles {
		id := newUUID()
		if err := exec(qb.Insert("roles").Set("id", id).Set("name", r[0]).Set("description", r[1]).
			Set("is_system", true).Set("created_at", now).Set("updated_at", now)); err != nil {
			return fmt.Errorf("seed role %s: %w", r[0], err)
		}
		perms := baselineRolePermissions[r[0]]
		if r[0] == "admin" {
			perms = make([]string, 0, len(baselinePermissions))
			for _, p := range baselinePermissions {
				perms = append(perms, p[0])
			}
		}
		for _, p := range perms {
			if err := grant(id, p); err != nil {
				return fmt.Errorf("seed role %s: %w", r[0], err)
			}
		}
	}
	for _, p := range baselineScanProfiles {
		if err := exec(qb.Insert("scan_profiles").Set("id", p.id).Set("name", p.name).Set("scan_type", p.scanType).
			Set("ports", p.ports).Set("enable_snmp", p.snmp).Set("enable_ssh", p.ssh).Set("timeout_sec", p.timeout).
			Set("max_workers", p.workers).Set("description", "").Set("created_at", now).Set("updated_at", now)); err != nil {
			return fmt.Errorf("seed scan profile %s: %w", p.id, err)
		}
	}
	return nil
}
