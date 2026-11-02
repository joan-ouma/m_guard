-- Guided chatbot history. Rows are purged after 90 days by the chat retention job.
CREATE TABLE IF NOT EXISTS chat_messages (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(8) NOT NULL CHECK (role IN ('rider', 'bot')),
    text TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_user_time ON chat_messages(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_messages_created ON chat_messages(created_at);
