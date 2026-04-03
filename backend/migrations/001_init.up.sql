CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id              SERIAL PRIMARY KEY,
    username        VARCHAR(50)  UNIQUE NOT NULL,
    email           VARCHAR(255) UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    display_name    VARCHAR(100),
    avatar_url      VARCHAR(500),
    is_admin        BOOLEAN NOT NULL DEFAULT FALSE,
    is_premium      BOOLEAN NOT NULL DEFAULT FALSE,
    failed_attempts INT NOT NULL DEFAULT 0,
    locked_until    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TYPE media_type AS ENUM ('film', 'tv_show', 'book');
CREATE TYPE media_status AS ENUM ('want_to', 'in_progress', 'completed', 'dropped', 'on_hold');

CREATE TABLE media_items (
    id           SERIAL PRIMARY KEY,
    user_id      INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    media_type   media_type NOT NULL,
    external_id  VARCHAR(100),
    title        VARCHAR(500) NOT NULL,
    year         INT,
    poster_url   VARCHAR(1000),
    metadata     JSONB NOT NULL DEFAULT '{}',
    status       media_status NOT NULL DEFAULT 'want_to',
    rating       FLOAT,
    review_text  TEXT,
    started_at   DATE,
    completed_at DATE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, external_id)
);
CREATE INDEX ON media_items (user_id);
CREATE INDEX ON media_items (user_id, media_type);
CREATE INDEX ON media_items (user_id, status);

CREATE TABLE tv_episode_logs (
    id             SERIAL PRIMARY KEY,
    media_item_id  INT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    season_number  INT NOT NULL,
    episode_number INT NOT NULL,
    watched_at     TIMESTAMPTZ,
    rating         FLOAT,
    note           TEXT,
    UNIQUE (media_item_id, season_number, episode_number)
);
CREATE INDEX ON tv_episode_logs (media_item_id);

CREATE TYPE chapter_status AS ENUM ('unread', 'in_progress', 'completed');

CREATE TABLE book_chapter_logs (
    id             SERIAL PRIMARY KEY,
    media_item_id  INT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    chapter_number INT NOT NULL,
    chapter_title  VARCHAR(300),
    start_page     INT,
    end_page       INT,
    status         chapter_status NOT NULL DEFAULT 'unread',
    note           TEXT,
    started_at     TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ,
    UNIQUE (media_item_id, chapter_number)
);
CREATE INDEX ON book_chapter_logs (media_item_id);

CREATE TABLE user_lists (
    id          SERIAL PRIMARY KEY,
    user_id     INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(200) NOT NULL,
    description TEXT,
    is_public   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON user_lists (user_id);

CREATE TABLE list_items (
    id            SERIAL PRIMARY KEY,
    list_id       INT NOT NULL REFERENCES user_lists(id) ON DELETE CASCADE,
    media_item_id INT NOT NULL REFERENCES media_items(id) ON DELETE CASCADE,
    position      INT NOT NULL DEFAULT 0,
    added_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON list_items (list_id);

CREATE TYPE action_type AS ENUM (
    'added', 'rated', 'reviewed', 'started', 'completed',
    'chapter_read', 'episode_watched'
);

CREATE TABLE activity_logs (
    id            SERIAL PRIMARY KEY,
    user_id       INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action        action_type NOT NULL,
    media_item_id INT REFERENCES media_items(id) ON DELETE SET NULL,
    details       JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON activity_logs (user_id, created_at DESC);

CREATE TABLE media_cache (
    id          SERIAL PRIMARY KEY,
    external_id VARCHAR(100) UNIQUE NOT NULL,
    source      VARCHAR(20) NOT NULL,
    media_type  VARCHAR(20) NOT NULL,
    title       VARCHAR(500) NOT NULL,
    metadata    JSONB,
    cached_at   TIMESTAMPTZ NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX ON media_cache (expires_at);

CREATE TABLE invite_codes (
    code       VARCHAR(32) PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    used_at    TIMESTAMPTZ
);
