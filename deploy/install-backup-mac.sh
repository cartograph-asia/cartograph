#!/bin/bash
# Sets up the pull of the production backups on Adrian's Mac (once):
#
#   deploy/install-backup-mac.sh /Volumes/<disk>/backups/cartograph SSH_HOST
#
#   1. a GPG key only for the backups, in its own keyring
#      (~/.config/cartograph/gnupg); its passphrase goes into the login
#      keychain (service "cartograph-backup"), the public half to the host as
#      cartograph-backups/backup-key.asc for deploy/backup.sh
#   2. ~/.config/cartograph/backup.env with the paths
#   3. the pull script as ~/.local/bin/cartograph-pull-backups and the launchd
#      job asia.cartograph.backup (daily at 07:30, or on wake if missed)
#
# Run again, it keeps an existing key and only refreshes the rest.
set -euo pipefail
local_dir=${1:?usage: install-backup-mac.sh BACKUP_DIR SSH_HOST}
host=${2:?usage: install-backup-mac.sh BACKUP_DIR SSH_HOST}
here=$(cd "$(dirname "$0")" && pwd)
conf=$HOME/.config/cartograph
export GNUPGHOME=$conf/gnupg
mkdir -p "$conf" "$GNUPGHOME" "$HOME/.local/bin" "$HOME/Library/LaunchAgents" "$HOME/Library/Logs"
chmod 700 "$GNUPGHOME"

uid="Cartograph backup <backup@cartograph.asia>"
if ! gpg --list-secret-keys "$uid" >/dev/null 2>&1; then
	pass=$(openssl rand -base64 32)
	# Through security's stdin, not its arguments: ps shows those to every
	# local user. The base64 passphrase has no spaces or quotes.
	printf 'add-generic-password -U -a %s -s cartograph-backup -w %s\n' "$USER" "$pass" | security -i
	printf '%s' "$pass" | gpg --batch --pinentry-mode loopback --passphrase-fd 0 \
		--quick-generate-key "$uid" cv25519 encr never
	echo "created the backup key; its passphrase is in the login keychain (cartograph-backup)"
fi
gpg --armor --export "$uid" > "$conf/backup-key.asc"
ssh -o BatchMode=yes "$host" 'mkdir -p cartograph-backups/db cartograph-backups/photos'
scp -q "$conf/backup-key.asc" "$host:cartograph-backups/backup-key.asc"

# Quoted: pull-backups.sh sources this file, and disk names have spaces.
{
	printf 'BACKUP_HOST=%q\n' "$host"
	printf 'BACKUP_REMOTE=cartograph-backups\n'
	printf 'BACKUP_LOCAL=%q\n' "$local_dir"
} > "$conf/backup.env"
install -m 755 "$here/pull-backups.sh" "$HOME/.local/bin/cartograph-pull-backups"

plist=$HOME/Library/LaunchAgents/asia.cartograph.backup.plist
sed -e "s|@HOME@|$HOME|g" "$here/asia.cartograph.backup.plist" > "$plist"
launchctl bootout "gui/$(id -u)" "$plist" 2>/dev/null || true
launchctl bootstrap "gui/$(id -u)" "$plist"
echo "installed: launchd job asia.cartograph.backup, log in ~/Library/Logs/cartograph-backup.log"
