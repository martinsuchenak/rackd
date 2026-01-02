-- Device Manager SQLite Schema

-- Devices table (main entity)
CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    make_model TEXT,
    os TEXT,
    location TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index on device name for fast lookups
CREATE INDEX IF NOT EXISTS idx_devices_name ON devices(name);

-- Addresses table (one-to-many with devices)
CREATE TABLE IF NOT EXISTS addresses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id TEXT NOT NULL,
    ip TEXT NOT NULL,
    port INTEGER,
    type TEXT CHECK(type IN ('ipv4', 'ipv6')) DEFAULT 'ipv4',
    label TEXT,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

-- Create index on device_id for addresses
CREATE INDEX IF NOT EXISTS idx_addresses_device_id ON addresses(device_id);

-- Tags table (one-to-many with devices)
CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id TEXT NOT NULL,
    tag TEXT NOT NULL,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

-- Create indexes for tags
CREATE INDEX IF NOT EXISTS idx_tags_device_id ON tags(device_id);
CREATE INDEX IF NOT EXISTS idx_tags_tag ON tags(tag);

-- Domains table (one-to-many with devices)
CREATE TABLE IF NOT EXISTS domains (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id TEXT NOT NULL,
    domain TEXT NOT NULL,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

-- Create indexes for domains
CREATE INDEX IF NOT EXISTS idx_domains_device_id ON domains(device_id);
CREATE INDEX IF NOT EXISTS idx_domains_domain ON domains(domain);

-- Device relationships table (many-to-many self-reference)
CREATE TABLE IF NOT EXISTS device_relationships (
    parent_id TEXT NOT NULL,
    child_id TEXT NOT NULL,
    relationship_type TEXT NOT NULL DEFAULT 'related',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (parent_id, child_id, relationship_type),
    FOREIGN KEY (parent_id) REFERENCES devices(id) ON DELETE CASCADE,
    FOREIGN KEY (child_id) REFERENCES devices(id) ON DELETE CASCADE,
    CHECK (parent_id != child_id)  -- Prevent self-relationships
);

-- Create index for relationship lookups
CREATE INDEX IF NOT EXISTS idx_relationships_parent ON device_relationships(parent_id);
CREATE INDEX IF NOT EXISTS idx_relationships_child ON device_relationships(child_id);
CREATE INDEX IF NOT EXISTS idx_relationships_type ON device_relationships(relationship_type);

-- Trigger to update updated_at timestamp
CREATE TRIGGER IF NOT EXISTS update_devices_timestamp
AFTER UPDATE ON devices
FOR EACH ROW
BEGIN
    UPDATE devices SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
