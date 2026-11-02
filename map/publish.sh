#!/bin/sh
# Copies the base map to the production host under a dated name, so that
# browsers never mix byte ranges of two builds (docs/DEPLOY.md):
#
#   map/publish.sh HOST RUNTIME_DIR [YYYY-MM-DD]
#
# e.g. map/publish.sh myhost /srv/cartograph. It resumes an interrupted copy
# and prints the TILES_URL line to put into the host's deploy/.env; after
# switching (deploy/prod.sh up -d api), delete the previous archive on the
# host right away, as the disk holds two only for the moment of the switch.
set -eu
host=${1:?usage: map/publish.sh HOST RUNTIME_DIR [YYYY-MM-DD]}
runtime=${2:?usage: map/publish.sh HOST RUNTIME_DIR [YYYY-MM-DD]}
day=${3:-$(date -r "$(dirname "$0")/asia.pmtiles" +%Y-%m-%d)}
src="$(dirname "$0")/asia.pmtiles"
name="asia-$day.pmtiles"

ssh "$host" "mkdir -p '$runtime/map'"
rsync -a --partial "$src" "$host:$runtime/map/$name"
echo "copied. Put this into $runtime/deploy/.env on $host, then run deploy/prod.sh up -d api:"
echo "TILES_URL=/tiles/$name"
