CREATE TABLE IF NOT EXISTS ussd_pins (
    phone_number VARCHAR(64) PRIMARY KEY,
    pin_hash VARCHAR(255) NOT NULL,
    failed_attempts INT NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ussd_pins_phone ON ussd_pins(phone_number);
