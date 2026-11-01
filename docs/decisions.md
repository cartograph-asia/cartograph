# Decisions

Decisions made while building Cartograph that are not in `CLAUDE.md` or a brief. One line each, newest at the bottom.

## Lead agent

- 2026-11-01 – HEIC uploads are rejected with 400 (JPEG and PNG only); HEIC→JPEG needs libheif via cgo and would rule out a distroless image.
- 2026-11-01 – Work happens on the `prototype` branch with one commit per area.
- 2026-11-01 – The api image is built with `CGO_ENABLED=1` (h3-go wraps the C library) and runs on `gcr.io/distroless/cc-debian12:nonroot`, not scratch.
- 2026-11-01 – `deploy/Caddyfile` is the only Caddyfile; the map pipeline does not ship a snippet.
- 2026-11-01 – Public image URLs are `IMG_BASE_URL/<key>` with `IMG_BASE_URL=http://localhost:8080/img`; Caddy maps `/img/<key>` to `/<bucket>/<key>` on MinIO. *(superseded, see Review follow-ups)*
- 2026-11-01 – Added env vars `API_ADDR`, `S3_USE_SSL`, `S3_REGION` and `HTTP_PORT` to the brief's list.
- 2026-11-01 – Migrations are embedded into the binary from `migrations/embed.go` (`go:embed`), since the final image has no filesystem to read them from.

## Map

- 2026-11-01 – Planetiler runs from `ghcr.io/onthegomap/planetiler:0.10.2` in Docker with `-Xmx4g` (Thailand peaks at ~2.1 GB heap, ~3.5 GB container memory); pmtiles CLI is `protomaps/go-pmtiles:v1.31.2`.
- 2026-11-01 – Planetiler's auxiliary sources live in `map/data/sources/` and its temp dir in `map/data/tmp/`; all of `map/data/` and `*.pmtiles` are git-ignored.
- 2026-11-01 – `--osm-url` is set to the Geofabrik Thailand URL so a missing extract can never silently build Planetiler's default area.
- 2026-11-01 – The Planetiler container runs as the host user so output files belong to the developer.
- 2026-11-01 – The extract's md5 check is best effort with a 30 s timeout (Geofabrik's `.md5` endpoint can hang).
- 2026-11-01 – Glyphs load from the Protomaps basemaps-assets fonts on GitHub Pages (Noto Sans, no key). *(superseded, see Review follow-ups)*
- 2026-11-01 – Labels prefer `name:en`, then `name_en`, `name:latin`, `name`; MapLibre does not shape Thai script well, so English names come first.
- 2026-11-01 – The style adds very low-contrast wood/park fills, buildings from z14 and a dashed country boundary; water names, POIs, road names and house numbers are hidden.

## Backend

- 2026-11-01 – HEIC is detected by the ISO-BMFF `ftyp` brand (heic/heix/hevc/hevx/heim/heis/hevm/hevs/mif1/msf1).
- 2026-11-01 – Coverage uses `plots.parent5` / `plots.parent6` columns (migration 0002), set on insert and indexed; 0001 matches `CLAUDE.md` exactly.
- 2026-11-01 – The plot row is upserted before the photo row in the same transaction (photo UUID generated in Go); `claimed` comes from `RETURNING (xmax = 0)`.
- 2026-11-01 – `claimed_at`, `uploaded_at` and the key's `yyyy/mm` all use the server's upload time in UTC.
- 2026-11-01 – `taken_at` is DateTimeOriginal adjusted by OffsetTimeOriginal; without an offset the wall-clock time is stored as UTC.
- 2026-11-01 – EXIF GPS wins over form `lat`/`lng`; GPS exactly at 0,0 counts as missing; a lone or out-of-range `lat`/`lng` is a 400.
- 2026-11-01 – EXIF is not read from PNGs, so PNG uploads always need `lat`/`lng`.
- 2026-11-01 – Images above 100 megapixels are rejected before decoding; at most two images are processed at once.
- 2026-11-01 – Transparent PNGs are flattened onto white before JPEG encoding; `thumb` is JPEG q80.
- 2026-11-01 – `surveyor` is trimmed, required, and at most 100 characters.
- 2026-11-01 – The multipart body is held in memory (no temp files on distroless); the body limit is `MAX_UPLOAD_MB` + 1 MB.
- 2026-11-01 – The `thumb` in `GET /api/plots` is the claiming photo's thumbnail.
- 2026-11-01 – An empty `cells` parameter returns an empty list; cells are de-duplicated and lower-cased.
- 2026-11-01 – Stored objects carry `Cache-Control: public, max-age=31536000, immutable` (keys never change).
- 2026-11-01 – The api checks that the bucket exists at start-up and only logs a warning if it does not; it never creates it.
- 2026-11-01 – `/healthz` is liveness only and does not ping the database.

