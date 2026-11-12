-- SMS language per rider phone number. "en" (English) or "sw" (Kiswahili).
-- Set by texting SW or EN to the gateway. A missing row means English.
-- It also applies to lost-bike texts sent to the rider's contacts.
CREATE TABLE IF NOT EXISTS sms_preferences (
    phone_number VARCHAR(64) PRIMARY KEY,
    language VARCHAR(8) NOT NULL DEFAULT 'en',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
