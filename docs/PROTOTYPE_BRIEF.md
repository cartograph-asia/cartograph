# Prototype brief: Thailand, first photo in a plot

You are the lead agent. Read `CLAUDE.md` first; it holds the vocabulary, settled architecture and conventions. Your job is to deliver the prototype defined under "What done means" there, by splitting the work into the sub-agent tasks below, running them, integrating the results and verifying the whole thing end to end.

## Ground rules for orchestration

- Run tasks 1–3 **in parallel** – they are independent. Task 4 depends on the API contract (below), not on the API being finished; it can mock the endpoints until task 3 lands. Task 5 runs last.
- Every sub-agent gets this file and `CLAUDE.md`. Each returns: files changed, how to run its part, what it verified, and open questions.
- The API contract below is fixed. Sub-agents implement against it; they do not redesign it. If a sub-agent finds the contract unworkable, it reports back instead of improvising.
- Prefer the simplest thing that satisfies the acceptance check. The prototype is for seeing whether the game feels right, not for scale.
- Do not add authentication, user accounts, ranks or leaderboards. Those are explicitly out of scope.

## API contract (fixed)

```
POST /api/photos                      multipart/form-data
  fields: photo (file, required), surveyor (string, required),
          lat, lng (optional; used only if the file has no EXIF GPS)
  201 → { "id": "...", "cell": "88...", "claimed": true|false,
          "urls": { "large": "...", "thumb": "..." } }
  400 → no GPS found and no lat/lng given; unsupported file; file > 25 MB

GET  /api/plots?cells=88a,88b,...      up to 2000 res-8 cells
  200 → { "plots": [ { "cell": "...", "photo_count": 3,
                       "first_surveyor": "...", "claimed_at": "...",
                       "thumb": "..." } ] }      (only surveyed cells returned)

GET  /api/plots/{cell}
  200 → { "cell": "...", "photo_count": n, "first_surveyor": "...",
          "photos": [ { "id", "surveyor", "taken_at", "uploaded_at",
                        "urls": { "large", "thumb" } } ] }
  404 → blank plot

GET  /api/coverage?res=5|6&cells=85a,85b,...   parent cells, up to 500
  200 → { "cells": [ { "cell": "85...", "surveyed": 12, "total": 343 } ] }
      total = number of res-8 children (h3 cellToChildrenSize); land/sea
      is not distinguished in the prototype.

GET  /healthz                         200 "ok"
```

Image URLs are `IMG_BASE_URL + "/" + key`.

## Task 1 – Base map pipeline (`map/`)

Produce `map/thailand.pmtiles` and a muted MapLibre style.

1. `map/download.sh`: fetch `thailand-latest.osm.pbf` from Geofabrik into `map/data/` (skip if present).
2. `map/build.sh`: run Planetiler (Java, use the released jar or the Docker image `ghcr.io/onthegomap/planetiler`) on the extract with the default OpenMapTiles-compatible profile, output `map/thailand.pmtiles`, zoom 0–14. Document RAM needs; must work on a 24 GB Mac.
3. `web/style.json`: a MapLibre style using the `pmtiles://` source. Muted palette: light grey land, slightly darker water, thin grey roads, labels only for places (no POIs, no road names). The plot layer will be drawn on top, so nothing in the base style may be saturated.
4. `deploy/Caddyfile` snippet serving `/tiles/thailand.pmtiles` as a static file with range requests and CORS.

Verify: `pmtiles show map/thailand.pmtiles` reports Thailand bounds and zoom 0–14; opening the style in a bare MapLibre page renders Bangkok.

## Task 2 – Infrastructure (`deploy/`)

Docker Compose with: `postgres` (postgis/postgis:16 image), `minio` (with a one-shot `mc` container that creates the `cartograph` bucket and sets anonymous read on `photos/*`), `api` (built from `cmd/api`), `web` (static build served by Caddy), `caddy` as the single entry point on `:8080` routing `/api/*` → api, `/tiles/*` → static, `/img/*` → minio bucket, everything else → web.

