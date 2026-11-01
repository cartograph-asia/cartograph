#!/usr/bin/env bash
# Build map/asia.pmtiles from Geofabrik extracts with Planetiler
# (OpenMapTiles-compatible default profile, zoom 0-14).
#
# The extracts (EXTRACTS: Thailand and its neighbours, see download.sh) are
# merged into one file with osmium; the merge is redone only when an extract
# is newer than the merged file. Planetiler then builds the archive.
#
# Both run on the host when they are installed (Java 21+ and osmium, e.g.
# `brew install openjdk@21 osmium-tool`), and otherwise in Docker. On the
# host the build may use all of the machine's memory and its disk directly;
# Docker Desktop on a Mac gives it a fixed VM and slow access to map/.
# Planetiler's own jar is fetched into map/data/ on first use.
#
# Auxiliary sources (water polygons, Natural Earth, lake centerlines) are
# downloaded by Planetiler on first run into map/data/sources/ and reused
# afterwards (--download only fetches files that are missing; the existing
# OSM extract is never re-downloaded). Re-running rebuilds the archive.
# Planetiler wipes --tmpdir, so it points at a directory used for nothing else.
#
# Env overrides:
#   USE_DOCKER        auto (default: host tools if found), 1 or 0
#   JAVA              java binary for the host build (default: Homebrew's
#                     openjdk@21, else java on PATH)
#   PLANETILER_VERSION  pinned release, default 0.10.2 (jar and Docker image)
#   JAVA_XMX          JVM heap, default 5g (Thailand and neighbours peak at
#                     ~3.8 GB; node locations and the feature sort live on
#                     disk, not in the heap; see README.md)
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
PLANETILER_VERSION="${PLANETILER_VERSION:-0.10.2}"
PLANETILER_IMAGE="${PLANETILER_IMAGE:-ghcr.io/onthegomap/planetiler:${PLANETILER_VERSION}}"
PLANETILER_JAR="${SCRIPT_DIR}/data/planetiler-${PLANETILER_VERSION}.jar"
JAVA_XMX="${JAVA_XMX:-5g}"
BOUNDS="${BOUNDS:-88,-2,114,28}"
DEBIAN_IMAGE="${DEBIAN_IMAGE:-debian:bookworm-slim}"
EXTRACTS="${EXTRACTS:-thailand myanmar laos cambodia vietnam malaysia-singapore-brunei}"
PBF_NAME="region-latest.osm.pbf"
OUTPUT_NAME="asia.pmtiles"
export EXTRACTS

# Host tools, if they work (macOS ships a /usr/bin/java stub that only
# asks to install Java).
java_bin="${JAVA:-}"
if [[ -z "${java_bin}" ]]; then
  for c in /opt/homebrew/opt/openjdk@21/bin/java "$(command -v java || true)"; do
    if [[ -n "${c}" ]] && "${c}" -version >/dev/null 2>&1; then
      java_bin="${c}"
      break
    fi
  done
fi
osmium_bin="$(command -v osmium || true)"
use_docker="${USE_DOCKER:-auto}"
if [[ "${use_docker}" == auto ]]; then
  if [[ -n "${java_bin}" && -n "${osmium_bin}" ]]; then use_docker=0; else use_docker=1; fi
fi

"${SCRIPT_DIR}/download.sh"

# The merge, as a script run on the host or in a Debian container. Arguments: the
# output path, then the extracts. Extracts of different days hold different
# versions of the objects along the borders, and Planetiler stops at the
# duplicate IDs, so their replication timestamps must match.
MERGE='out="$1"; shift
  stamp() { osmium fileinfo -g header.option.osmosis_replication_timestamp "$1"; }
  if [ "$(for f in "$@"; do stamp "$f"; done | sort -u | wc -l)" -ne 1 ]; then
    echo "merge: the extracts are from different days:" >&2
    for f in "$@"; do echo "  $(basename "$f") $(stamp "$f")" >&2; done
    echo "merge: delete the older ones in map/data/ and run build.sh again" >&2
    exit 3
  fi
  osmium merge --overwrite -o "$out" -f pbf "$@"'

read -r -a names <<<"${EXTRACTS}"
stale=0
for name in "${names[@]}"; do
  [[ "${SCRIPT_DIR}/data/${name}-latest.osm.pbf" -nt "${SCRIPT_DIR}/data/${PBF_NAME}" ]] && stale=1
