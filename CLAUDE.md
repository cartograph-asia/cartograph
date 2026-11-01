# Cartograph – project context for coding agents

Cartograph (cartograph.asia) is a community game: a photo for every plot of land in South, Southeast and East Asia. The map is divided into H3 hexagons ("plots"). Members ("Surveyors") upload a geotagged photo; the first photo claims the plot. See the organisation README at github.com/cartograph-asia for the full concept.

This file is the standing context for every agent working in this repository. Read it before doing anything.

**Principles:** [docs/PRINCIPLES.md](docs/PRINCIPLES.md) decides feature questions; when a proposal conflicts with a principle, the principle wins unless Adrian says otherwise.

**Decisions:** [docs/decisions.md](docs/decisions.md) is the log of what was decided and why, newest last. Read the entries for the area you touch before changing it; append a dated line for every decision you make.

## Status: Alpha (since 2026-11-01)

The prototype goals are met: the stack runs from `docker compose up`, pages are server-rendered, Surveyors sign in with a magic link, plots have names, numbers, provinces and districts. Adrian has decided that the structure is right and the project leaves the prototype stage. What changes:

- **Data is kept.** Treat everything uploaded from now on as permanent. No migration drops or rewrites user content; a schema change that cannot be made without a rewrite is proposed first. Seeds and fixtures (`cmd/seed`, `testdata/`) are for development only; a production stack never seeds.
- **Design is part of the work.** The prototype's look was deliberately not designed; from now on every page change is measured against [docs/DESIGN_BRIEF.md](docs/DESIGN_BRIEF.md) and the style it establishes. Nothing ships with inline styles or one-off CSS.
- **Production is the target.** The stack will run on Adrian's Hetzner host under cartograph.asia with Caddy and real TLS, real SMTP (hello@cartograph.asia via mailbox.org) and backups of Postgres and the photo bucket. Keep every environment difference in `.env`, never in code. A deployment brief follows; do not build hosting-specific pieces before it exists.
- **Briefs still rule.** Each work package comes as `docs/<TOPIC>_BRIEF.md`, written by Adrian with Claude; the coding agent implements on a branch and opens a PR. Open questions go into the PR description, not into guesses.

Stage names, so everyone means the same: *prototype* (done) → *alpha* (now: Adrian and a few invited Surveyors, real domain, data kept) → *beta* (public or by invitation, backups and monitoring in place, nothing resets) → *release*.

## Vocabulary (use these words in code, UI and docs)

- **Plot** – one H3 cell at resolution 8 (~0.74 km²). Never "square", never "hex" in user-facing text (`/about/plots` is the one exception).
- **Surveyor** – a member. **Survey** – the act of uploading a photo into a plot.
- **Claimed / surveyed** – a plot with at least one photo. **Blank / unsurveyed** – no photo yet.
- **First surveyor** – the member whose photo claimed the plot. Stored permanently.
- **Coverage** – percentage of surveyed plots in an area.
- **Wordmark** – the site's mark is `Cartograph.asia`: "Cartograph" in the display size, ".asia" set smaller (about 60 %, same baseline, muted ink) as a visible suffix, never dropped. It appears wherever the site stands as a mark: masthead, footer, `<title>`, `og:site_name`, feed titles, and the sender name and signature of e-mails (plain text there: "Cartograph.asia"). In running text the project is called **Cartograph** ("Cartograph divides the land …"). Never "Cartograph Asia" on the site itself; the social handles keep their own spelling.
- **Plot name** – "{place}, {district} · {province} {number}", formatted only by `views.PlotRef.Name()`. The H3 index appears in links and a details line, never as a title.

## Architecture decisions (settled – do not re-open without asking)

