CREATE TABLE IF NOT EXISTS entries (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    content TEXT NOT NULL,
    plugin_slug VARCHAR(64) NOT NULL DEFAULT 'freetext',
    fields JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_entries_user_created ON entries(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_entries_user_plugin_created ON entries(user_id, plugin_slug, created_at DESC);

INSERT INTO entries (user_id, content, plugin_slug, fields, created_at, updated_at)
SELECT user_id, content, 'freetext', fields, created_at, updated_at FROM diary_freetext;

INSERT INTO entries (user_id, content, plugin_slug, fields, created_at, updated_at)
SELECT user_id, content, 'mood', fields, created_at, updated_at FROM diary_mood;

INSERT INTO entries (user_id, content, plugin_slug, fields, created_at, updated_at)
SELECT user_id, content, 'emotions', fields, created_at, updated_at FROM diary_emotions;

DROP TABLE IF EXISTS diary_freetext;
DROP TABLE IF EXISTS diary_mood;
DROP TABLE IF EXISTS diary_emotions;
