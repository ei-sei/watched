ALTER TABLE portal_links ADD COLUMN position INT NOT NULL DEFAULT 0;
UPDATE portal_links SET position = id;
