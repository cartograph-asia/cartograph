#!/usr/bin/env bash
# Cut a Geofabrik extract down to what the country import and the scan read:
#
#   map/cut.sh laos
#   map/cut.sh malaysia-singapore-brunei
#
# Writes map/data/scan/<extract>-latest.osm.pbf.cut.osm.pbf from
# map/data/<extract>-latest.osm.pbf: the boundary=administrative relations
# with their ways and nodes, and the place=* nodes (a small part of a large
# extract). It is the same cut as internal/countryscan/cut.go makes, for a host
# without Go: cmd/regions, cmd/places and cmd/import read it when it exists and
# is newer than the extract. Nothing is done when the cut is already newer
# than the extract.
#
# osmium runs on the host when it is installed (`brew install osmium-tool`),
# and otherwise in a throwaway Debian container. The file names are arguments
# of the container's script, never part of it.
#
# Env overrides:
#   USE_DOCKER   auto (default: host osmium if found), 1 or 0
#   DATA_DIR     the directory holding the extracts, default map/data
#   DEBIAN_IMAGE default debian:bookworm-slim
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_DIR="${DATA_DIR:-${SCRIPT_DIR}/data}"
DEBIAN_IMAGE="${DEBIAN_IMAGE:-debian:bookworm-slim}"

if [[ $# -ne 1 || ! "$1" =~ ^[a-z0-9][a-z0-9-]*$ ]]; then
  echo "usage: map/cut.sh <extract>   (a Geofabrik name such as laos or east-timor)" >&2
  exit 2
fi
name="$1"
in="${DATA_DIR}/${name}-latest.osm.pbf"
out_name="${name}-latest.osm.pbf.cut.osm.pbf"
out_dir="${DATA_DIR}/scan"
out="${out_dir}/${out_name}"

if [[ ! -s "${in}" ]]; then
  echo "cut: ${in} not found (map/download.sh fetches the extracts)" >&2
  exit 1
fi
if [[ -s "${out}" && "${out}" -nt "${in}" ]]; then
  echo "cut: ${out} is newer than the extract, nothing to do"
  exit 0
fi
mkdir -p "${out_dir}"

osmium_bin="$(command -v osmium || true)"
use_docker="${USE_DOCKER:-auto}"
if [[ "${use_docker}" == auto ]]; then
  if [[ -n "${osmium_bin}" ]]; then use_docker=0; else use_docker=1; fi
fi

# Built beside the result and moved in place only when complete.
part="${out}.part"
trap 'rm -f "${part}"' EXIT
start=$(date +%s)
if [[ ${use_docker} == 0 ]]; then
  if [[ -z "${osmium_bin}" ]]; then
    echo "cut: osmium is not installed (brew install osmium-tool), or use USE_DOCKER=1" >&2
    exit 1
  fi
  echo "cut: ${name} with osmium on the host"
  "${osmium_bin}" tags-filter --overwrite -f pbf -o "${part}" "${in}" \
    r/boundary=administrative n/place
else
  echo "cut: ${name} with osmium in ${DEBIAN_IMAGE}"
  # apt needs root inside the container; the cut is handed back to the host
  # user.
  docker run --rm \
    -e HOST_UID="$(id -u)" -e HOST_GID="$(id -g)" \
    -v "${DATA_DIR}:/in:ro" -v "${out_dir}:/out" \
    "${DEBIAN_IMAGE}" \
    sh -euc 'apt-get update -qq >/dev/null
      apt-get install -y -qq --no-install-recommends osmium-tool >/dev/null
      osmium tags-filter --overwrite -f pbf -o "/out/$2" "/in/$1" r/boundary=administrative n/place
      chown "$HOST_UID:$HOST_GID" "/out/$2"' \
    cut "${name}-latest.osm.pbf" "$(basename "${part}")"
fi
mv "${part}" "${out}"
echo "cut: wrote ${out} in $(( $(date +%s) - start ))s"
