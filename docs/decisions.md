# Decisions

Decisions made while building the prototype that are not in `CLAUDE.md`. One line each, newest at the bottom.

## Lead agent

- 2026-11-01 – HEIC uploads are rejected with 400 (JPEG and PNG only); HEIC→JPEG needs libheif via cgo and would rule out a distroless image.
- 2026-11-01 – Work happens on the `prototype` branch with one commit per area.
- 2026-11-01 – The api image is built with `CGO_ENABLED=1` (h3-go wraps the C library) and runs on `gcr.io/distroless/cc-debian12:nonroot`, not scratch.
- 2026-11-01 – `deploy/Caddyfile` is the only Caddyfile; the map pipeline does not ship a snippet.
- 2026-11-01 – Public image URLs are `IMG_BASE_URL/<key>` with `IMG_BASE_URL=http://localhost:8080/img`; Caddy maps `/img/<key>` to `/<bucket>/<key>` on MinIO.
- 2026-11-01 – Added env vars `API_ADDR`, `S3_USE_SSL`, `S3_REGION` and `HTTP_PORT` to the brief's list.
- 2026-11-01 – Migrations are embedded into the binary from `migrations/embed.go` (`go:embed`), since the final image has no filesystem to read them from.

## Map

- 2026-11-01 – Planetiler runs from `ghcr.io/onthegomap/planetiler:0.10.2` in Docker with `-Xmx4g` (Thailand peaks at ~2.1 GB heap, ~3.5 GB container memory); pmtiles CLI is `protomaps/go-pmtiles:v1.31.2`.
- 2026-11-01 – Planetiler's auxiliary sources live in `map/data/sources/` and its temp dir in `map/data/tmp/`; all of `map/data/` and `*.pmtiles` are git-ignored.
- 2026-11-01 – `--osm-url` is set to the Geofabrik Thailand URL so a missing extract can never silently build Planetiler's default area.
- 2026-11-01 – The Planetiler container runs as the host user so output files belong to the developer.
- 2026-11-01 – The extract's md5 check is best effort with a 30 s timeout (Geofabrik's `.md5` endpoint can hang).
- 2026-11-01 – Glyphs load from the Protomaps basemaps-assets fonts on GitHub Pages (Noto Sans, no key).
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

- 2026-11-01 – MinIO runs from the community build `pgsty/minio` / `pgsty/mc`, because upstream no longer publishes container images.
- 2026-11-01 – `postgis/postgis:16-3.6` is amd64-only and runs under emulation on Apple Silicon (`platform: linux/amd64`).
- 2026-11-01 – The api uses its own bucket-scoped MinIO user (`cartograph-app`), created by `minio-init`, not the root credentials.
- 2026-11-01 – Only host ports 8080 (Caddy) and 127.0.0.1:9001 (MinIO console) are published.
- 2026-11-01 – Caddy serves only `*.pmtiles` from the `map/` mount; `/img` allows GET/HEAD only; the request body limit for `/api/*` is 30 MB.
- 2026-11-01 – The api container has no healthcheck (distroless has no shell); Caddy waits for it to start, not to be healthy.

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
