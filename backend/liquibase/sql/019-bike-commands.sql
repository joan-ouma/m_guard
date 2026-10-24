-- An immobilizer command changes the bike's state only after the device confirms it.
CREATE TABLE IF NOT EXISTS bike_commands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bike_id UUID NOT NULL REFERENCES bikes(id) ON DELETE CASCADE,
    imei VARCHAR(32) NOT NULL,
    action VARCHAR(16) NOT NULL CHECK (action IN ('IMMOBILIZE', 'REACTIVATE')),
    ts BIGINT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'CONFIRMED', 'FAILED')),
    attempts INT NOT NULL DEFAULT 1,
    last_sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at TIMESTAMPTZ
);

-- At most one command awaiting confirmation per bike.
CREATE UNIQUE INDEX IF NOT EXISTS uq_bike_commands_one_pending ON bike_commands(bike_id) WHERE status = 'PENDING';
CREATE INDEX IF NOT EXISTS idx_bike_commands_pending_imei_ts ON bike_commands(imei, ts) WHERE status = 'PENDING';
