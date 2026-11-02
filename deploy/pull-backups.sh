#!/bin/bash
# Takes a backup of production and pulls it to Adrian's Mac, on demand:
#
#   cartograph-pull-backups        (installed by deploy/install-backup-mac.sh)
#
# The Mac asks for it; the host runs nothing on its own and holds no
# credentials for the Mac. Over ssh this runs deploy/backup.sh on the host,
# which writes a fresh encrypted dump and updates the photo mirror there, and
# then copies both here.
#
# Settings come from ~/.config/cartograph/backup.env (not in git):
#
#   BACKUP_HOST=myhost                        ssh host (as in ~/.ssh/config)
#   BACKUP_REMOTE=cartograph-backups          directory on the host
#   BACKUP_LOCAL=/Volumes/…/cartograph        where the backups go on the Mac
#   BACKUP_RUNTIME=/srv/cartograph            the stack on the host (default)
#
# Every dump pulled is kept here (a few MB each); delete old ones by hand.
# Photos are mirrored without deleting.
set -euo pipefail
conf=${CARTOGRAPH_BACKUP_CONF:-$HOME/.config/cartograph/backup.env}
# shellcheck disable=SC1090
. "$conf"
: "${BACKUP_HOST:?}" "${BACKUP_REMOTE:?}" "${BACKUP_LOCAL:?}"
runtime=${BACKUP_RUNTIME:-/srv/cartograph}

if [ ! -d "$(dirname "$BACKUP_LOCAL")" ]; then
	echo "The backup disk is not mounted ($(dirname "$BACKUP_LOCAL"))." >&2
	exit 1
fi
mkdir -p "$BACKUP_LOCAL/db" "$BACKUP_LOCAL/photos"

ssh -o BatchMode=yes "$BACKUP_HOST" "$runtime/deploy/backup.sh"

# --ignore-existing: a dump never changes once written; photos keep their keys.
rsync -a --ignore-existing -e "ssh -o BatchMode=yes" "$BACKUP_HOST:$BACKUP_REMOTE/db/" "$BACKUP_LOCAL/db/"
rsync -a --no-links --ignore-existing -e "ssh -o BatchMode=yes" "$BACKUP_HOST:$BACKUP_REMOTE/photos/" "$BACKUP_LOCAL/photos/"

echo "$(date '+%F %T') ok: $(find "$BACKUP_LOCAL/db" -name '*.gpg' | wc -l | tr -d ' ') dumps, $(du -sh "$BACKUP_LOCAL/photos" | cut -f1) photos in $BACKUP_LOCAL"
