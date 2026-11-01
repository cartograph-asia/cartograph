# Brief: Deployment – the alpha on cartograph.asia

**Status: draft by Claude for Adrian, 2026-11-01.** The open questions at the end decide parts of it; each has a recommendation. When they are answered, this note goes and the brief is final.

Read `CLAUDE.md`, the Infrastructure, Security, Map and Admin entries in `docs/decisions.md` and `deploy/` first. This brief takes the stack that runs today with `docker compose up` on a laptop and runs it on Adrian's Hetzner host under `https://cartograph.asia`, for the alpha: Adrian and a few invited Surveyors, data kept from the first upload on.

## What is already done

- **Code:** sign-up by invitation (`SIGNUP=invite`), the admin area with moderation, the crawler block (`CRAWLERS=block`), upload pixel and memory limits, `mem_limit` per container, `GOMEMLIMIT`, the magic-link token filter in Caddy's log, the start-up refusal without `SMTP_HOST` on an https `BASE_URL`, exact photo positions kept out of public reads.
- **Mail:** real delivery through mailbox.org is verified (decisions.md, Home).
- **Base map:** the world archive is built on the Mac (10.1 GB); the client reads its bounds from the archive's header.

## What this brief builds

### 1. A production compose file

`deploy/docker-compose.prod.yml`, used as an override: `docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d`. It changes only what differs; everything else stays one file.

- Every secret and URL is required, `${VAR:?message}`, never a default: `POSTGRES_PASSWORD`/`DATABASE_URL`, the S3 keys, `BASE_URL`, `IMG_BASE_URL`, `TILES_URL`, `SMTP_*`, `MAIL_FROM`, `ADMIN_EMAILS`.
- `seed` (and `regions`, `places`, `fixture`) get `profiles: ["tools"]`, so a plain `up` never seeds or imports; the imports are run once by hand with `--profile tools`.
- Only Caddy publishes ports: `80`, `443` and `443/udp`. Postgres, the api, web and Garage (if it stays) have no `ports:`.
- Images are pinned by digest; the api and web images are built from a git tag (see Deploy).

### 2. Caddy for production

A `deploy/Caddyfile.prod` (or a site block chosen by an env placeholder, whichever stays readable; the brief prefers a separate file over cleverness).

- `cartograph.asia { … }` with automatic Let's Encrypt certificates and an ACME e-mail from `.env`; `www.cartograph.asia` redirects permanently.
- Security headers on every response, also `/map/*`, `/tiles/*` and `/img/*`:
  - `Strict-Transport-Security: max-age=31536000` (no `preload` in alpha),
  - a Content-Security-Policy written for what the pages actually load: own origin, the photo host (`IMG_BASE_URL`), the tile host (`TILES_URL`), MapLibre's workers (`worker-src blob:`), inline style attributes are not needed (no template has one); tested against `/`, a plot page with its small map, `/map` with upload and the admin pages,
  - `X-Frame-Options: DENY` plus `frame-ancestors 'none'`, `Referrer-Policy: strict-origin-when-cross-origin` (account pages keep their `no-referrer`), `Permissions-Policy` that allows only what the upload uses, `-Server`.
- The access log keeps the token filter, rolls daily and keeps 14 days (open question 6).
- `request_body` limits as today.

### 3. Photos

Photos move to an S3 store reachable from the internet, not through the api (open question 2 decides which). Whichever it is:

