#!/usr/bin/env bash
# Backfills missing poster URLs for imported anime using Jikan API
# Usage: ./scripts/backfill_posters.sh

set -e

DB_URL="${DATABASE_URL:-postgres://brsti:localdevpassword@localhost/brsti_db?sslmode=disable}"

# Get all anime missing posters
IDS=$(psql "$DB_URL" -t -c "SELECT id || ':' || metadata->>'mal_id' FROM media_items WHERE media_type='anime' AND poster_url IS NULL AND metadata->>'mal_id' IS NOT NULL;")

COUNT=0
for row in $IDS; do
  item_id="${row%%:*}"
  mal_id="${row##*:}"

  # Fetch from Jikan
  response=$(curl -s "https://api.jikan.moe/v4/anime/${mal_id}" 2>/dev/null)
  poster=$(echo "$response" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data']['images']['jpg'].get('large_image_url',''))" 2>/dev/null)

  if [ -n "$poster" ] && [ "$poster" != "None" ]; then
    psql "$DB_URL" -c "UPDATE media_items SET poster_url = '$poster' WHERE id = $item_id;" > /dev/null
    echo "[$((COUNT+1))] Updated: item $item_id (MAL $mal_id)"
    COUNT=$((COUNT+1))
  else
    echo "[ ] No poster for item $item_id (MAL $mal_id)"
  fi

  # Jikan rate limit: ~3 req/s, use 400ms to be safe
  sleep 0.4
done

echo "Done. Updated $COUNT posters."
