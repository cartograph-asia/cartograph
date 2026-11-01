#!/usr/bin/env bash
# Build map/thailand.pmtiles from the Geofabrik extract with Planetiler
# (OpenMapTiles-compatible default profile, zoom 0-14), via Docker.
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
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLANETILER_IMAGE="${PLANETILER_IMAGE:-ghcr.io/onthegomap/planetiler:0.10.2}"
JAVA_XMX="${JAVA_XMX:-4g}"
PBF_NAME="thailand-latest.osm.pbf"
OUTPUT_NAME="thailand.pmtiles"

if [[ ! -s "${SCRIPT_DIR}/data/${PBF_NAME}" ]]; then
  "${SCRIPT_DIR}/download.sh"
fi

mkdir -p "${SCRIPT_DIR}/data/sources" "${SCRIPT_DIR}/data/tmp"

echo "build: ${PLANETILER_IMAGE}, -Xmx${JAVA_XMX}"
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
  --osm-url="https://download.geofabrik.de/asia/${PBF_NAME}" \
  --water-polygons-path=/map/data/sources/water-polygons-split-3857.zip \
  --natural-earth-path=/map/data/sources/natural_earth_vector.sqlite.zip \
  --lake-centerlines-path=/map/data/sources/lake_centerline.shp.zip \
  --tile-weights=/map/data/sources/tile_weights.tsv.gz \
  --download-dir=/map/data/sources \
  --tmpdir=/map/data/tmp \
  --minzoom=0 \
  --maxzoom=14 \
  --output="/map/${OUTPUT_NAME}" \
  --layer-stats=/map/data/thailand.layerstats.tsv.gz \
  --force

rm -rf "${SCRIPT_DIR}/data/tmp"
echo "build: wrote ${SCRIPT_DIR}/${OUTPUT_NAME} in $(( $(date +%s) - start ))s"
