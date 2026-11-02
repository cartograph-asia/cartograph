#!/usr/bin/env bash
# Merge the osmium cuts of several Geofabrik extracts into one file, so that
# a province relation whose member ways lie in a neighbour's extract can be
# assembled whole (docs/countries/incomplete-relations.md).
#
#   map/merge-cut.sh vietnam philippines
#     reads  map/data/scan/vietnam-latest.osm.pbf.cut.osm.pbf
#            map/data/scan/philippines-latest.osm.pbf.cut.osm.pbf
#     writes map/data/scan/vietnam+philippines-merged.cut.osm.pbf
#
# The first extract is the country's own, the others are neighbours. The
# merged cut is for reading boundaries only: it holds the neighbours' place
# nodes too, so places are read from the country's own cut.
#
# The cuts are made by cmd/countryscan (or map/cut.sh) beside the extracts;
# a missing cut is an error. The merge is skipped when the merged file is
# newer than every cut it is made from. Cuts of different days hold
# different versions of the objects along the borders, so their replication
# timestamps must match (as in build.sh).
#
# osmium runs on the host when installed, else in a throwaway
# debian:bookworm-slim container.
#
# Env overrides:
#   USE_DOCKER   auto (default: host osmium if found), 1 or 0
#   SCAN_DIR     directory of the cuts, default map/data/scan
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCAN_DIR="${SCAN_DIR:-${SCRIPT_DIR}/data/scan}"
DEBIAN_IMAGE="${DEBIAN_IMAGE:-debian:bookworm-slim}"

if [[ $# -lt 2 ]]; then
  echo "usage: $0 <extract> <neighbour extract> [...]   e.g. $0 vietnam philippines" >&2
  exit 2
fi
if [[ ! -d "${SCAN_DIR}" ]]; then
  echo "merge-cut: ${SCAN_DIR}: no such directory" >&2
  exit 1
fi
SCAN_DIR="$(cd "${SCAN_DIR}" && pwd)"

names=()
cuts=()
for name in "$@"; do
  if [[ ! "${name}" =~ ^[a-z0-9-]+$ ]]; then
    echo "merge-cut: ${name}: not a Geofabrik extract name" >&2
    exit 2
  fi
  cut="${SCAN_DIR}/${name}-latest.osm.pbf.cut.osm.pbf"
  if [[ ! -s "${cut}" ]]; then
    echo "merge-cut: ${cut} is missing; cut it first (go run ./cmd/countryscan, or map/cut.sh ${name})" >&2
    exit 1
  fi
  names+=("${name}")
  cuts+=("${cut}")
done

joined="$(IFS=+; echo "${names[*]}")"
OUT_NAME="${joined}-merged.cut.osm.pbf"
OUT="${SCAN_DIR}/${OUT_NAME}"

stale=0
[[ -s "${OUT}" ]] || stale=1
for cut in "${cuts[@]}"; do
  [[ "${cut}" -nt "${OUT}" ]] && stale=1
done
if [[ ${stale} == 0 ]]; then
  echo "merge-cut: ${OUT_NAME} is newer than its cuts, up to date"
  exit 0
fi

osmium_bin="$(command -v osmium || true)"
use_docker="${USE_DOCKER:-auto}"
if [[ "${use_docker}" == auto ]]; then
  if [[ -n "${osmium_bin}" ]]; then use_docker=0; else use_docker=1; fi
fi

# The merge, as a script run on the host or in the container. Arguments: the
# output path, then the cuts.
MERGE='out="$1"; shift
  stamp() { osmium fileinfo -g header.option.osmosis_replication_timestamp "$1"; }
  if [ "$(for f in "$@"; do stamp "$f"; done | sort -u | wc -l)" -ne 1 ]; then
    echo "merge-cut: the cuts are from different days:" >&2
    for f in "$@"; do echo "  $(basename "$f") $(stamp "$f")" >&2; done
    echo "merge-cut: download the older extracts again and cut them" >&2
    exit 3
  fi
  osmium merge --overwrite -f pbf -o "$out" "$@"'

trap 'rm -f "${OUT}.part"' EXIT
echo "merge-cut: ${names[*]} -> ${OUT_NAME}"
if [[ ${use_docker} == 0 ]]; then
  sh -euc "${MERGE}" merge "${OUT}.part" "${cuts[@]}"
else
  parts=()
  for name in "${names[@]}"; do parts+=("/scan/${name}-latest.osm.pbf.cut.osm.pbf"); done
  # apt needs root inside the container; the merged file is handed back to
  # the host user.
  docker run --rm \
    -e HOST_UID="$(id -u)" -e HOST_GID="$(id -g)" -e MERGE="${MERGE}" \
    -v "${SCAN_DIR}:/scan" \
    "${DEBIAN_IMAGE}" \
    sh -euc 'apt-get update -qq >/dev/null
      apt-get install -y -qq --no-install-recommends osmium-tool >/dev/null
      sh -euc "$MERGE" merge "$@"
      chown "$HOST_UID:$HOST_GID" "$1"' \
    merge "/scan/${OUT_NAME}.part" "${parts[@]}"
fi
# Moved in place only when complete.
mv "${OUT}.part" "${OUT}"
echo "merge-cut: wrote ${OUT}"
