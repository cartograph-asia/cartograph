# Brief: Deployment – the alpha on cartograph.asia

**Status: decided by Adrian, 2026-11-01** (the decisions are at the end); ready to implement.

Read `CLAUDE.md`, the Infrastructure, Security, Map and Admin entries in `docs/decisions.md` and `deploy/` first. This brief takes the stack that runs today with `docker compose up` on a laptop and runs it on Adrian's Hetzner VPS under `https://cartograph.asia`, for the alpha: Adrian and a few invited Surveyors, data kept from the first upload on.

The concrete facts about the host (ports in use, paths, users, other sites, the state of its backups) are in `deploy/host.local.md`, which is **not in git**: this repository is public. Ask Adrian for it, or read the host yourself if you have access. This brief only names what the setup must fit.

## The host, in general terms

- A shared VPS with several other sites. It has **less memory than the current defaults assume** (single-digit GB, shared) and a disk with room for the base map, but not for much more.
- **Caddy runs on the host** and holds 80 and 443 for every site, with one site file per host, copied from each project's repository and activated with `caddy reload`.
- Every project is a compose stack whose public container binds a **port on 127.0.0.1**; databases publish no port.
- Projects are deployed by **pushes to a bare repository on the host**, whose `post-receive` hook builds the image, probes it on a test port and switches only if it is healthy.
- The Hetzner Cloud Firewall sits in front of the host; SSH is key-only without root login; unattended upgrades are on.

## What is already done in the code

- Sign-up by invitation (`SIGNUP=invite`), the admin area with moderation and file removal, the crawler block (`CRAWLERS=block`), upload pixel and memory limits, `mem_limit` per container, `GOMEMLIMIT`, the magic-link token filter in Caddy's log, the start-up refusal without `SMTP_HOST` on an https `BASE_URL`, exact photo positions kept out of public reads.
- Real mail through mailbox.org is verified. The world base map is built on the Mac (10.1 GB); the client reads its bounds from the archive's header.

## What this brief builds

### 1. Cartograph as one more site on the host

The host's Caddy does TLS and the public headers; Cartograph keeps its own small Caddy container for its internal routing (`/map`, `/tiles`, `/img`, the rest to the api), bound to `127.0.0.1:${HTTP_PORT}` (a free port, set in `.env`). That keeps `deploy/Caddyfile` the one place where Cartograph's routes live, in development and production alike.

- `deploy/cartograph.asia.caddy`, a site file for the host Caddy (decided, decision 1):
  - `cartograph.asia { reverse_proxy 127.0.0.1:{port} }`; `www.cartograph.asia` redirects permanently.
  - No `encode` for `/tiles/*` (the archive is read in byte ranges); `encode zstd gzip` for the rest.
  - The security headers (section 2) and the access log live here, because the host Caddy is what the browser talks to. The log is JSON, rolled daily and **kept 7 days** (decision 6), readable by the user that runs the Matomo import (section 9).
  - The magic-link token filter moves here too: the host Caddy's log is the one that sees `?token=`.
- `deploy/Caddyfile` keeps `auto_https off` and `:80`; in production it is reached only from the host Caddy.

### 2. Security headers

On every response of `cartograph.asia`, in the host site file:

- `Strict-Transport-Security: max-age=31536000` (no `preload` in alpha).
- A Content-Security-Policy written for what the pages actually load: own origin, the photo host (`IMG_BASE_URL`), the tile host (`TILES_URL`), MapLibre's workers (`worker-src blob:`), no inline styles (no template has one); tested against `/`, a plot page with its small map, `/map` with an upload, and the admin pages.
- `X-Frame-Options: DENY` and `frame-ancestors 'none'`, `Referrer-Policy: strict-origin-when-cross-origin` (account and admin pages keep their own `no-referrer`), a `Permissions-Policy` that allows only what the upload uses, `-Server`.

### 3. A production compose override

`deploy/compose.prod.yaml`, applied with `docker compose -f docker-compose.yml -f compose.prod.yaml`:

- Every secret and URL is required, `${VAR:?message}`, never a default: `POSTGRES_PASSWORD`/`DATABASE_URL`, the S3 keys, `BASE_URL`, `IMG_BASE_URL`, `TILES_URL`, `SMTP_*`, `MAIL_FROM`, `ADMIN_EMAILS`, `HTTP_PORT`.
- `seed`, `regions`, `places` and `fixture` get `profiles: ["tools"]`: a plain `up` never seeds or imports; the Thailand import runs once by hand with `--profile tools`.
- Only Cartograph's Caddy publishes a port, on 127.0.0.1. No other service has `ports:`.
- **Memory** for a shared host: api 1 GB with `GOMEMLIMIT` 768 MiB and **one image processed at a time** (decision 8), Postgres 512 MB, Garage 384 MB, Caddy 128 MB, web 64 MB.
- Garage keeps its data in a named volume; its admin API and RPC stay inside the compose network.
- Images are pinned by digest.

### 4. Photos in Garage on the host

**Decided (Adrian): photos stay in Garage, on the VPS**, as in development. Cartograph's Caddy keeps serving them at `/img/photos/*` from Garage's website endpoint (read-only, only `photos/`), so `IMG_BASE_URL` is `https://cartograph.asia/img`, and the crawler block's header and the security headers apply to them as to everything else.

- Fresh secrets for Garage (`GARAGE_RPC_SECRET`, `GARAGE_ADMIN_TOKEN`) and a fresh key for the bucket, required in `.env`, never the development values.
- Objects keep `Cache-Control: public, max-age=31536000, immutable`; removing a hidden photo's files deletes them from Garage.
- Photos and database share the host's disk, so the off-host backup (section 7) must include the photos.

