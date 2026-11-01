# Cartograph – project context for coding agents

Cartograph (cartograph.asia) is a community game: a photo for every plot of land in South, Southeast and East Asia. The map is divided into H3 hexagons ("plots"). Members ("Surveyors") upload a geotagged photo; the first photo claims the plot. See the organisation README at github.com/cartograph-asia for the full concept.

This file is the standing context for every agent working in this repository. Read it before doing anything.

## Vocabulary (use these words in code, UI and docs)

- **Plot** – one H3 cell at resolution 8 (~0.74 km²). Never "square", never "hex" in user-facing text.
- **Surveyor** – a member. **Survey** – the act of uploading a photo into a plot.
- **Claimed / surveyed** – a plot with at least one photo. **Blank / unsurveyed** – no photo yet.
- **First surveyor** – the member whose photo claimed the plot. Stored permanently.
- **Coverage** – percentage of surveyed plots in an area.

## Architecture decisions (settled – do not re-open without asking)

- **Grid:** H3, resolution 8 for plots. Aggregation for overview maps uses H3 parents (res 5–6). H3 is computed in the backend (Go) with `github.com/uber/h3-go/v4`; the database stores the cell index as `text` (15-char hex string). No h3-pg extension in the prototype.
- **Base map:** OpenStreetMap, built with Planetiler into a single PMTiles archive, served as a static file with HTTP range requests. Style is deliberately muted so the plot layer dominates.
- **Map rendering:** MapLibre GL JS with the `pmtiles` protocol. Plot geometry is computed client-side with `h3-js` for the visible viewport at high zoom; at low zoom the client fetches aggregated coverage per parent cell from the API.
- **Backend:** Go (single binary), `net/http` + `chi`, `pgx` for Postgres. Image processing in Go (`disintegration/imaging` or `bimg`), EXIF via `rwcarlsen/goexif`.
- **Database:** PostgreSQL 16 (PostGIS available in the image but not required for the prototype).
- **Photo storage:** any S3-compatible store, accessed only through the AWS S3 SDK (or `minio-go`). Prototype: MinIO in Docker. Production: Hetzner Object Storage. **The database stores object keys, never full URLs.** Public image URLs are built at runtime from `IMG_BASE_URL` + key.
- **Frontend:** Vite + TypeScript, no framework for the prototype. One page: map, side panel, upload form.
- **Deployment:** Docker Compose. Backend image built from scratch/distroless. Everything configured via environment variables, documented in `.env.example`.

## Conventions

- Code, comments, commit messages, docs: English.
- Configuration only via environment variables; no config files with secrets. Keep `.env.example` current.
- Storage access goes through one package (`internal/storage`) with a small interface (`Put`, `Get`, `URL`). Nothing else imports the S3 SDK.
- Image pipeline on upload: validate (JPEG/PNG/HEIC→JPEG), auto-orient, strip EXIF except the fields we keep in the DB, store two variants: `large` (longest side 2000 px, JPEG q85) and `thumb` (longest side 480 px). Keys: `photos/{yyyy}/{mm}/{photo_id}_{variant}.jpg`.
- API is JSON under `/api/`. Errors are `{ "error": "message" }` with proper status codes.
- No authentication in the prototype. The upload form takes a free-text surveyor name; it is stored as-is. Auth comes later; do not build a half auth system now.
- Migrations: plain SQL files in `migrations/`, applied in order by the backend at start-up (or by `golang-migrate`). Never edit an applied migration; add a new one.
- Keep dependencies few. Ask before adding a framework, an ORM, or a build tool that is not already in the repo.
- Commit small and often, with messages in the form `area: what changed` (e.g. `api: add plot coverage endpoint`).

## Repository layout

```
cartograph/
  cmd/api/            Go entry point
  internal/
    api/              HTTP handlers, routing
    h3/               thin wrappers around h3-go (latlng→cell, parent, boundary)
    storage/          S3 abstraction
    photos/           image pipeline, EXIF
    db/               queries (pgx), migrations runner
  migrations/         *.sql
  web/                Vite app (index.html, src/, style.json for the base map)
  map/                scripts to download the OSM extract and build PMTiles
  deploy/             docker-compose.yml, Caddyfile, .env.example
  docs/               briefs and decisions
```

## Data model (prototype)

```sql
plots (
  cell        text primary key,          -- H3 res-8 index
  first_photo uuid,
  first_surveyor text,
  claimed_at  timestamptz,
  photo_count int not null default 0
);
photos (
  id          uuid primary key,
  cell        text not null references plots(cell),
  surveyor    text not null,
  key_large   text not null,
  key_thumb   text not null,
  lat         double precision not null,
  lng         double precision not null,
  taken_at    timestamptz,
  uploaded_at timestamptz not null default now()
);
create index on photos(cell);
```

Coverage per parent cell is computed on read for the prototype (`cellToParent` in Go, group by). Materialise later if it gets slow.

## What "done" means for the prototype

`docker compose up` from `deploy/`, open `http://localhost:8080`, see Thailand on a muted base map with a hex grid; upload a geotagged photo; the plot turns surveyed and shows the photo in the side panel; reload the page and it is still there; zoom out and see coverage shading per parent cell.
