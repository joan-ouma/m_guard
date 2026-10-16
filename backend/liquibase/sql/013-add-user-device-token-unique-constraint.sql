DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_user_device_token'
    ) THEN
        ALTER TABLE user_notification_endpoints
        ADD CONSTRAINT uq_user_device_token UNIQUE (user_id, device_token);
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_notification_endpoints_user_token
ON user_notification_endpoints(user_id, device_token);
