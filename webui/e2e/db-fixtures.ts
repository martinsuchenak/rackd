import { randomUUID } from 'node:crypto';
import { execFileSync } from 'node:child_process';
import path from 'node:path';

const DEFAULT_DB_PATH = path.resolve(process.cwd(), '../.tmp/e2e-data/rackd.db');
const DB_PATH = process.env.E2E_DB_PATH || DEFAULT_DB_PATH;

function quote(value: string): string {
  return `'${value.replaceAll('\'', '\'\'')}'`;
}

function nullable(value?: string | null): string {
  return value ? quote(value) : 'NULL';
}

function runSQL(sql: string): string {
  return execFileSync('sqlite3', [DB_PATH, sql], { encoding: 'utf8' }).trim();
}

export function getIDByName(table: string, name: string): string {
  const id = runSQL(`SELECT id FROM ${table} WHERE name = ${quote(name)} LIMIT 1;`);
  if (!id) {
    throw new Error(`No ${table} row found for name ${name}`);
  }
  return id;
}

export function insertDiscoveredDevice(values: {
  networkName: string;
  ip: string;
  hostname?: string;
  macAddress?: string;
  osGuess?: string;
  vendor?: string;
  confidence?: number;
  openPorts?: number[];
  services?: Array<{ port: number; service: string; version?: string }>;
}): string {
  const id = randomUUID();
  const now = new Date().toISOString();
  const networkID = getIDByName('networks', values.networkName);

  runSQL(`
    INSERT INTO discovered_devices (
      id, ip, mac_address, hostname, network_id, status, confidence, os_guess, vendor,
      open_ports, services, first_seen, last_seen, created_at, updated_at
    ) VALUES (
      ${quote(id)},
      ${quote(values.ip)},
      ${nullable(values.macAddress)},
      ${nullable(values.hostname)},
      ${quote(networkID)},
      'up',
      ${values.confidence ?? 8},
      ${nullable(values.osGuess)},
      ${nullable(values.vendor)},
      ${quote(JSON.stringify(values.openPorts ?? []))},
      ${quote(JSON.stringify(values.services ?? []))},
      ${quote(now)},
      ${quote(now)},
      ${quote(now)},
      ${quote(now)}
    );
  `);

  return id;
}

export function insertDNSRecord(values: {
  zoneName: string;
  name: string;
  type: string;
  value: string;
  ttl?: number;
  syncStatus?: 'pending' | 'synced' | 'failed';
  deviceID?: string | null;
  addressID?: string | null;
  errorMessage?: string | null;
}): string {
  const id = randomUUID();
  const now = new Date().toISOString();
  const zoneID = getIDByName('dns_zones', values.zoneName);

  runSQL(`
    INSERT INTO dns_records (
      id, zone_id, device_id, address_id, name, type, value, ttl, sync_status,
      last_sync_at, error_message, created_at, updated_at
    ) VALUES (
      ${quote(id)},
      ${quote(zoneID)},
      ${nullable(values.deviceID)},
      ${nullable(values.addressID)},
      ${quote(values.name)},
      ${quote(values.type)},
      ${quote(values.value)},
      ${values.ttl ?? 3600},
      ${quote(values.syncStatus ?? 'synced')},
      ${quote(now)},
      ${nullable(values.errorMessage)},
      ${quote(now)},
      ${quote(now)}
    );
  `);

  return id;
}

export function insertDiscoveryScan(values: {
  networkName: string;
  scanType?: 'quick' | 'full' | 'deep';
  status?: 'pending' | 'running' | 'completed' | 'failed';
  totalHosts?: number;
  scannedHosts?: number;
  foundHosts?: number;
}): string {
  const id = randomUUID();
  const now = new Date().toISOString();
  const networkID = getIDByName('networks', values.networkName);

  runSQL(`
    INSERT INTO discovery_scans (
      id, network_id, status, scan_type, total_hosts, scanned_hosts, found_hosts,
      progress_percent, error_message, started_at, completed_at, created_at, updated_at
    ) VALUES (
      ${quote(id)},
      ${quote(networkID)},
      ${quote(values.status ?? 'completed')},
      ${quote(values.scanType ?? 'quick')},
      ${values.totalHosts ?? 1},
      ${values.scannedHosts ?? 1},
      ${values.foundHosts ?? 1},
      100,
      '',
      ${quote(now)},
      ${quote(now)},
      ${quote(now)},
      ${quote(now)}
    );
  `);

  return id;
}
