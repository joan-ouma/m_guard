-- A report stays open until the rider marks the bike as found.
ALTER TABLE theft_incidents ADD COLUMN IF NOT EXISTS resolved_at TIMESTAMPTZ;

DROP INDEX IF EXISTS uq_theft_incidents_one_open;
CREATE UNIQUE INDEX IF NOT EXISTS uq_theft_incidents_one_open ON theft_incidents(user_id) WHERE resolved_at IS NULL;
