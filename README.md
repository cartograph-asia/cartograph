<img src="web/public/logo.svg" alt="" width="96" height="96" />

# Cartograph

Cartograph is a community game: a photo for every plot of land in South, Southeast and East Asia. The map is divided into plots of about 0.74 km², and the first Surveyor to upload a geotagged photo into a plot claims it. More about the idea on the [organisation profile](https://github.com/cartograph-asia).

This repository holds the prototype, Thailand only: a server-rendered community front page with Surveyor, plot, photo and region pages and RSS feeds, the map at `/map`, and magic-link sign-in for uploading.

## Run it

You need Docker (with at least 5 GB of memory assigned) and about 3 GB of free disk.

```sh
git clone git@github.com:cartograph-asia/cartograph.git && cd cartograph
map/build.sh                  # first run downloads ~1.7 GB and takes 15–20 min
cd deploy && cp .env.example .env
docker compose up --build
open http://localhost:8080
```

The first start imports the Pattaya fixture (Chon Buri province and its places) and seeds five example Surveyors with 30 synthetic photos, so the front page is populated. To upload, choose "Sign in", enter any address, and open the sign-in link that the api logs (`docker compose logs api | grep sign-in`; no mail is sent while `SMTP_HOST` is empty). Then pick a Surveyor name and upload a geotagged JPEG or PNG from `/map` (the photos in `testdata/` are taken in Pattaya and work).

The stack runs PostgreSQL, Garage (local S3 for the photos), the api (pages, sign-in and JSON API), the map's web build and Caddy, with only port 8080 published. The one-shot services `regions`, `places` and `seed` run before you use it and exit. `docker compose down -v` resets all data.

### All of Thailand

After `map/build.sh`, set in `deploy/.env`:

```sh
IMPORT_PBF=/srv/map/data/thailand-latest.osm.pbf
IMPORT_WATER=/srv/map/data/sources/water-polygons-split-3857.zip
IMPORT_BBOX=
```

and run `docker compose up regions places` (77 provinces and about 18,000 places). Set `SEED_PHOTOS=0` if you do not want the synthetic photos.

## Build the map

`map/build.sh` downloads the Geofabrik extract for Thailand and runs Planetiler in Docker to produce `map/thailand.pmtiles` (zoom 0–14, about 400 MB). Caddy serves it at `/tiles/thailand.pmtiles`, and the muted base style is `web/style.json`. Build times, memory needs and how to refresh the data are in [map/README.md](map/README.md).

## Configuration

Everything is configured through environment variables, all documented in [deploy/.env.example](deploy/.env.example). Every variable has a working default for local use, so `cp .env.example .env` is optional.

## Development

- Backend (Go, `cmd/` and `internal/`): `go test ./...`. Database tests run only when `TEST_DATABASE_URL` points at a Postgres instance.
- Pages (`internal/web`, `html/template`): `go run ./cmd/webpreview` serves every page from in-memory fixtures on 127.0.0.1:8093, with crops of the photos in `testdata/`; add `?type=a|b1|b2` to compare typographic directions.
- Design screenshots: `cd web && npm run screenshots -- round-1` takes the pages of a set in `tools/screenshots/pages.json` at 400 and 1400 px into `docs/design/round-1/`, through playwright-core and the installed Chrome (`CHROME_PATH` for another Chromium).
- Map (`web/`, served under `/map/`): `npm ci && npm run dev:mock` runs against in-memory fixtures; `npm run dev` proxies to the stack on :8080. Open http://localhost:5173/map/.
- Importers: `regions`, `places` and `seed` in `cmd/`; `go run ./cmd/fixture` regenerates the Pattaya fixture in `testdata/` from `map/data/`.
- Briefs: [docs/PROTOTYPE_BRIEF.md](docs/PROTOTYPE_BRIEF.md) (map and JSON API) and [docs/HOME_BRIEF.md](docs/HOME_BRIEF.md) (front page, pages, accounts). [docs/PRINCIPLES.md](docs/PRINCIPLES.md) decides feature questions, [docs/GAMIFICATION.md](docs/GAMIFICATION.md) is the design reference the data model leaves room for, and decisions made while building are in [docs/decisions.md](docs/decisions.md).

## Not there yet (on purpose)

- Comments, ranks, points and leaderboards (the `points` table exists but stays empty), country titles, the "where was this" mode, moderation tools and the e-mail digest.
- HEIC uploads: JPEG and PNG only for now.
- Anything beyond Thailand, and city-level regions.
- An image CDN.
