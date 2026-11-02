#!/bin/sh
# Docker Compose for the production stack on the host, with the right files
# and the image tags of the running release (docs/DEPLOY.md):
#
#   deploy/prod.sh ps
#   deploy/prod.sh logs -f api
#   deploy/prod.sh --profile tools run --rm regions
#
# deploy/release holds API_TAG and WEB_TAG, written by deploy/post-receive
# when it switches to a new release; without it the images tagged "local"
# are used.
set -eu
dir=$(cd "$(dirname "$0")" && pwd)
if [ -f "$dir/release" ]; then
	set -a
	. "$dir/release"
	set +a
fi
exec docker compose --project-name "${CARTOGRAPH_PROJECT:-cartograph}" --project-directory "$dir" \
	-f "$dir/docker-compose.yml" -f "$dir/compose.prod.yaml" "$@"
