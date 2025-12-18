#!/bin/bash
set -euo pipefail

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/tmp/brsti-backups"
BACKUP_FILE="brsti_db_${TIMESTAMP}.sql.gz"
S3_BUCKET="${S3_BUCKET:-brsti-backups}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"

mkdir -p "$BACKUP_DIR"

echo "[$(date)] Starting backup..."

docker exec brsti-db pg_dump -U "${POSTGRES_USER}" "${POSTGRES_DB}" | \
  gzip > "${BACKUP_DIR}/${BACKUP_FILE}"

aws s3 cp "${BACKUP_DIR}/${BACKUP_FILE}" "s3://${S3_BUCKET}/daily/${BACKUP_FILE}"

rm -f "${BACKUP_DIR}/${BACKUP_FILE}"

aws s3 ls "s3://${S3_BUCKET}/daily/" | awk '{print $4}' | while read -r file; do
  file_date=$(echo "$file" | grep -oP '\d{8}' | head -1)
  cutoff_date=$(date -d "-${RETENTION_DAYS} days" +%Y%m%d)
  if [[ -n "$file_date" && "$file_date" < "$cutoff_date" ]]; then
    aws s3 rm "s3://${S3_BUCKET}/daily/${file}"
    echo "[$(date)] Removed old backup: ${file}"
  fi
done

echo "[$(date)] Backup completed: ${BACKUP_FILE}"
