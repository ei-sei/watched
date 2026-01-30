CREATE TABLE rewatches (
    id          SERIAL PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    media_id    INTEGER NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    started_at  DATE,
    finished_at DATE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rewatches_user_media ON rewatches(user_id, media_id);
