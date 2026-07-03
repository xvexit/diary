CREATE TABLE IF NOT EXISTS user_settings (
    user_id BIGINT PRIMARY KEY,
    reminder_enabled BOOLEAN NOT NULL DEFAULT true,
    reminder_time TIME NOT NULL DEFAULT '21:00'
);

INSERT INTO user_settings (user_id, reminder_enabled, reminder_time)
SELECT DISTINCT user_id, true, '21:00'::TIME FROM entries
ON CONFLICT (user_id) DO NOTHING;