## Infrastructure

- 2026-11-01 – MinIO runs from the community build `pgsty/minio` / `pgsty/mc`, because upstream no longer publishes container images. *(superseded, see Review follow-ups)*
- 2026-11-01 – `postgis/postgis:16-3.6` is amd64-only and runs under emulation on Apple Silicon (`platform: linux/amd64`). *(superseded, see Review follow-ups)*
- 2026-11-01 – The api uses its own bucket-scoped MinIO user (`cartograph-app`), created by `minio-init`, not the root credentials. *(superseded, see Review follow-ups)*
- 2026-11-01 – Only host ports 8080 (Caddy) and 127.0.0.1:9001 (MinIO console) are published. *(superseded, see Review follow-ups)*
- 2026-11-01 – Caddy serves only `*.pmtiles` from the `map/` mount; `/img` allows GET/HEAD only; the request body limit for `/api/*` is 30 MB.
- 2026-11-01 – The api container has no healthcheck (distroless has no shell); Caddy waits for it to start, not to be healthy. *(superseded, see Review follow-ups)*

## Frontend

- 2026-11-01 – The plot layer shows at zoom ≥ 10 only while the padded viewport holds ≤ 8000 plots (about zoom 10.3 at 1440×900); otherwise the coverage layer shows.
- 2026-11-01 – `GET /api/plots` is split into batches of ≤ 2000 cells and `/api/coverage` into batches of ≤ 500; the contract is unchanged.
- 2026-11-01 – The viewport is padded 10 % per side and clipped to a Thailand box (97.0–106.0 E, 5.4–20.8 N) before computing cells.
- 2026-11-01 – Coverage uses res 6 from zoom 7 and falls back to res 5 when res 6 would exceed 8000 cells.
- 2026-11-01 – Parent cells with nothing surveyed get no fill; surveyed parents get at least 0.05 opacity plus an outline, so a single photo stays visible.
- 2026-11-01 – Map position lives in the URL hash, so a reload keeps the view.
- 2026-11-01 – A floating "Upload photo" button works without selecting a plot; the plot always comes from the photo's location.
- 2026-11-01 – `lat`/`lng` are only sent after "Use my location"; the clicked plot's centre is never used as a fallback.
- 2026-11-01 – `web/style.json` is bundled with `import.meta.glob`, and `pmtiles:///…` URLs are rewritten to the page origin at runtime.
- 2026-11-01 – The client blocks HEIC and files over 25 MB before upload, mirroring the backend.
- 2026-11-01 – A 404 from `GET /api/plots/{cell}` is shown as a blank plot.

## Review follow-ups

- 2026-11-01 – Photo storage runs on Garage (`dxflrs/garage:v2.5.0`, single node, config in `deploy/garage.toml`) instead of a MinIO community image; `quay.io/minio/minio` now requires authentication (401). The api code is unchanged: only `internal/storage` speaks S3.
- 2026-11-01 – `garage-init` (`curlimages/curl:8.22.0` + `deploy/garage-init.sh`) sets the node layout, creates the bucket and imports the api's key through Garage's admin API v2; Garage secrets come from `GARAGE_RPC_SECRET` / `GARAGE_ADMIN_TOKEN`, never from the config file.
- 2026-11-01 – Garage has no anonymous-read bucket policy, so the bucket gets website access and Caddy proxies `/img/photos/*` to Garage's website endpoint (Host `<bucket>.web.garage.localhost`); other paths under `/img` return 404, non-GET/HEAD 405.
- 2026-11-01 – `Cache-Control` for photos comes from the stored object only; Caddy no longer adds its own header.
- 2026-11-01 – Garage requires its key format: `S3_ACCESS_KEY` is `GK` + 24 hex characters, `S3_SECRET_KEY` 64 hex characters.
- 2026-11-01 – Only host port 8080 is published; Garage has no console.
- 2026-11-01 – PostgreSQL is `postgres:16.15-alpine3.24`; PostGIS was optional and ran emulated on Apple Silicon. Existing local volumes from the PostGIS image should be reset with `docker compose down -v`.
- 2026-11-01 – `api -healthcheck` probes `/healthz` on `API_ADDR` and exits 0/1; compose uses it as the api's healthcheck and Caddy waits for the api to be healthy.
- 2026-11-01 – `GET /api/plots/{cell}` returns `claimed_at`; the frontend no longer derives it from the photo list.
- 2026-11-01 – `GET /api/coverage` returns only parents with at least one surveyed plot.
- 2026-11-01 – Map glyphs are self-hosted in `web/public/fonts/` (Noto Sans Regular and Medium, all 256 ranges, OFL) and the style loads `/fonts/{fontstack}/{range}.pbf`.
- 2026-11-01 – The logo lives in `web/public/logo.svg` (favicon and header); `apple-touch-icon.png` (180 px) is rendered from it with square corners because iOS masks the icon itself.
- 2026-11-01 – On desktop the open side panel no longer covers the map controls: zoom buttons and the OSM attribution move left by the panel width while it is open.
- 2026-11-01 – The brief's API contract now lists `claimed_at` in `GET /api/plots/{cell}` and states that `/api/coverage` returns only parents with at least one surveyed plot.
- 2026-11-01 – `CLAUDE.md` links `docs/PRINCIPLES.md` under a "Principles" line.
- 2026-11-01 – While the window is being resized, the panel and map controls have no transitions (body class `resizing`, cleared 150 ms after the last resize event), so the closed panel does not fly across the map when the layout crosses the 768 px breakpoint.

