#!/usr/bin/env bash
# Download OSM extracts from Geofabrik (asia/<name>-latest.osm.pbf) into
# map/data/: the names given as arguments, else every extract the base map
# is built from (EXTRACTS, see build.sh). Idempotent: an extract already
# present is skipped (delete it to fetch a fresh one). Works from any
# working directory.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_DIR="${SCRIPT_DIR}/data"
# Thailand, where Cartograph has plots, and its neighbours, so that the base
# map does not end at the border. cmd/regions, cmd/places and cmd/fixture
# still read thailand-latest.osm.pbf alone.
EXTRACTS="${EXTRACTS:-thailand myanmar laos cambodia vietnam malaysia-singapore-brunei}"
if [[ $# -gt 0 ]]; then
  names=("$@")
else
  read -r -a names <<<"${EXTRACTS}"
fi

# download fetches asia/<name>-latest.osm.pbf unless it is already there.
download() {
  local PBF_URL PBF_FILE md5_line expected actual
  PBF_URL="https://download.geofabrik.de/asia/$1-latest.osm.pbf"
  PBF_FILE="${DATA_DIR}/$(basename "${PBF_URL}")"

  if [[ -s "${PBF_FILE}" ]]; then
    echo "download: ${PBF_FILE} already present, skipping (delete it to re-download)"
    return 0
  fi

  echo "download: fetching ${PBF_URL}"
  curl -fL --retry 3 --retry-delay 5 -o "${PBF_FILE}.part" "${PBF_URL}"

  # Geofabrik publishes an .md5 next to every extract; verify when it answers
  # in time (the .md5 endpoint is sometimes slow, so this is best effort).
  if md5_line="$(curl -fsL --max-time 30 "${PBF_URL}.md5" 2>/dev/null)" && [[ -n "${md5_line}" ]]; then
    expected="${md5_line%% *}"
    if command -v md5sum >/dev/null 2>&1; then
      actual="$(md5sum "${PBF_FILE}.part" | cut -d' ' -f1)"
    else
      actual="$(md5 -q "${PBF_FILE}.part")"
    fi
    if [[ "${expected}" != "${actual}" ]]; then
      echo "download: checksum mismatch (expected ${expected}, got ${actual})" >&2
      rm -f "${PBF_FILE}.part"
      exit 1
    fi
    echo "download: md5 ok (${actual})"
  else
    echo "download: could not fetch ${PBF_URL}.md5, skipping checksum" >&2
  fi

  mv "${PBF_FILE}.part" "${PBF_FILE}"
  echo "download: saved ${PBF_FILE}"
}

mkdir -p "${DATA_DIR}"
for name in "${names[@]}"; do
  download "${name}"
done
