CREATE TABLE IF NOT EXISTS trackers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    imei VARCHAR(32) UNIQUE NOT NULL,
    model VARCHAR(64) NOT NULL DEFAULT 'MGuard-T100 GPS',
    status VARCHAR(32) NOT NULL DEFAULT 'AVAILABLE',
    last_online_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trackers_imei ON trackers(imei);
CREATE INDEX IF NOT EXISTS idx_trackers_status ON trackers(status);
