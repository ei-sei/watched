-- Backfill completed_at for items that were marked completed before the field was auto-set
UPDATE media_items
SET completed_at = updated_at
WHERE status = 'completed'
  AND completed_at IS NULL;

-- Backfill started_at for items that are in progress or completed but missing started_at
UPDATE media_items
SET started_at = created_at
WHERE status IN ('in_progress', 'completed', 'dropped', 'on_hold')
  AND started_at IS NULL;
