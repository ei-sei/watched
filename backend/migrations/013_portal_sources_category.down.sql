ALTER TABLE portal_links DROP CONSTRAINT portal_links_category_check;
ALTER TABLE portal_links ADD CONSTRAINT portal_links_category_check CHECK (category IN ('movies_tv', 'anime'));
