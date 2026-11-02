#!/bin/bash
# Pulls the production backups to Adrian's Mac (docs/DEPLOY_BRIEF.md,
# section 7). Run daily by launchd (deploy/asia.cartograph.backup.plist); a run
# missed while the Mac slept happens when it wakes.
#
# Settings come from ~/.config/cartograph/backup.env (not in git):
#
#   BACKUP_HOST=myhost                        ssh host (as in ~/.ssh/config)
#   BACKUP_REMOTE=cartograph-backups          directory on the host
#   BACKUP_LOCAL=/Volumes/…/cartograph        where the backups go on the Mac
#
# The database dumps arrive encrypted (deploy/backup.sh); they are kept 14
# days, then only Sundays for 8 weeks, then only the first of each month for
# 12 months. Photos are mirrored without deleting. When the newest dump on
# the Mac is older than two days, a notification says so.
set -euo pipefail
conf=${CARTOGRAPH_BACKUP_CONF:-$HOME/.config/cartograph/backup.env}
# shellcheck disable=SC1090
. "$conf"
: "${BACKUP_HOST:?}" "${BACKUP_REMOTE:?}" "${BACKUP_LOCAL:?}"

notify() {
	osascript -e "display notification \"$1\" with title \"Cartograph backup\"" >/dev/null 2>&1 || true
	echo "$(date '+%F %T') $1"
}

if [ ! -d "$(dirname "$BACKUP_LOCAL")" ]; then
	notify "The backup disk is not mounted ($(dirname "$BACKUP_LOCAL"))."
	exit 1
fi
mkdir -p "$BACKUP_LOCAL/db" "$BACKUP_LOCAL/photos"

# --ignore-existing: a dump never changes once written; photos keep their keys.
rsync -a --ignore-existing -e "ssh -o BatchMode=yes" "$BACKUP_HOST:$BACKUP_REMOTE/db/" "$BACKUP_LOCAL/db/"
rsync -a --no-links --ignore-existing -e "ssh -o BatchMode=yes" "$BACKUP_HOST:$BACKUP_REMOTE/photos/" "$BACKUP_LOCAL/photos/"

# Keep 14 daily, 8 weekly (Sundays), 12 monthly (the 1st) dumps.
now=$(date +%s)
for f in "$BACKUP_LOCAL"/db/cartograph-*.dump.gpg; do
	[ -e "$f" ] || continue
	d=$(basename "$f" | sed -E 's/cartograph-([0-9]{8})\.dump\.gpg/\1/')
	t=$(date -j -f %Y%m%d "$d" +%s 2>/dev/null) || continue
	age=$(((now - t) / 86400))
	dow=$(date -j -f %Y%m%d "$d" +%u)
	dom=$(date -j -f %Y%m%d "$d" +%d)
	if [ "$age" -le 14 ] || { [ "$dow" = 7 ] && [ "$age" -le 56 ]; } || { [ "$dom" = 01 ] && [ "$age" -le 366 ]; }; then
		continue
	fi
	rm -f "$f"
done

newest=$(find "$BACKUP_LOCAL/db" -name 'cartograph-*.dump.gpg' -mtime -2 | head -n 1)
if [ -z "$newest" ]; then
	notify "The newest database backup is older than two days."
	exit 1
fi
echo "$(date '+%F %T') ok: $(find "$BACKUP_LOCAL/db" -name '*.gpg' | wc -l | tr -d ' ') dumps, $(du -sh "$BACKUP_LOCAL/photos" | cut -f1) photos"