- **Grid:** H3, resolution 8 for plots. Aggregation for overview maps uses H3 parents (res 5–6). H3 is computed in the backend (Go) with `github.com/uber/h3-go/v4`; the database stores the cell index as `text` (15-char hex string). No h3-pg extension.
- **Base map:** OpenStreetMap, built with Planetiler into a single PMTiles archive, served as a static file with HTTP range requests. Style is deliberately muted so the plot layer dominates.
- **Map rendering:** MapLibre GL JS with the `pmtiles` protocol. Plot geometry is computed client-side with `h3-js` for the visible viewport at high zoom; at low zoom the client fetches aggregated coverage per parent cell from the API.
- **Pages:** server-rendered HTML from the Go backend with `html/template` (`internal/web`), complete and readable without JavaScript; a few lines of vanilla JS for lazy small maps and forms are fine. No frontend framework. The map page at `/map` is the one Vite + TypeScript app (`web/`), also without a framework.
- **Backend:** Go (single binary), `net/http` + `chi`, `pgx` for Postgres. Image processing in Go, EXIF via `rwcarlsen/goexif`. OSM import in `internal/geo` (`cmd/regions`, `cmd/places`, `cmd/fixture`).
- **Database:** PostgreSQL 16 (PostGIS available in the image but not used).
- **Photo storage:** any S3-compatible store, accessed only through `internal/storage`. Development: Garage in Docker. Production: decided in the deployment brief (Hetzner Object Storage is the candidate); the api must not care. **The database stores object keys, never full URLs.** Public image URLs are built at runtime from `IMG_BASE_URL` + key.
- **Accounts:** e-mail + magic link, no passwords, one Surveyor name chosen once. No OAuth, no roles, no admin UI; editorial data (`featured`, overrides) is edited in SQL until a brief says otherwise.
- **Deployment:** Docker Compose. Backend image built from scratch/distroless. Everything configured via environment variables, documented in `deploy/.env.example`.

## Conventions

- Code, comments, commit messages, docs: English. Chat with Adrian: German.
- Configuration only via environment variables; no config files with secrets. Keep `.env.example` current.
- Storage access goes through one package (`internal/storage`) with a small interface (`Put`, `Get`, `URL`). Nothing else imports the S3 SDK.
- Image pipeline on upload: validate (JPEG/PNG/HEIC→JPEG), auto-orient, strip EXIF except the fields we keep in the DB, store three variants: `large` (longest side 2000 px, JPEG q85), `medium` (1200 px) and `thumb` (480 px, resized from medium). Keys: `photos/{yyyy}/{mm}/{photo_id}_{variant}.jpg`.
- API is JSON under `/api/`. Errors are `{ "error": "message" }` with proper status codes.
- Migrations: plain SQL files in `migrations/`, applied in order by the backend at start-up. Never edit an applied migration; add a new one. Lead agents number from the next free tens block (e.g. 0080+), sub-agents take the numbers the lead assigns.
- Keep dependencies few. Ask before adding a framework, an ORM, a build tool or a font that is not already in the repo.
- Page weight: the front page stays under 100 KB of HTML and CSS; photo grids use the thumb variant only.
- Tests: a change to templates runs the vocabulary and page tests; a change to imports runs against the Pattaya fixture (`testdata/`).
- Commit small and often, with messages in the form `area: what changed` (e.g. `api: add plot coverage endpoint`).

## Repository layout

```
cartograph/
  cmd/api/            Go entry point: pages, sign-in, JSON API, migrations
  cmd/regions, cmd/places, cmd/fixture, cmd/seed, cmd/webpreview
  internal/
    api/              JSON handlers, routing, middleware
    auth/             magic links, sessions, CSRF
    db/               queries (pgx), migrations runner, views source
    geo/              OSM reading, boundaries, land cells, place naming
    h3/               thin wrappers around h3-go (latlng→cell, parent, boundary)
    importer/         regions and places import
    photos/           image pipeline, EXIF
    storage/          S3 abstraction
    views/            plain structs every page renders from
    web/              templates/, static/ (style.css, logo), page handlers, RSS
  migrations/         *.sql
  docs/               briefs, decisions, principles, pages/ (static Markdown pages)
  web/                Vite app for /map (src/, style.json for the base map)
  map/                scripts to download the OSM extract and build PMTiles
  deploy/             docker-compose.yml, Caddyfile, Dockerfiles, garage.toml, .env.example
  testdata/           Pattaya fixture (OSM extract, water, sample photos)
```

## Data model

The schema lives in `migrations/`; read it there, it is the truth. The core tables are `plots` (cell, first photo and surveyor, counts, region, place name, number), `photos` (variants as keys, position, taken/uploaded times, surveyor), `surveyors` and the auth tables, `regions` (country / province / district, boundaries, land cells, coverage), `places`, `plot_numbers`, `place_naming` and `place_overrides`, `featured`, and the empty gamification hooks (`points`, `tags`, `photo_tags`). Coverage per parent cell is computed on read; materialise later if it gets slow.