## Home

- 2026-11-01 – Work happens on the `home` branch; tasks 1–3 run on `home-data`, `home-accounts` and `home-pages`, cut from `home` after `internal/views` and the schema are in, and are merged back by the lead.
- 2026-11-01 – Task 1 owns every migration of the Home brief (accounts, sessions, places, regions, featured, medium variant), so migration numbers never clash between parallel branches.
- 2026-11-01 – OSM data is read in Go with `github.com/paulmach/osm` and geometry is done with `github.com/paulmach/orb` (approved by Adrian), not with osmium in Docker.
- 2026-11-01 – Photos are licensed CC BY-SA 4.0, retroactively including the prototype photos (Adrian's choice: share-alike keeps the archive open and matches Geograph, OSM and Wikimedia Commons).
- 2026-11-01 – Automated tests use small synthetic JPEGs with GPS tags, committed to the repo; the acceptance run and screenshots use Adrian's own photos in `testdata/`, released under CC BY-SA 4.0.
- 2026-11-01 – Footer contact: hello@cartograph.asia (mailbox still to be set up), X @cartographasia, Instagram @cartograph.asia, GitHub cartograph-asia.
- 2026-11-01 – In coastal provinces a res-8 cell whose centre lies inside an OSM water polygon (`water-polygons-split-3857`, already used for the map) does not count as land.
- 2026-11-01 – Region names and slugs come from `name:en`; the Thai name is shown as a second name.
- 2026-11-01 – A placeholder account is claimed by setting its e-mail in SQL; after that the normal magic link works.
- 2026-11-01 – Prototype surveyor names that do not fit 3–30 characters of letters, digits and hyphens are slugified, with a numeric suffix on collision.
- 2026-11-01 – Before the prototype stack was reset, its database (`pg_dump -Fc`) and Garage volumes (tar) were saved to the git-ignored `backup/`.
- 2026-11-01 – `internal/views` carries photos as object keys (`ImageKeys{Thumb, Medium, Large}`, `Best()` = medium else large) and IDs as strings, so it imports only the standard library.
- 2026-11-01 – Every page struct embeds `views.Page` (Viewer, CSRF, Now, Title, Canonical, Description, Feed, NoIndex, OG); `Page.Base()` lets one render helper set Viewer/CSRF/Now on any page. `Now` is carried so relative times are deterministic in fixtures.
- 2026-11-01 – Two list pages beyond the brief, `/surveyors` and `/regions`, give the front-page counters a target; "plots surveyed" and "photos" link to `/latest`.
- 2026-11-01 – `views.Source` methods treat page < 1 as 1 and return `ErrNotFound` for a page past the last one; name and slug lookups are case-insensitive and return the canonical spelling for a redirect.
- 2026-11-01 – `/plot/{cell}` renders a blank plot (noindex) for a valid res-8 cell inside a region with no photo; other cells are 404.
- 2026-11-01 – "New Surveyors" lists only Surveyors with at least one photo; the Surveyors counter counts all accounts, placeholders included.
- 2026-11-01 – The fixture (`internal/views/viewstest`) is a fixed 30-photo, 5-surveyor slice of Pattaya with real res-8 cells and `viewstest.Fake`, an in-memory `views.Source`, for task 3.
- 2026-11-01 – `regions` has an identity `id`, `osm_id`, `admin_level`, `country` and `parent_id` so countries can join later; the boundary is simplified GeoJSON text with a bbox for prefiltering; `land_cells`, `claimed_cells` and `last_surveyed_at` are stored and updated on claim.
- 2026-11-01 – A plot's place and region names are copied into `plots.place_name` / `place_name_th` on first claim (plus `region_id`), so `places` can be re-imported without renaming plots.
- 2026-11-01 – Surveyor name uniqueness is a unique index on `lower(name)`, e-mail uniqueness on `lower(email)` (nulls allowed, for placeholders); the backend stores e-mails lower-cased.
- 2026-11-01 – Magic-link tokens and sessions store SHA-256 hashes (`bytea`, 32 bytes); a session row exists before a name is chosen (`surveyor_id` null, `email` set), so no nameless surveyor rows are needed.
- 2026-11-01 – `photos` gains `width`/`height` of the large variant (for `<img>` sizes and og:image) and `licence` (default `CC BY-SA 4.0`); both are null/default for prototype photos.
- 2026-11-01 – `featured` uses `starts_at`/`ends_at` (null = until replaced) and `body` for the text; the current entry is the latest started, not ended one.
- 2026-11-01 – No comment hook in the schema: comments need their own table, which the brief puts out of scope.
- 2026-11-01 – OSM reading lives in `internal/geo/osmfile`, apart from `internal/geo`: with cgo on (h3), `paulmach/osm/osmpbf` links the system zlib through `github.com/DataDog/czlib` (transitive, with protobuf and protoscan). Only `cmd/regions`, `cmd/places` and `cmd/fixture` link it; their image build needs `zlib1g-dev` + `pkg-config` and their runtime `libz.so.1`, the api binary needs neither.
- 2026-11-01 – Provinces are the relations with `boundary=administrative`, `admin_level=4` and an `ISO3166-2` tag starting `TH-`; names drop " Province" / "จังหวัด", so slugs are e.g. `chon-buri`, `phra-nakhon-si-ayutthaya`.
- 2026-11-01 – Boundaries are stored simplified with Douglas-Peucker at 0.0005° (~55 m); the same geometry drives plot lookup and `land_cells`, so coverage never exceeds 100 %.
- 2026-11-01 – A plot's region is the province containing the cell's centre, else the nearest province boundary within 0.01° (~1 km, covers slivers between simplified neighbours and beaches); a land cell counts for a province only when that lookup returns it, so no cell counts twice.
- 2026-11-01 – A photo whose plot is outside every province is rejected with 400 "outside the area Cartograph covers so far (Thailand)", also when no regions are imported; the stored variants are logged as orphaned.
- 2026-11-01 – The api keeps the province index in memory (`db.ReloadRegions`); it loads lazily and reloads at most once a minute when a point is in no region, so an import is picked up without restart. Countries and cities are not in the index.
- 2026-11-01 – The water shapefile reader streams the `.shp` inside the zip (no `.dbf`/`.shx`), keeps records whose box meets the area and applies the even-odd rule over all rings of a record.
- 2026-11-01 – Places are OSM nodes with `place=city|town|village` and a Latin-script name (`name:en`, `int_name`, `name:th-Latn`, `name`); Thai-only places (about 1,050 villages) are skipped so English text never shows Thai script. `cmd/places` replaces the table (within `-bbox` if given).
- 2026-11-01 – The nearest place is measured from the plot's centre with an equirectangular distance (≤ 5 km), plain nearest regardless of kind, ties by id; plots copy name and Thai name, or the province's names when nothing is within 5 km.
- 2026-11-01 – `cmd/regions` and `cmd/places` backfill plots that lack a region or place name and recompute `claimed_cells`/`last_surveyed_at`; place names are filled once and never replaced.
- 2026-11-01 – Full Thailand import, measured once on an M-series Mac: `regions` 14.4 s, 490 MB peak RSS (77 provinces, 3,030 ways, 824,429 nodes, 41 water polygons read in 7.1 s, 598,404 land cells, 3.0 MB of boundary GeoJSON); `places` 3.3 s, 121 MB (18,145 places).
- 2026-11-01 – The fixture is `testdata/pattaya.osm.gz` (complete Chon Buri relation + 5 places, 170 KB) and `testdata/pattaya-water.shp` (water clipped to the box, 13 KB) for the box `100.82,12.82,100.98,13.02`, cut by `go run ./cmd/fixture`; with `-bbox`, `land_cells` counts only cells inside the box.
- 2026-11-01 – `views.Source` is implemented by `db.Views` (`d.Views()`), not `*db.DB` itself, because `*db.DB` already has the JSON API's `Plot` method.
- 2026-11-01 – Photos without an account (prototype, before the placeholder migration) show and group under their free-text surveyor name; "most active" ranks by photos uploaded, ties by name; a region's plot counter is `claimed_cells`.
- 2026-11-01 – The thumb variant is now resized from medium (large → medium → thumb); the DB stores the large variant's width and height, medium's size is derived.
- 2026-11-01 – `cmd/seed` creates five accounts `<name>@example.test`, 30 synthetic 1600×1200 JPEGs (drawn at run time, EXIF GPS + DateTimeOriginal) over 18 Pattaya plots spread across the last 44 days, and one featured row only if `featured` is empty; ids are UUIDv5, so a second run creates nothing.
