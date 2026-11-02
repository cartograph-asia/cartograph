# Test data

- `IMG_*.jpeg` – Adrian's own photos for the acceptance run and screenshots, released under CC BY-SA 4.0.
- `pattaya.osm.gz` – OSM XML: the complete Chon Buri province boundary relation (with its ways and nodes) and the city/town/village nodes inside the Pattaya box `100.82,12.82,100.98,13.02`. Metadata (users, timestamps) is stripped.
- `pattaya-water.shp` – the OSM water polygons (`water-polygons-split-3857`, EPSG:3857) clipped to the same box. Only the `.shp` geometry file; the importers do not read `.dbf`/`.shx`.

Both are cut from the full inputs in `map/data/` by `cmd/fixture` (run from the repository root):

    go run ./cmd/fixture

The fixture area and the importers' `-bbox` for it are `cmd/fixture.PattayaBBox`.

Map data © OpenStreetMap contributors, available under the Open Database Licence (ODbL) 1.0, https://www.openstreetmap.org/copyright. The water polygons are derived from OpenStreetMap data by osmdata.openstreetmap.de and are also ODbL.

`testdata/commons/` (git-ignored, made by `go run ./cmd/commons`) holds free photos from Wikimedia Commons for development profiles, with author and licence in its `manifest.json` and `CREDITS.md`; see `docs/SEED_PHOTOS_BRIEF.md`. They are never committed and not for production.