### 5. The base map on the server

The archive is built on the Mac only (decisions.md, Map) and copied with `rsync --partial` into the project's `map/` directory under a dated name (`asia-2026-11-01.pmtiles`), so that browsers never mix ranges of two builds; switching is one line, `TILES_URL=/tiles/asia-2026-11-01.pmtiles`, and an api restart. A script `map/publish.sh` does the copy and prints that line. The world archive is 10.1 GB; a switch needs the old and the new file side by side for a moment; the old one is deleted right after (decision 9).

### 6. Deploy like the other sites

A bare repository on the host with `deploy/post-receive` versioned in this repository: on a push to `main` it builds the api and web images tagged with the commit, starts a probe of the api on a test port (`/healthz` must answer and `/` must render against the real database), switches the compose stack only then, and keeps the last three images. Migrations run at api start; they are never destructive (CLAUDE.md), so a rollback is a push of the previous commit. `docs/DEPLOY.md` lists the steps for a release, a base-map update and a rollback, without host-specific paths (those are in `deploy/host.local.md`).

### 7. Backups, pulled to Adrian's Mac

"Data is kept" (CLAUDE.md) needs backups that survive the host, before the first invited Surveyor uploads. **Decided (Adrian): the backups go from the VPS to Adrian's Mac**, and the Mac fetches them: the server holds no credentials for the Mac, so a compromised server cannot reach or delete the backups.

- **On the server**, `deploy/backup.sh` from a crontab writes a nightly `pg_dump -Fc` of the whole database into one directory, encrypted with `age` to a public key whose private half stays on the Mac, and keeps only the last 3 nights there.
- **On the Mac**, `deploy/pull-backups.sh` run by a launchd job (`deploy/asia.cartograph.backup.plist`, daily, and again when the Mac wakes) pulls the new dumps with `rsync` over SSH into a backup directory on the external SSD (path in `.env.local`, not in git), keeps 14 daily, 8 weekly and 12 monthly, and logs to `~/Library/Logs/`.
- **Photos**: the same launchd job mirrors the bucket to the SSD with `rclone` against Garage's S3 endpoint through an SSH tunnel (Garage's S3 port is never published), with a read-only key for the bucket kept in the Mac's keychain; deleted objects are kept on the Mac (`--backup-dir`), so a removal on the server is not a removal in the backup.
- **When the Mac is off** for days, nothing is lost on the server side for 3 nights; the pull reports in its log and with a macOS notification when the newest dump on the Mac is older than 2 days.
- **Restore test**: the brief is done only after last night's dump and the photos were restored into a fresh stack on the Mac and the site came up with the same plots and photos. `docs/RESTORE.md` holds the steps.

### 8. DNS and mail

- `A`/`AAAA` for `cartograph.asia` and `www`, a `CAA` record for Let's Encrypt.
- SPF, DKIM (mailbox.org's records) and DMARC (`p=quarantine`, reports to Adrian) for `cartograph.asia`, so that sign-in and invitation mails arrive and cannot be forged.
- A start-up check that SMTP answers (connect and `EHLO`, no mail sent), and a log line naming the sender, never the password.

### 9. Watching it

An external uptime check of `https://cartograph.asia/healthz` mailing Adrian, `docker compose logs` on the host, and the admin overview.

**Matomo, like the other site that uses it (decision 7):** an hourly import of the access log into the Matomo on the host, no JavaScript and no cookies on the pages. The import skips `/admin`, `/api`, `/signin*`, `/settings`, assets and tiles, and drops the `?token=` query (already redacted in the log). The privacy page says so.

## Out of scope

A CDN, a registry and CI-built images, metrics, a second server, the forum, plots beyond Thailand, Object Storage.

## Done means

`https://cartograph.asia` answers through the host's Caddy with a valid certificate and the headers above (checked with `curl -I` and a header scanner); an invited test account signs in with a real mail and uploads a photo that appears under `https://cartograph.asia/img/photos/…`; `/robots.txt` blocks; a push to the host's repository deploys through the probe, and a broken commit leaves the running version untouched; the Mac's launchd job has pulled a dump and the photos, and a restore of both into a fresh stack on the Mac shows the same site; `docker compose … up` without an `.env` fails with a clear message; the other sites on the host are unaffected; `/security-review` on the branch; `docs/DEPLOY.md` and `docs/RESTORE.md` exist; no host-specific detail is committed.

## Decisions (Adrian, 2026-11-01)

1. **One more site on the host**, behind the host Caddy on a local port (proposal in `deploy/host.local.md`), deployed by a push to a bare repository with a probing hook, like the other sites.
2. **Photos in Garage on the host**, not Object Storage (section 4).
3. **Backups go to Adrian's Mac**, pulled by the Mac (section 7).
4. **Two remotes**: GitHub and the bare repository on the host; a push to the host deploys.
5. **Impressum and privacy page before the domain goes live**: `/impressum` and `/privacy`, static Markdown pages like `/about`, linked from the footer. The privacy page names what is stored (e-mail, sessions, photos with position, hashed sign-in refusals, access logs with IPs for 7 days, the Matomo import) and how to ask for deletion. Text by Adrian; Claude drafts.
6. **Access logs are kept 7 days.**
7. **Page views go to the host's Matomo** through the log import, like the other site that uses it (section 9).
8. **One photo is processed at a time; the api gets at most 1 GB.**
9. **The base map stays on the host**; the old archive is deleted right after a switch. If the disk gets tight, it moves to a CDN later (`TILES_URL`).
