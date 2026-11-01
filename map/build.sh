#!/usr/bin/env bash
# Build map/asia.pmtiles from Geofabrik extracts with Planetiler
# (OpenMapTiles-compatible default profile, zoom 0-14), via Docker.
#
# The extracts (EXTRACTS: Thailand and its neighbours, see download.sh) are
# merged into one file with osmium, run from Debian's package in a
# throwaway debian container, so nothing is installed on the host. The
# merge is redone only when an extract is newer than the merged file.
#
# Auxiliary sources (water polygons, Natural Earth, lake centerlines) are
# downloaded by Planetiler on first run into map/data/sources/ and reused
# afterwards (--download only fetches files that are missing; the existing
# OSM extract is never re-downloaded). Re-running rebuilds the archive.
# Planetiler wipes --tmpdir, so it points at a directory used for nothing else.
#
# Env overrides:
#   PLANETILER_IMAGE  Docker image (pinned release)
#   JAVA_XMX          JVM heap, default 4g (Thailand peaks at ~2.1 GB heap,
#                     ~3.5 GB container memory incl. mmap; see README.md)
#   BOUNDS            area the archive covers, default 88,-2,114,28: the map's
#                     maxBounds in web/src/main.ts. Planetiler would otherwise
#                     stop at the extract's bounding box, so the sea west of
#                     97.3°E and everything beyond Thailand's box went blank
#                     ("cut off"). Ocean and low-zoom land come from the
#                     global water polygons and Natural Earth.
#   EXTRACTS          Geofabrik asia/ extracts to merge, default Thailand
#                     and its neighbours (see download.sh)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLANETILER_IMAGE="${PLANETILER_IMAGE:-ghcr.io/onthegomap/planetiler:0.10.2}"
JAVA_XMX="${JAVA_XMX:-5g}"
BOUNDS="${BOUNDS:-88,-2,114,28}"
DEBIAN_IMAGE="${DEBIAN_IMAGE:-debian:bookworm-slim}"
EXTRACTS="${EXTRACTS:-thailand myanmar laos cambodia vietnam malaysia-singapore-brunei}"
PBF_NAME="region-latest.osm.pbf"
OUTPUT_NAME="asia.pmtiles"
export EXTRACTS

"${SCRIPT_DIR}/download.sh"

read -r -a names <<<"${EXTRACTS}"
parts=()
stale=0
for name in "${names[@]}"; do
  parts+=("/map/data/${name}-latest.osm.pbf")
  [[ "${SCRIPT_DIR}/data/${name}-latest.osm.pbf" -nt "${SCRIPT_DIR}/data/${PBF_NAME}" ]] && stale=1
done
if [[ ! -s "${SCRIPT_DIR}/data/${PBF_NAME}" || ${stale} == 1 ]]; then
  echo "merge: ${EXTRACTS} -> data/${PBF_NAME}"
  # apt needs root inside the container; the merged file is handed back to
  # the host user and moved in place only when complete.
  docker run --rm \
    -e HOST_UID="$(id -u)" -e HOST_GID="$(id -g)" \
    -v "${SCRIPT_DIR}:/map" \
    "${DEBIAN_IMAGE}" \
    sh -euc 'apt-get update -qq >/dev/null
      apt-get install -y -qq --no-install-recommends osmium-tool >/dev/null
      # Extracts of different days hold different versions of the objects
      # along the borders, and Planetiler stops at the duplicate IDs.
      stamps=$(for f in "$@"; do osmium fileinfo -g header.option.osmosis_replication_timestamp "$f"; done | sort -u)
      if [ "$(echo "$stamps" | wc -l)" -ne 1 ]; then
        echo "merge: the extracts are from different days:" >&2
        for f in "$@"; do echo "  $(basename "$f") $(osmium fileinfo -g header.option.osmosis_replication_timestamp "$f")" >&2; done
        echo "merge: delete the older ones in map/data/ and run build.sh again" >&2
        exit 3
      fi
      osmium merge --overwrite -o /map/data/'"${PBF_NAME}"'.part -f pbf "$@"
      chown "$HOST_UID:$HOST_GID" /map/data/'"${PBF_NAME}"'.part' \
    merge "${parts[@]}"
  mv "${SCRIPT_DIR}/data/${PBF_NAME}.part" "${SCRIPT_DIR}/data/${PBF_NAME}"
fi

mkdir -p "${SCRIPT_DIR}/data/sources" "${SCRIPT_DIR}/data/tmp"

echo "build: ${PLANETILER_IMAGE}, -Xmx${JAVA_XMX}, bounds ${BOUNDS}"
start=$(date +%s)

# map/ is mounted at /map; every path is explicit so nothing depends on the
# image's working directory.
docker run --rm \
  --user "$(id -u):$(id -g)" \
  -e JAVA_TOOL_OPTIONS="-Xmx${JAVA_XMX}" \
  -v "${SCRIPT_DIR}:/map" \
  "${PLANETILER_IMAGE}" \
  --download \
  --osm-path="/map/data/${PBF_NAME}" \
  --water-polygons-path=/map/data/sources/water-polygons-split-3857.zip \
  --natural-earth-path=/map/data/sources/natural_earth_vector.sqlite.zip \
  --lake-centerlines-path=/map/data/sources/lake_centerline.shp.zip \
  --tile-weights=/map/data/sources/tile_weights.tsv.gz \
  --download-dir=/map/data/sources \
  --tmpdir=/map/data/tmp \
  --minzoom=0 \
  --maxzoom=14 \
  --bounds="${BOUNDS}" \
  --output="/map/data/${OUTPUT_NAME}" \
  --layer-stats=/map/data/region.layerstats.tsv.gz \
  --force

rm -rf "${SCRIPT_DIR}/data/tmp"
# Built beside the served file and moved in place in one step, so Caddy never
# serves a half-written archive.
mv "${SCRIPT_DIR}/data/${OUTPUT_NAME}" "${SCRIPT_DIR}/${OUTPUT_NAME}"
echo "build: wrote ${SCRIPT_DIR}/${OUTPUT_NAME} in $(( $(date +%s) - start ))s"
