# Brief: Deployment – the alpha on cartograph.asia

**Status: draft by Claude for Adrian, 2026-11-01.** The open questions at the end decide parts of it; each has a recommendation. When they are answered, this note goes and the brief is final.

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

- `deploy/cartograph.asia.caddy`, a site file for the host Caddy:
  - `cartograph.asia { reverse_proxy 127.0.0.1:{port} }`; `www.cartograph.asia` redirects permanently.
  - No `encode` for `/tiles/*` (the archive is read in byte ranges); `encode zstd gzip` for the rest.
  - The security headers (section 2) and the access log (rolled daily, kept 14 days, open question 6) live here, because the host Caddy is what the browser talks to.
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
- **Memory** for a shared host: api 1 GB with `GOMEMLIMIT` 768 MiB and one image processed at a time (open question 8), Postgres 512 MB, Caddy 128 MB, web 64 MB; no Garage (photos go to Object Storage, section 4).
- Images are pinned by digest.

### 4. Photos in Hetzner Object Storage

Photos leave the host (open question 2): one bucket, a key that may only use this bucket (read and write, which covers the admin's file removal), no admin or root keys on the server. Public read only for `photos/*` (`s3:GetObject`), no listing, no public write; `IMG_BASE_URL` is the bucket's https URL. Objects keep `Cache-Control: public, max-age=31536000, immutable`. The `X-Robots-Tag` of the crawler block cannot come from Caddy for these files; the implementation checks whether the bucket can send it and documents the result (decisions.md, Security). This also keeps the host's disk for the database and the base map.

### 5. The base map on the server

The archive is built on the Mac only (decisions.md, Map) and copied with `rsync --partial` into the project's `map/` directory under a dated name (`asia-2026-11-01.pmtiles`), so that browsers never mix ranges of two builds; switching is one line, `TILES_URL=/tiles/asia-2026-11-01.pmtiles`, and an api restart. A script `map/publish.sh` does the copy and prints that line. The world archive is 10.1 GB; a switch needs the old and the new file side by side for a moment (open question 9).

### 6. Deploy like the other sites

A bare repository on the host with `deploy/post-receive` versioned in this repository: on a push to `main` it builds the api and web images tagged with the commit, starts a probe of the api on a test port (`/healthz` must answer and `/` must render against the real database), switches the compose stack only then, and keeps the last three images. Migrations run at api start; they are never destructive (CLAUDE.md), so a rollback is a push of the previous commit. `docs/DEPLOY.md` lists the steps for a release, a base-map update and a rollback, without host-specific paths (those are in `deploy/host.local.md`).

### 7. Backups, pulled to Adrian's Mac

"Data is kept" (CLAUDE.md) needs backups that survive the host, before the first invited Surveyor uploads. **Decided (Adrian): the backups go from the VPS to Adrian's Mac**, and the Mac fetches them: the server holds no credentials for the Mac, so a compromised server cannot reach or delete the backups.

- **On the server**, `deploy/backup.sh` from a crontab writes a nightly `pg_dump -Fc` of the whole database into one directory, encrypted with `age` to a public key whose private half stays on the Mac, and keeps only the last 3 nights there.
- **On the Mac**, `deploy/pull-backups.sh` run by a launchd job (`deploy/asia.cartograph.backup.plist`, daily, and again when the Mac wakes) pulls the new dumps with `rsync` over SSH into a backup directory on the external SSD (path in `.env.local`, not in git), keeps 14 daily, 8 weekly and 12 monthly, and logs to `~/Library/Logs/`.
- **Photos**: versioning on the Object Storage bucket, plus `rclone sync` of the bucket to the same SSD from the same launchd job (a read-only key for the bucket, kept in the Mac's keychain).
- **When the Mac is off** for days, nothing is lost on the server side for 3 nights; the pull reports in its log and with a macOS notification when the newest dump on the Mac is older than 2 days.
- **Restore test**: the brief is done only after last night's dump and the photos were restored into a fresh stack on the Mac and the site came up with the same plots and photos. `docs/RESTORE.md` holds the steps.

### 8. DNS and mail

- `A`/`AAAA` for `cartograph.asia` and `www`, a `CAA` record for Let's Encrypt.
- SPF, DKIM (mailbox.org's records) and DMARC (`p=quarantine`, reports to Adrian) for `cartograph.asia`, so that sign-in and invitation mails arrive and cannot be forged.
- A start-up check that SMTP answers (connect and `EHLO`, no mail sent), and a log line naming the sender, never the password.

### 9. Watching it

An external uptime check of `https://cartograph.asia/healthz` mailing Adrian, `docker compose logs` on the host, and the admin overview. Whether page views go to a web analytics tool is open question 7.

## Out of scope

A CDN, a registry and CI-built images, metrics, a second server, the forum, plots beyond Thailand.

## Done means

`https://cartograph.asia` answers through the host's Caddy with a valid certificate and the headers above (checked with `curl -I` and a header scanner); an invited test account signs in with a real mail and uploads a photo that appears with an https URL from Object Storage; `/robots.txt` blocks; a push to the host's repository deploys through the probe, and a broken commit leaves the running version untouched; a restore of last night's dump and the photos into a fresh stack on the Mac shows the same site; `docker compose … up` without an `.env` fails with a clear message; the other sites on the host are unaffected; `/security-review` on the branch; `docs/DEPLOY.md` and `docs/RESTORE.md` exist; no host-specific detail is committed.

## Open questions for Adrian

1. **Cartograph as one more site behind the host Caddy, on a local port, deployed by a push to a bare repository with a probing hook?** *Recommendation: yes*, the same way as the other sites on the host (port proposal in `deploy/host.local.md`).
2. **Photos in Hetzner Object Storage, not Garage on the host?** *Recommendation: Object Storage.* The host's memory and disk are tight for another storage service, and photos off the host make backups simpler.
3. ~~Where do backups go off the host?~~ **Decided (Adrian): to the Mac**, pulled by the Mac (section 7). The other projects on the host could use the same pull job later.
4. **GitHub stays the main remote, the host gets a second remote for deploys?** *Recommendation: yes.*
5. **Impressum and Datenschutzerklärung before the domain goes live?** Adrian runs the site from Germany, and even an invitation-only site is publicly reachable. *Recommendation: yes, two static pages (`/impressum`, `/privacy`)* linked from the footer; the privacy page names what is stored (e-mail, sessions, photos with position, hashed sign-in refusals, access logs with IPs for 14 days) and how to ask for deletion. Text by Adrian; Claude can draft.
6. **How long are access logs kept?** *Recommendation: 14 days*, then deleted; they hold IP addresses.
7. **Web analytics for Cartograph?** *Recommendation: none in alpha*; with a handful of invited Surveyors it says little, and the privacy page stays shorter. Reconsider for the beta.
8. **One photo processed at a time?** On a shared host, *recommendation: one at a time and an api limit of 1 GB in alpha*; a second upload waits a few seconds instead of risking memory for the other sites.
9. **Disk for the base map?** The world archive takes 10 GB, about 20 GB while switching. *Recommendation: fine for alpha*, with old archives deleted right after a switch; if the disk gets tight, the archive moves to Object Storage or a CDN (`TILES_URL` makes that one line).