- One bucket, a key that may only use this bucket (read and write, which covers the admin's file removal), no admin or root keys on the server.
- Public read only for `photos/*` (`s3:GetObject`), no listing, no public write. `IMG_BASE_URL` is https.
- Objects keep `Cache-Control: public, max-age=31536000, immutable` (keys never change). Removing a hidden photo's files deletes the objects; a CDN in front, later, needs a purge for them (not in this brief).
- The `X-Robots-Tag` of the crawler block must come from the photo host or a CDN rule when photos no longer pass through Caddy (decisions.md, Security).

### 4. The base map on the server

The archive is built on the Mac only (decisions.md, Map) and copied to the server with `rsync --partial` into `/srv/cartograph/map/`, under a dated name (`asia-2026-11-01.pmtiles`), so that browsers never mix ranges of two builds. Caddy serves that directory at `/tiles/`; switching is one line, `TILES_URL=/tiles/asia-2026-11-01.pmtiles`, and an api restart. The old file stays until the next switch, then is deleted. A short script `map/publish.sh HOST` does the copy and prints the line to put into `.env`.

### 5. Backups and restore

"Data is kept" (CLAUDE.md) needs backups before the first invited Surveyor uploads.

- **Postgres:** a nightly `pg_dump -Fc` from a small `backup` service (cron in the container or a systemd timer on the host), encrypted with `age` to a public key whose private half is not on the server, copied off the host (open question 3). Keep 14 daily, 8 weekly, 12 monthly.
- **Photos:** bucket versioning, or a nightly `rclone sync` to a second location; the database dump alone cannot restore photos.
- **Restore test:** the brief is done only after a dump and the photos were restored into a fresh stack on the Mac and the site came up with the same plots and photos. `docs/RESTORE.md` holds the steps.

### 6. DNS and mail

- `A`/`AAAA` for `cartograph.asia` and `www`, a `CAA` record for Let's Encrypt.
- SPF, DKIM (mailbox.org's records) and DMARC (`p=quarantine`, reports to Adrian) for `cartograph.asia`, so that sign-in and invitation mails arrive and cannot be forged.
- A start-up check that SMTP answers (connect and `EHLO`, no mail sent) and a log line naming the sender, not the password.

### 7. The host

- SSH by key only, no root login, no password login; the Hetzner Cloud Firewall (it applies before Docker's iptables rules, unlike `ufw`) allows only 22, 80 and 443.
- Unattended security upgrades; Docker from Docker's repository.
- The stack lives in `/srv/cartograph` (a checkout of a tag plus `.env` with `chmod 600`).

### 8. Deploy and update

From the Mac: tag a release (`v0.1.0` …), on the host `git fetch --tags && git checkout v0.1.0 && docker compose … build && docker compose … up -d` (open question 4). Migrations run at api start as today. A short `docs/DEPLOY.md` lists the steps for a release, a base-map update and a rollback (check out the previous tag; migrations are never destructive, see CLAUDE.md).

### 9. Watching it

Minimal for alpha: an external uptime check of `https://cartograph.asia/healthz` with mail to Adrian, `docker compose logs` on the host, and the admin overview. Metrics and dashboards belong to the beta.

## Out of scope

A CDN, a registry and CI-built images, metrics, a second server, the forum, Southeast Asian plots beyond Thailand.

## Done means

`https://cartograph.asia` answers with a valid certificate and the headers above (checked with `curl -I` and a header scanner); an invited test account signs in with a real mail, uploads a photo that appears with an https image URL from the photo store; `/robots.txt` blocks; a restore of last night's dump and the photos into a fresh stack on the Mac shows the same site; `docker compose … up` without an `.env` fails with a clear message instead of starting with defaults; `/security-review` on the branch; `docs/DEPLOY.md` and `docs/RESTORE.md` exist.

## Open questions for Adrian

1. **What else runs on the Hetzner host?** If another service already holds 80/443 (a Caddy or nginx for other sites), Cartograph's Caddy cannot bind them, and it becomes a site behind that proxy instead. *Recommendation: tell me the host's specs (CPU, RAM is 16 GB incl. swap per the earlier session, disk) and what already runs there; the brief adapts.*
2. **Where do the photos live: Hetzner Object Storage, or Garage on the host?** *Recommendation: Hetzner Object Storage* (decisions.md already names it as the candidate): no storage to run, disk independent of the server, versioning available, a public-read bucket policy for `photos/*`. Garage on the host would keep photos and database on one disk, which makes the backup question harder.
3. **Where do backups go?** *Recommendation: a Hetzner Storage Box* (cheap, separate from the server, reachable by SFTP/rsync) for the encrypted Postgres dumps, plus bucket versioning for the photos. A second copy at home on the SSD weekly is good practice.
4. **Build on the host or on the Mac?** *Recommendation: build on the host from a git tag* in alpha (no registry to run; the Go and Vite builds are small). The base map stays the exception and is built on the Mac.
5. **Impressum and Datenschutzerklärung before the alpha?** Adrian runs the site from Germany, and even an invitation-only site is reachable publicly. *Recommendation: yes, two static pages (`/impressum`, `/privacy`) before the domain goes live*, linked from the footer; the privacy page names what is stored (e-mail, sessions, photos with position, the hashed sign-in refusals, server logs with IPs for 14 days) and how to ask for deletion. Text by Adrian; Claude can draft.
6. **How long are access logs kept?** *Recommendation: 14 days*, then deleted; they hold IP addresses.
7. **Uptime check: which service?** *Recommendation: a free external check (for example Hetzner's own or a hosted one) mailing Adrian*; nothing self-hosted in alpha.
