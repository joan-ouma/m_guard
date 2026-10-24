CREATE TABLE IF NOT EXISTS theft_incidents (
    id VARCHAR(32) PRIMARY KEY,
    user_id VARCHAR(32) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bike_id VARCHAR(32) REFERENCES bikes(id) ON DELETE SET NULL,
    mode VARCHAR(32) NOT NULL CHECK (mode IN ('violent_robbery', 'theft_from_parking', 'theft_unattended', 'hijacking', 'other')),
    lost_at TIMESTAMPTZ NOT NULL,
    location_lost TEXT NOT NULL,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    ob_number VARCHAR(32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_theft_incidents_user_created ON theft_incidents(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS incident_attachments (
    incident_id VARCHAR(32) PRIMARY KEY REFERENCES theft_incidents(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    content_type VARCHAR(64) NOT NULL,
    size_bytes INT NOT NULL,
    content BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_contacts (
    id VARCHAR(32) PRIMARY KEY,
    user_id VARCHAR(32) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(120) NOT NULL,
    phone VARCHAR(32) NOT NULL,
    category VARCHAR(16) NOT NULL CHECK (category IN ('emergency', 'insurance')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_contacts_user_created ON user_contacts(user_id, created_at DESC);
