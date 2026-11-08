-- SMS menu conversations. One open conversation per rider phone number.
-- steps holds the accumulated replies in USSD form (for example "2*1"), so the
-- same menu tree serves both channels. Idle conversations expire in the service.
CREATE TABLE IF NOT EXISTS sms_sessions (
    phone_number VARCHAR(64) PRIMARY KEY,
    steps TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
