-- Device registry schema
CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    address TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Device metadata (key-value pairs)
CREATE TABLE IF NOT EXISTS device_metadata (
    device_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (device_id, key),
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

-- Index for faster lookups
CREATE INDEX IF NOT EXISTS idx_device_name ON devices(name);
CREATE INDEX IF NOT EXISTS idx_device_address ON devices(address);
CREATE INDEX IF NOT EXISTS idx_metadata_key ON device_metadata(key);
