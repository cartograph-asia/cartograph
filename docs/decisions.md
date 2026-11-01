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
- 2026-11-01 – Magic links last 15 minutes and work once; `GET /signin/link?token=…` only shows a "Sign in" button that POSTs the token, so mail scanners that prefetch links cannot use it up (new page `views.PageConfirmSignIn`).
- 2026-11-01 – Sign-in always answers with the "check your e-mail" page, for known and unknown addresses alike; an unknown address becomes an account only when a name is chosen. A failed SMTP send is a 503 page, not hidden.
- 2026-11-01 – Throttle: at most 5 links per address per hour (counted in `login_tokens`, so it survives restarts; beyond that the page is the same but nothing is sent) and 20 sign-in posts per client IP per hour (in memory, IPv6 by /64, 429 page). Behind Caddy the IP is the last `X-Forwarded-For` entry when the peer is a private or loopback address.
- 2026-11-01 – Mail goes through stdlib `net/smtp` with STARTTLS (required except on localhost) and PLAIN auth when `SMTP_USER` is set; with `SMTP_HOST` empty the link is logged at info level (dev mode). New env vars: `BASE_URL`, `SMTP_HOST`, `SMTP_PORT` (default 587), `SMTP_USER`, `SMTP_PASSWORD`, `MAIL_FROM`.
- 2026-11-01 – Sessions: cookie `cg_session` (`__Host-cg_session` on https) with a random 32-byte token, HttpOnly, SameSite=Lax, Secure on https; 30 days without a visit end it, and `last_seen_at`/`expires_at` slide at most once an hour. Signing in again replaces the browser's previous session.
- 2026-11-01 – CSRF: every POST form carries the hidden field `csrf` = HMAC-SHA256 keyed by the session token (before sign-in: by a `cg_csrf` cookie that only the sign-in pages set, so plain visitors get no cookie); and every state-changing request must pass `net/http.CrossOriginProtection` (Sec-Fetch-Site/Origin). The renderer gets the token from `views.CSRFFrom`.
- 2026-11-01 – `POST /api/photos` needs a session with a chosen name (401 otherwise) and passes CrossOriginProtection instead of a token, since the map posts multipart from the same origin; the form's `surveyor` field is ignored and the Surveyor's name goes into the old text column.
- 2026-11-01 – Migration 0051 adds insert triggers that fill `photos.surveyor_id` and `plots.first_surveyor_id` from the name in the text columns when the insert leaves them null, so the upload query did not have to change.
- 2026-11-01 – Routes: `/signin`, `/signin/link`, `/signin/name` (one-time name, then `next`), `POST /signout`, `/settings` (bio and home region; saves redirect to `/settings?saved=1`), `GET /api/me`. Account pages send `Cache-Control: no-store` and `Referrer-Policy: no-referrer`.
- 2026-11-01 – `next` is accepted only as a local path (starts with one `/`, no backslash, no control characters, at most 512 bytes); anything else becomes `/`.
- 2026-11-01 – Surveyor names follow exactly `views.NamePattern`; a nameless session visiting `/signin` or `/settings` is sent to the name form. Bios are trimmed, CRLF becomes LF and control characters other than newline and tab are dropped before the 500-character check.
- 2026-11-01 – Migration 0050 slugifies prototype names in plpgsql: common Latin accents folded, other runs become one hyphen; nothing left (e.g. Thai only) becomes `surveyor`, 1–2 characters get `-1`, collisions count up from `-2`, in order of first upload. Distinct texts stay distinct accounts ("Ada" and "ada" are two placeholders).
