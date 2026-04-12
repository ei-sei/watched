-- Remove any duplicate (user_id, media_id, finished_at) rows before adding the constraint,
-- keeping the earliest entry per group.
DELETE FROM rewatches
WHERE id NOT IN (
    SELECT MIN(id)
    FROM rewatches
    GROUP BY user_id, media_id, finished_at
);

ALTER TABLE rewatches
  ADD CONSTRAINT rewatches_user_media_date_unique UNIQUE (user_id, media_id, finished_at);
