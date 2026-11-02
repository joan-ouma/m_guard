-- Daily count of OpenAI calls made by the chatbot (one row per East Africa Time day).
CREATE TABLE IF NOT EXISTS chat_ai_usage (
    day DATE PRIMARY KEY,
    calls INT NOT NULL DEFAULT 0
);
