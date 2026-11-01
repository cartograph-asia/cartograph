#!/bin/bash
# Nightly backup on the production host (docs/DEPLOY_BRIEF.md, section 7),
# run from the deploy user's crontab:
#
#   30 3 * * * $HOME/cartograph/deploy/backup.sh >> $HOME/.local/state/cartograph-backup.log 2>&1
#
# It writes into $BACKUP_DIR (default ~/cartograph-backups), which Adrian's Mac
# pulls with deploy/pull-backups.sh; the server holds no credentials for the
# Mac.
#
#   db/cartograph-YYYYMMDD.dump.gpg  pg_dump -Fc of the whole database,
#                                    encrypted to the public key in
#                                    $BACKUP_DIR/backup-key.asc (its private
#                                    half lives only on the Mac); the last 3
#                                    nights are kept here
#   photos/                          a mirror of the photo bucket, without
#                                    deleting: a photo removed on the site
#                                    stays in the backup
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd)
dest=${BACKUP_DIR:-$HOME/cartograph-backups}
key="$dest/backup-key.asc"
mkdir -p "$dest/db" "$dest/photos"
[ -f "$key" ] || { echo "$(date -Is) ERROR: $key is missing (the Mac's public backup key)"; exit 1; }

day=$(date +%Y%m%d)
out="$dest/db/cartograph-$day.dump.gpg"
"$here/prod.sh" exec -T postgres pg_dump -U cartograph -d cartograph -Fc |
	gpg --batch --yes --trust-model always --recipient-file "$key" --encrypt --output "$out.tmp"
mv "$out.tmp" "$out"
find "$dest/db" -name 'cartograph-*.dump.gpg' -mtime +2 -delete

# The bucket through Garage's S3 API inside the stack's network, with the
# api's key from .env; nothing is published on the host for this. .env is
# read value by value: it is not shell syntax (MAIL_FROM has "<" in it).
env_val() { sed -n "s/^$1=//p" "$here/.env" | tail -n 1; }
S3_ACCESS_KEY=$(env_val S3_ACCESS_KEY)
S3_SECRET_KEY=$(env_val S3_SECRET_KEY)
S3_BUCKET=$(env_val S3_BUCKET)
docker run --rm --user "$(id -u):$(id -g)" --network "${CARTOGRAPH_PROJECT:-cartograph}_default" \
	-v "$dest/photos:/data" \
	-e RCLONE_CONFIG_G_TYPE=s3 -e RCLONE_CONFIG_G_PROVIDER=Other \
	-e RCLONE_CONFIG_G_ENDPOINT=http://garage:3900 -e RCLONE_CONFIG_G_REGION=us-east-1 \
	-e RCLONE_CONFIG_G_ACCESS_KEY_ID="$S3_ACCESS_KEY" -e RCLONE_CONFIG_G_SECRET_ACCESS_KEY="$S3_SECRET_KEY" \
	-e RCLONE_CONFIG_G_FORCE_PATH_STYLE=true \
	rclone/rclone:1.71.2 copy "g:${S3_BUCKET:-cartograph}" /data --quiet

echo "$(date -Is) backup ok: $(du -sh "$out" | cut -f1) dump, $(du -sh "$dest/photos" | cut -f1) photos"
