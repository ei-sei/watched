ALTER TABLE rewatches
  ADD CONSTRAINT rewatches_user_media_date_unique UNIQUE (user_id, media_id, finished_at);
