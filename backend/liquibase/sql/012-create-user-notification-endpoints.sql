CREATE TABLE IF NOT EXISTS user_notification_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint_arn TEXT NOT NULL,
    device_token TEXT NOT NULL,
    platform VARCHAR(32) NOT NULL DEFAULT 'GCM',
    phone_number VARCHAR(32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_user_device_token UNIQUE (user_id, device_token)
);

CREATE INDEX IF NOT EXISTS idx_user_notification_endpoints_user_id ON user_notification_endpoints(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_notification_endpoints_user_token ON user_notification_endpoints(user_id, device_token);