done
if [[ ! -s "${SCRIPT_DIR}/data/${PBF_NAME}" || ${stale} == 1 ]]; then
  echo "merge: ${EXTRACTS} -> data/${PBF_NAME}"
  if [[ ${use_docker} == 0 ]]; then
    parts=()
    for name in "${names[@]}"; do parts+=("${SCRIPT_DIR}/data/${name}-latest.osm.pbf"); done
    sh -euc "${MERGE}" merge "${SCRIPT_DIR}/data/${PBF_NAME}.part" "${parts[@]}"
  else
    parts=()
    for name in "${names[@]}"; do parts+=("/map/data/${name}-latest.osm.pbf"); done
    # apt needs root inside the container; the merged file is handed back to
    # the host user.
    docker run --rm \
      -e HOST_UID="$(id -u)" -e HOST_GID="$(id -g)" -e MERGE="${MERGE}" \
      -v "${SCRIPT_DIR}:/map" \
      "${DEBIAN_IMAGE}" \
      sh -euc 'apt-get update -qq >/dev/null
        apt-get install -y -qq --no-install-recommends osmium-tool >/dev/null
        sh -euc "$MERGE" merge "$@"
        chown "$HOST_UID:$HOST_GID" "$1"' \
      merge "/map/data/${PBF_NAME}.part" "${parts[@]}"
  fi
  # Moved in place only when complete.
  mv "${SCRIPT_DIR}/data/${PBF_NAME}.part" "${SCRIPT_DIR}/data/${PBF_NAME}"
fi

mkdir -p "${SCRIPT_DIR}/data/sources" "${SCRIPT_DIR}/data/tmp"

if [[ ${use_docker} == 0 ]]; then
  if [[ ! -s "${PLANETILER_JAR}" ]]; then
    echo "build: fetching planetiler ${PLANETILER_VERSION}"
    curl -fL --retry 3 -o "${PLANETILER_JAR}.part" \
      "https://github.com/onthegomap/planetiler/releases/download/v${PLANETILER_VERSION}/planetiler.jar"
    mv "${PLANETILER_JAR}.part" "${PLANETILER_JAR}"
  fi
  root="${SCRIPT_DIR}"
  planetiler=("${java_bin}" "-Xmx${JAVA_XMX}" -jar "${PLANETILER_JAR}")
  echo "build: planetiler ${PLANETILER_VERSION} on the host ($("${java_bin}" -version 2>&1 | head -1)), -Xmx${JAVA_XMX}, bounds ${BOUNDS}"
else
  # map/ is mounted at /map; every path is explicit so nothing depends on the
  # image's working directory.
  root=/map
  planetiler=(docker run --rm --user "$(id -u):$(id -g)" -e JAVA_TOOL_OPTIONS="-Xmx${JAVA_XMX}"
    -v "${SCRIPT_DIR}:/map" "${PLANETILER_IMAGE}")
  echo "build: ${PLANETILER_IMAGE} in Docker, -Xmx${JAVA_XMX}, bounds ${BOUNDS}"
fi
start=$(date +%s)

"${planetiler[@]}" \
  --download \
  --osm-path="${root}/data/${PBF_NAME}" \
  --water-polygons-path="${root}/data/sources/water-polygons-split-3857.zip" \
  --natural-earth-path="${root}/data/sources/natural_earth_vector.sqlite.zip" \
  --lake-centerlines-path="${root}/data/sources/lake_centerline.shp.zip" \
  --tile-weights="${root}/data/sources/tile_weights.tsv.gz" \
  --download-dir="${root}/data/sources" \
  --tmpdir="${root}/data/tmp" \
  --minzoom=0 \
  --maxzoom=14 \
  --bounds="${BOUNDS}" \
  --output="${root}/data/${OUTPUT_NAME}" \
  --layer-stats="${root}/data/region.layerstats.tsv.gz" \
  --force

rm -rf "${SCRIPT_DIR}/data/tmp"
# Built beside the served file and moved in place in one step, so Caddy never
# serves a half-written archive.
mv "${SCRIPT_DIR}/data/${OUTPUT_NAME}" "${SCRIPT_DIR}/${OUTPUT_NAME}"
echo "build: wrote ${SCRIPT_DIR}/${OUTPUT_NAME} in $(( $(date +%s) - start ))s"
