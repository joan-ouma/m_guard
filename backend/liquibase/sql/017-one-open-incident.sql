-- A rider may have only one incident without an OB number (unfiled) at a time.
CREATE UNIQUE INDEX IF NOT EXISTS uq_theft_incidents_one_open ON theft_incidents(user_id) WHERE ob_number IS NULL;
