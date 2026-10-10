CREATE TABLE IF NOT EXISTS diary_freetext (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    content TEXT NOT NULL,
    fields JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_diary_freetext_user_created ON diary_freetext(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS diary_mood (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    content TEXT NOT NULL,
    fields JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_diary_mood_user_created ON diary_mood(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS diary_emotions (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    content TEXT NOT NULL,
    fields JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_diary_emotions_user_created ON diary_emotions(user_id, created_at DESC);

INSERT INTO diary_freetext (user_id, content, fields, created_at, updated_at)
SELECT user_id, content,
       CASE WHEN fields IS NULL OR fields = '[]'::jsonb
            THEN jsonb_build_array(jsonb_build_object('name', 'text', 'value', content))
            ELSE fields END,
       created_at, updated_at
FROM entries
WHERE plugin_slug = 'freetext';

INSERT INTO diary_mood (user_id, content, fields, created_at, updated_at)
SELECT user_id, content, fields, created_at, updated_at
FROM entries
WHERE plugin_slug = 'mood';

INSERT INTO diary_emotions (user_id, content, fields, created_at, updated_at)
SELECT user_id, content, fields, created_at, updated_at
FROM entries
WHERE plugin_slug = 'emotions';

DROP TABLE IF EXISTS entries;

DELETE FROM user_plugins WHERE plugin_slug = 'freetext';
