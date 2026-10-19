CREATE TABLE IF NOT EXISTS bike_events (
    id BIGSERIAL PRIMARY KEY,
    bike_id UUID NOT NULL REFERENCES bikes(id) ON DELETE CASCADE,
    event_type VARCHAR(64) NOT NULL,
    latitude DOUBLE PRECISION DEFAULT 0.0,
    longitude DOUBLE PRECISION DEFAULT 0.0,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bike_events_bike_time ON bike_events(bike_id, recorded_at DESC);
