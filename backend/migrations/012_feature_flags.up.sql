CREATE TABLE feature_flags (
    key        VARCHAR(50) PRIMARY KEY,
    is_premium BOOLEAN     NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO feature_flags (key) VALUES ('stats'), ('trending'), ('portal');
