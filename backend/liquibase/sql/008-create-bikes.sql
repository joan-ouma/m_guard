CREATE TABLE IF NOT EXISTS bikes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    registration_number VARCHAR(32) NOT NULL,
    label VARCHAR(128),
    make VARCHAR(64),
    model VARCHAR(64),
    year INT,
    color VARCHAR(64),
    imei VARCHAR(32) NOT NULL,
    status VARCHAR(32) DEFAULT 'STOPPED',
    ignition BOOLEAN DEFAULT FALSE,
    speed_kmh DOUBLE PRECISION DEFAULT 0.0,
    heading_deg DOUBLE PRECISION DEFAULT 0.0,
    latitude DOUBLE PRECISION DEFAULT 0.0,
    longitude DOUBLE PRECISION DEFAULT 0.0,
    current_address TEXT,
    sub_county VARCHAR(128),
    county VARCHAR(128),
    battery_percent INT DEFAULT 0,
    signal_quality_csq INT DEFAULT 0,
    last_telemetry_at TIMESTAMPTZ,
    geofence_id UUID REFERENCES geofences(id) ON DELETE SET NULL,
    driver_contact_id UUID REFERENCES driver_contacts(id) ON DELETE SET NULL,
    is_immobilized BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bikes_user_id ON bikes(user_id);
CREATE INDEX IF NOT EXISTS idx_bikes_imei ON bikes(imei);
CREATE INDEX IF NOT EXISTS idx_bikes_reg_number ON bikes(LOWER(registration_number));
