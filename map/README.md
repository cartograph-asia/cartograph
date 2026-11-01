# Base map

Builds `map/asia.pmtiles`, an OpenMapTiles-schema vector tile archive
(zoom 0–14) from OpenStreetMap, for the base style in `web/style.json`.
Caddy serves this directory at `/tiles/` with range requests.

The archive covers the map's whole area (88,-2 – 114,28, the `maxBounds` in
`web/src/main.ts`) and holds OSM detail for Thailand and its neighbours:
Myanmar, Laos, Cambodia, Vietnam and Malaysia/Singapore/Brunei, merged into
one file with osmium. Cartograph's plots, regions and places are still
Thailand's only; `cmd/regions`, `cmd/places` and `cmd/fixture` read
`thailand-latest.osm.pbf` alone.

## Build

Requires `curl`, and either Java 21+ and osmium on the host or Docker:

```sh
brew install openjdk@21 osmium-tool   # host build (preferred on a Mac)
```

With both installed, `build.sh` runs osmium and Planetiler on the host and
fetches Planetiler's jar (pinned, `PLANETILER_VERSION`) into `map/data/`.
Otherwise it runs both in Docker (`USE_DOCKER=1` forces that). On a Mac
the host build is the better choice for large sets: it may use all of the
machine's memory, while Docker Desktop gives it a fixed VM and slow access
to `map/`.

```sh
map/download.sh   # Geofabrik extracts -> map/data/<name>-latest.osm.pbf (each skipped if present)
map/build.sh      # osmium merge + Planetiler -> map/asia.pmtiles (runs download.sh first)
```

Both scripts work from any directory. `EXTRACTS` (space-separated Geofabrik
`asia/` names) changes the set; `map/download.sh thailand` fetches one. To
refresh the OSM data, delete the extracts in `map/data/` and run `build.sh`
again; the merged `map/data/region-latest.osm.pbf` is rebuilt whenever an
extract is newer. In Docker the merge runs osmium from Debian's package in
a throwaway `debian:bookworm-slim` container.

On the first build Planetiler also downloads its auxiliary sources into
`map/data/sources/` (water polygons ~930 MB, Natural Earth ~430 MB, lake
centerlines ~80 MB, tile weights ~6 MB). Later builds reuse them.

Everything under `map/data/` and the `.pmtiles` output is git-ignored.

## Resources (measured on an M-series Mac, 16 GB RAM, Docker Desktop with 7.7 GB / 10 CPUs)

| | Thailand alone | Thailand and neighbours (default) |
|---|---|---|
| OSM extracts | 330 MB | 1.3 GB merged (six extracts) |
| Merge | – | ~1 min, including installing osmium |
| Build time | ~95 s (Docker) | ~3½ min on the host incl. merge; ~7 min in Docker (sources already downloaded) |
| JVM heap peak | ~2.1 GB | 2–4 GB of 5 GB (`-Xmx5g`, the default); 6.2 GB resident on the host incl. memory-mapped files |
| Disk | ~1.7 GB in `map/data/` | ~3.8 GB in `map/data/` |
| Output | ~400 MB | `asia.pmtiles`, ~1.4 GB |

In Docker, Docker Desktop needs at least ~6 GB memory assigned for the default set.
Override the heap with `JAVA_XMX=4g map/build.sh`, the set with
`EXTRACTS=thailand map/build.sh` and the image with `PLANETILER_IMAGE=...`.

All extracts must be from the same Geofabrik day: an extract of another day
holds other versions of the objects along the borders, and Planetiler stops
at the duplicate IDs. `build.sh` checks the replication timestamps before it
merges and names the files to delete and fetch again.

## Check the output

```sh
docker run --rm -v "$PWD/map:/map:ro" protomaps/go-pmtiles:v1.31.2 show /map/asia.pmtiles
```

Expect bounds 88,-2 – 114,28 and zoom 0–14.

## Licence

Map data © OpenStreetMap contributors (ODbL); schema © OpenMapTiles. Any map
built from these tiles must show that credit (the style's source attribution
does).
