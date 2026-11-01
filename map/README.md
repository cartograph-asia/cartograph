# Base map

Builds `map/thailand.pmtiles`, an OpenMapTiles-schema vector tile archive
(zoom 0–14) from OpenStreetMap, for the muted base style in `web/style.json`.
Caddy serves this directory at `/tiles/` with range requests.

## Build

Requires Docker and `curl`. No local Java needed.

```sh
map/download.sh   # Geofabrik thailand-latest.osm.pbf -> map/data/ (skipped if present)
map/build.sh      # Planetiler -> map/thailand.pmtiles (runs download.sh if needed)
```

Both scripts work from any directory. To refresh the OSM data, delete
`map/data/thailand-latest.osm.pbf` and run `build.sh` again.

On the first build Planetiler also downloads its auxiliary sources into
`map/data/sources/` (water polygons ~930 MB, Natural Earth ~430 MB, lake
centerlines ~80 MB, tile weights ~6 MB). Later builds reuse them.

Everything under `map/data/` and the `.pmtiles` output is git-ignored.

## Resources (measured on an M-series Mac, 16 GB RAM, Docker Desktop with 7.7 GB / 10 CPUs)

| | |
|---|---|
| OSM extract | 330 MB (44 M nodes, 5.3 M ways) |
| Build time | ~95 s (sources already downloaded) |
| JVM heap peak | ~2.1 GB of 4 GB (`-Xmx4g`, the default) |
| Container memory peak | ~3.5 GB (includes memory-mapped temp files) |
| Disk | ~1.7 GB in `map/data/`, plus ~400 MB output |
| Output | `thailand.pmtiles`, ~400 MB |

A 16 GB Mac is plenty; Docker Desktop needs at least ~5 GB memory assigned.
Override the heap with `JAVA_XMX=3g map/build.sh` and the image with
`PLANETILER_IMAGE=...`.

## Check the output

```sh
docker run --rm -v "$PWD/map:/map:ro" protomaps/go-pmtiles:v1.31.2 show /map/thailand.pmtiles
```

Expect bounds roughly 97.1,5.6 – 105.6,20.5 and zoom 0–14.

## Licence

Map data © OpenStreetMap contributors (ODbL); schema © OpenMapTiles. Any map
built from these tiles must show that credit (the style's source attribution
does).
