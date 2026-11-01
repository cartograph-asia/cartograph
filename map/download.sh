#!/usr/bin/env bash
# Download the Thailand OSM extract from Geofabrik into map/data/.
# Idempotent: skips the download if the file is already present (delete it
# to fetch a fresh extract). Works from any working directory.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_DIR="${SCRIPT_DIR}/data"
PBF_URL="${PBF_URL:-https://download.geofabrik.de/asia/thailand-latest.osm.pbf}"
PBF_FILE="${DATA_DIR}/$(basename "${PBF_URL}")"

mkdir -p "${DATA_DIR}"

if [[ -s "${PBF_FILE}" ]]; then
  echo "download: ${PBF_FILE} already present, skipping (delete it to re-download)"
  exit 0
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
