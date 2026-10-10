ALTER TABLE entries
    ADD COLUMN IF NOT EXISTS plugin_slug VARCHAR(64) NOT NULL DEFAULT 'freetext',
    ADD COLUMN IF NOT EXISTS fields JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE INDEX IF NOT EXISTS idx_entries_user_plugin_created
    ON entries(user_id, plugin_slug, created_at DESC);

CREATE TABLE IF NOT EXISTS user_plugins (
    user_id BIGINT NOT NULL,
    plugin_slug VARCHAR(64) NOT NULL,
    installed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, plugin_slug)
);

INSERT INTO user_plugins (user_id, plugin_slug)
SELECT DISTINCT user_id, 'freetext' FROM entries
ON CONFLICT (user_id, plugin_slug) DO NOTHING;
