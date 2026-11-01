# Cartograph

Cartograph is a community game: a photo for every plot of land in South, Southeast and East Asia. The map is divided into plots of about 0.74 km², and the first Surveyor to upload a geotagged photo into a plot claims it. More about the idea on the [organisation profile](https://github.com/cartograph-asia).

This repository holds the prototype: Thailand only, one page with a map, a side panel and an upload form.

## Run the prototype

You need Docker (with at least 5 GB of memory assigned) and about 3 GB of free disk.

```sh
git clone git@github.com:cartograph-asia/cartograph.git && cd cartograph
map/build.sh                  # first run downloads ~1.7 GB and takes 15–20 min
cd deploy && cp .env.example .env
docker compose up --build
open http://localhost:8080
```

Upload a geotagged JPEG or PNG (the photos in `testdata/` work) and its plot turns surveyed. The MinIO console runs on http://localhost:9001 for debugging. `docker compose down -v` resets all data.

## Build the map

`map/build.sh` downloads the Geofabrik extract for Thailand and runs Planetiler in Docker to produce `map/thailand.pmtiles` (zoom 0–14, about 400 MB). Caddy serves it at `/tiles/thailand.pmtiles`, and the muted base style is `web/style.json`. Build times, memory needs and how to refresh the data are in [map/README.md](map/README.md).

## Configuration

Everything is configured through environment variables, all documented in [deploy/.env.example](deploy/.env.example). Every variable has a working default for local use, so `cp .env.example .env` is optional.

## Development

- Backend (Go, `cmd/api` and `internal/`): `go test ./...`. Database tests run only when `TEST_DATABASE_URL` points at a Postgres instance.
- Frontend (`web/`): `npm ci && npm run dev:mock` runs against in-memory fixtures; `npm run dev` proxies to the stack on :8080.
- The API contract is in [docs/PROTOTYPE_BRIEF.md](docs/PROTOTYPE_BRIEF.md); decisions made while building it are in [docs/decisions.md](docs/decisions.md).

## Not there yet (on purpose)

- Accounts and authentication: the upload form takes a free-text name.
- Ranks, leaderboards, the "where was this" mode, moderation and rate limiting.
- HEIC uploads: JPEG and PNG only for now.
- Country and province boundaries, land/sea distinction in coverage, and anything beyond Thailand.
- An image CDN and self-hosted map fonts (glyphs currently load from the Protomaps assets host).
