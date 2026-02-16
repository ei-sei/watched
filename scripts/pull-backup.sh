#!/bin/bash
set -euo pipefail

VPS_USER="${VPS_USER:-ubuntu}"
VPS_HOST="${VPS_HOST:?Set VPS_HOST environment variable}"
VPS_BACKUP_DIR="${VPS_BACKUP_DIR:-~/backups}"
LOCAL_BACKUP_DIR="${LOCAL_BACKUP_DIR:-$HOME/backups/watched}"
KEEP_DAYS="${KEEP_DAYS:-30}"

DATE=$(date +%Y%m%d)
FILENAME="watched-${DATE}.sql.gz"

mkdir -p "$LOCAL_BACKUP_DIR"

echo "[$(date)] Pulling $FILENAME from $VPS_HOST..."
scp "${VPS_USER}@${VPS_HOST}:${VPS_BACKUP_DIR}/${FILENAME}" "${LOCAL_BACKUP_DIR}/${FILENAME}"
echo "[$(date)] Saved to ${LOCAL_BACKUP_DIR}/${FILENAME}"

# Remove local copies older than KEEP_DAYS
find "$LOCAL_BACKUP_DIR" -name "watched-*.sql.gz" -mtime "+${KEEP_DAYS}" -delete
echo "[$(date)] Cleaned up backups older than ${KEEP_DAYS} days"
