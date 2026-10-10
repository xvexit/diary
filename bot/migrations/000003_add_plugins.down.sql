DROP TABLE IF EXISTS user_plugins;

DROP INDEX IF EXISTS idx_entries_user_plugin_created;

ALTER TABLE entries
    DROP COLUMN IF EXISTS fields,
    DROP COLUMN IF EXISTS plugin_slug;