Deliverables: `docker-compose.yml`, `Caddyfile`, `.env.example` with every variable (`DATABASE_URL`, `S3_ENDPOINT`, `S3_BUCKET`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_USE_PATH_STYLE`, `IMG_BASE_URL`, `MAX_UPLOAD_MB`), a multi-stage `Dockerfile` for the api (final stage distroless or scratch, non-root).

Verify: `docker compose up` brings everything up healthy; `curl localhost:8080/healthz` once task 3 exists; MinIO console reachable; bucket present.

## Task 3 – Backend (`cmd/api`, `internal/`, `migrations/`)

Implement the API contract in Go.

- `internal/h3`: `CellFromLatLng(lat, lng, res)`, `Parent(cell, res)`, `ChildrenCount(cell, childRes)`; thin wrappers so the rest of the code never imports h3-go directly.
- `internal/storage`: interface `Store { Put(ctx, key, contentType, r io.Reader) error; URL(key) string }` with an S3 implementation driven by env vars (path-style switch!).
- `internal/photos`: decode, auto-orient, EXIF GPS + DateTimeOriginal extraction, resize to `large` and `thumb`, encode JPEG. Reject anything over `MAX_UPLOAD_MB`.
- `internal/db`: migrations runner (apply `migrations/*.sql` in order, record in `schema_migrations`), queries for the four endpoints. Claiming a plot is one transaction: insert photo, upsert plot (`first_photo` and `first_surveyor` only set when the row is new), increment `photo_count`.
- `internal/api`: handlers, chi router, JSON errors, request logging, CORS for local dev.
- Tests: unit tests for `h3` wrappers and for the EXIF/GPS extraction (use two small fixture images, one with GPS and one without). One handler test for `POST /api/photos` using the storage interface with an in-memory fake.

Verify: `go test ./...` green; `go vet` clean; uploading a fixture image via `curl` returns 201 with a cell; second upload to the same cell returns `claimed: false` and `photo_count` 2.

## Task 4 – Frontend (`web/`)

Vite + TypeScript, MapLibre GL JS, `pmtiles`, `h3-js`. One page.

- Map centred on Thailand (zoom 6), base style from task 1.
- **Plot layer (zoom ≥ 10):** on `moveend`, compute res-8 cells covering the viewport with `polygonToCells`, cap at 2000 (if more, raise the threshold zoom), call `GET /api/plots?cells=`, draw all cells as a GeoJSON fill layer: blank = transparent with a thin outline, surveyed = green fill ~0.35 opacity, hover = brighter outline. Debounce requests; cache responses per cell for the session.
- **Coverage layer (zoom < 10):** compute parent cells (res 5 for zoom < 7, res 6 otherwise) for the viewport, call `GET /api/coverage`, draw fill with opacity proportional to `surveyed/total` (interpolate 0 → 0.05, 1 → 0.6). Blank parents still get a faint outline so the grid is visible.
- **Side panel:** clicking a plot opens a panel with the cell id, status, first surveyor, photo thumbnails (click → large). For any plot: an upload form (file, surveyor name remembered in `localStorage`, optional "use my location" button that fills lat/lng via the Geolocation API when the file has no GPS). On success, refresh that cell and animate it to surveyed.
- Mobile layout: panel slides up from the bottom below 768 px width. Big enough tap targets; the upload button is the primary action.
- Until task 3 is available, develop against a mock module that returns fixture data for the four endpoints; switch to the real API with a Vite proxy.

Verify: `npm run build` clean, no TypeScript errors; manual check of the acceptance flow in the browser; at zoom 12 over Bangkok the request carries fewer than 2000 cells and the grid renders in under a second.

## Task 5 – Integration and acceptance (lead agent, after 1–4)

1. Bring everything up with `docker compose up --build` from a clean checkout.
2. Walk the acceptance flow from `CLAUDE.md` yourself: map renders, grid shows, upload a real geotagged photo (take one from `testdata/`, add two more with different locations), the plot turns surveyed, panel shows the photo, reload persists, zoom out shows coverage shading.
3. Write `README.md` for the repository: what it is (two sentences, link to the org profile), how to run the prototype in five commands, how to build the map, where the env vars are documented, what is deliberately not there yet.
4. Record every decision a sub-agent had to make that is not in `CLAUDE.md` in `docs/decisions.md`, one line each, dated.
5. Report: what works, what does not, what you would change in the brief next time.

## Out of scope (do not build, do not scaffold)

Authentication, accounts, ranks, leaderboards, country/province boundaries, the "where was this" mode, moderation, image CDN, rate limiting, anything beyond Thailand.
