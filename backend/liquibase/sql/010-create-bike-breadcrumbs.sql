CREATE TABLE IF NOT EXISTS bike_breadcrumbs (
    id BIGSERIAL PRIMARY KEY,
    bike_id UUID NOT NULL REFERENCES bikes(id) ON DELETE CASCADE,
    latitude DOUBLE PRECISION NOT NULL,
    longitude DOUBLE PRECISION NOT NULL,
    speed_kmh DOUBLE PRECISION DEFAULT 0.0,
    heading_deg DOUBLE PRECISION DEFAULT 0.0,
    ignition BOOLEAN DEFAULT FALSE,
    battery_percent INT DEFAULT 100,
    address TEXT,
    geofence_breach BOOLEAN DEFAULT FALSE,
    event_type VARCHAR(64) DEFAULT 'ROUTINE',
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bike_breadcrumbs_bike_time ON bike_breadcrumbs(bike_id, recorded_at DESC);
