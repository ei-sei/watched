CREATE TABLE portal_links (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(200) NOT NULL,
    url         VARCHAR(1000) NOT NULL,
    category    VARCHAR(20) NOT NULL CHECK (category IN ('movies_tv', 'anime')),
    created_by  INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
