#!/usr/bin/env python3
"""Backfills missing anime poster URLs using Jikan API."""
import subprocess, json, time, sys, urllib.request

def psql(sql):
    result = subprocess.run(
        ['docker', 'compose', '-f',
         '/home/ei-sei/Documents/CodingProjects/Watched/docker-compose.yml',
         'exec', '-T', 'db', 'psql', '-U', 'brsti', '-d', 'brsti_db', '-t', '-c', sql],
        capture_output=True, text=True
    )
    return result.stdout.strip()

# Get all anime missing posters
rows = psql("SELECT id::text || ':' || (metadata->>'mal_id') FROM media_items WHERE media_type='anime' AND poster_url IS NULL AND (metadata->>'mal_id') IS NOT NULL;")
items = [r.strip() for r in rows.splitlines() if ':' in r.strip()]

print(f"Found {len(items)} anime missing posters")
updated = 0

for row in items:
    item_id, mal_id = row.split(':', 1)
    item_id = item_id.strip()
    mal_id = mal_id.strip()
    if not mal_id:
        continue

    try:
        req = urllib.request.Request(
            f"https://api.jikan.moe/v4/anime/{mal_id}",
            headers={"User-Agent": "watched-backfill/1.0"}
        )
        with urllib.request.urlopen(req, timeout=10) as resp:
            data = json.loads(resp.read())
        poster = data['data']['images']['jpg'].get('large_image_url')
    except Exception as e:
        print(f"  Skip MAL {mal_id}: {e}")
        time.sleep(0.4)
        continue

    if poster:
        safe_poster = poster.replace("'", "''")
        psql(f"UPDATE media_items SET poster_url = '{safe_poster}' WHERE id = {item_id};")
        print(f"  Updated item {item_id} (MAL {mal_id})")
        updated += 1
    else:
        print(f"  No poster for MAL {mal_id}")

    time.sleep(0.4)

print(f"\nDone. Updated {updated}/{len(items)} posters.")
