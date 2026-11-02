# Deploying cartograph.asia

How the production stack is set up, released, updated and rolled back (`docs/DEPLOY_BRIEF.md` explains why). Host-specific values (the host's name, paths, the local port) are not in this public repository; they live in the git-ignored `deploy/host.local.md` and, on the host, in `cartograph-deploy.env` next to the bare repository and in `deploy/.env`.

Below, `HOST` is the ssh name of the production host, `RUNTIME` the directory the stack runs from (default `/srv/cartograph`), `REPO` the bare repository on the host.

## Setting up the host (once)

1. **Bare repository and hook**

   ```sh
   ssh HOST 'git init --bare REPO'
   scp deploy/post-receive HOST:REPO/hooks/post-receive
   ssh HOST 'chmod +x REPO/hooks/post-receive'
   ```

   If the stack does not run from `/srv/cartograph`, write `REPO/cartograph-deploy.env` on the host with `CARTOGRAPH_RUNTIME=…` (and, if 18070 is taken, `CARTOGRAPH_PROBE_PORT=…`).

2. **Settings**: copy `deploy/.env.prod.example` to `RUNTIME/deploy/.env` on the host, fill in every value (fresh secrets, the mailbox password, the local port), `chmod 600`.

3. **Base map and import data**: `map/publish.sh HOST RUNTIME` copies the archive and prints the `TILES_URL` line for `.env`. For the imports, copy each country's extract to `RUNTIME/map/data/` (same layout as `map/data/`): `thailand-latest.osm.pbf`, `laos-latest.osm.pbf` and so on, named as in `Extract` of `internal/countries/import*.go`, plus `map/data/sources/water-polygons-split-3857.zip` to `RUNTIME/map/data/sources/`. A large extract can be cut down on the Mac first (`map/cut.sh <extract>`, which writes `map/data/scan/<extract>-latest.osm.pbf.cut.osm.pbf`); copy the cut to the same place under `RUNTIME/map/data/scan/` and the importers read it instead of the full extract when it is newer. Nothing about the import goes into `.env` (`IMPORT_COUNTRIES` is only the default for a run without arguments); `IMPORT_PBF`, `IMPORT_WATER` and `IMPORT_BBOX` are for the development fixture and are ignored on the host.

4. **Remote and first release**: `git remote add deploy HOST:REPO`, then `git push deploy main`. The hook builds, starts the database and the photo store, probes the api and starts the stack.

5. **Import Thailand** (once, about half an hour). Running imports on production is Adrian's step, by hand; no agent or hook does it. First a dry run, which only reads and changes nothing:

   ```sh
   ssh HOST 'RUNTIME/deploy/prod.sh --profile tools run --rm import -countries th -dry-run'
   ```

   Read its report (provinces, districts, slugs, land plots, new plot numbers, border cells skipped, places) and the checks; a real run refuses if one fails. Then the same without `-dry-run`:

   ```sh
   ssh HOST 'RUNTIME/deploy/prod.sh --profile tools run --rm import -countries th'
   ```

6. **Host Caddy**: `deploy/cartograph.asia.caddy` to `/etc/caddy/sites/`, then `caddy reload` (see the file's header).

7. **Backups**: on the Mac, once: `deploy/install-backup-mac.sh /Volumes/<disk>/backups/cartograph HOST`. It creates the backup key (passphrase in the login keychain), copies its public half to the host and installs `cartograph-pull-backups` in `~/.local/bin`. Nothing is scheduled, on the host or the Mac.

## Importing another country

One country per run, each after the previous one's summary has been read, in the decided order: Laos (`la`), Cambodia (`kh`), Vietnam (`vn`), Timor-Leste (`tl`), Malaysia, Brunei and Singapore (`my`, `bn`, `sg`), the Philippines (`ph`), South Korea (`kr`), North Korea (`kp`), Taiwan (`tw`), Japan (`jp`), Myanmar (`mm`), Indonesia (`id`), Mongolia (`mn`) and China (`cn`). A country is importable once its settings no longer carry `Hold` in the release that is running. Running the imports on production is Adrian's step, by hand (an agent prepares the code and the settings, never the run).

1. Copy the country's extract (`RUNTIME/map/data/<extract>-latest.osm.pbf`, for example `laos-latest.osm.pbf`; Malaysia, Brunei and Singapore share `malaysia-singapore-brunei`) or its cut (`RUNTIME/map/data/scan/<extract>-latest.osm.pbf.cut.osm.pbf`, from `map/cut.sh <extract>`) to the host. Use the extract of the day the scan report was made, or expect the dry run to report the difference.
2. A dry run, which reads only and does not migrate (a pending migration makes it refuse until the release is up):

   ```sh
   ssh HOST 'RUNTIME/deploy/prod.sh --profile tools run --rm import -countries la -dry-run'
   ```

   Read the report and the checks: province count, slugs, land plots against the settings' figure, provinces without land plots, relations that do not assemble, border cells held by another country (1 to 3 per border is expected), places added and skipped. Nothing is written.
3. The same without `-dry-run`:

   ```sh
   ssh HOST 'RUNTIME/deploy/prod.sh --profile tools run --rm import -countries la'
   ```

   It repeats the checks, imports the regions, then the places, and prints a summary row for the country. A run that fails part-way is completed by running it again (numbers only append).
4. Look at the country page and the api's memory afterwards (`RUNTIME/deploy/prod.sh ps`, `docker stats --no-stream`), and note time and memory in `docs/decisions.md`.

Several countries can be named in one run (`-countries la,kh`); a failed country is logged, the next still runs and the exit code is 1. Production goes one at a time. `-pbf`, `-water` and `-bbox` are allowed only with a single country and are not needed on the host.

## Taking a backup

```sh
cartograph-pull-backups
```

The Mac runs `deploy/backup.sh` on the host over ssh (a fresh encrypted dump, the photo mirror brought up to date) and copies both to the backup disk. Every dump pulled stays there; delete old ones by hand. The host keeps its dumps for 3 days.

## Releasing

```sh
git push deploy main
```

The hook prints its five steps. If the probe fails, it prints the probe's log and the running version stays. While the stack switches, the site answers for a few seconds with the host Caddy's "coming soon" page.

## Looking at the stack

```sh
ssh HOST 'RUNTIME/deploy/prod.sh ps'
ssh HOST 'RUNTIME/deploy/prod.sh logs --tail 100 api'
```

## Updating the base map

Build on the Mac (`map/build.sh`), then `map/publish.sh HOST RUNTIME`, put the printed `TILES_URL` into `RUNTIME/deploy/.env`, run `RUNTIME/deploy/prod.sh up -d api`, and delete the previous archive in `RUNTIME/map/` right away.

## Rolling back

Push the previous commit: `git push -f deploy <commit>:main`. Migrations are never destructive (CLAUDE.md), so an older api runs on a newer schema. To skip the build, set `API_TAG`/`WEB_TAG` in `RUNTIME/deploy/release` to an image still on the host (`docker images cartograph-api`) and run `RUNTIME/deploy/prod.sh up -d`.

## Going live

**State on 2026-11-02:** offline. The site was live for about an hour that day and then taken offline again (Adrian): the stack runs on the host on a private local port (`HTTP_PORT`, not the one the host Caddy proxies to), and the host Caddy answers `https://cartograph.asia` with its "coming soon" page. Going live again is step 3 below; the legal pages stay drafts until step 1 is done.

Until the switch, the stack can run on another local port than the one the host Caddy proxies to, and be checked through an ssh tunnel. To go live:

1. The legal pages (see below; on 2026-11-02 the site went live without them).
2. `SMTP_PASSWORD` in `RUNTIME/deploy/.env`; the api's log says `smtp ok` at start.
3. `HTTP_PORT` in `.env` to the port in `deploy/cartograph.asia.caddy`, then `RUNTIME/deploy/prod.sh up -d`.
4. Check `https://cartograph.asia`: the front page, `/map`, a sign-in mail, an upload, `/robots.txt`.

## Legal notice and privacy

On 2026-11-02 the site went live without them (Adrian), and was taken offline again the same day. The drafts wait in `docs/drafts/legal-notice.md` and `docs/drafts/privacy.md`, outside the embedded pages: they are not served, not linked, and `/legal-notice`, `/privacy` and `/impressum` answer 404 (a test holds this). To publish them:

1. Fill in the address, the supervisory authority and the date; nothing in brackets left, the DRAFT line removed.
2. Undo the commit that took them offline (`git log --oneline -- docs/drafts`) with `git revert --no-commit <commit>`, but keep `docs/decisions.md` as it is (`git checkout HEAD -- docs/decisions.md`; the log only grows). That brings back the files in `docs/pages/`, the routes, the `/impressum` redirect, the footer and map links and their test; then move the filled-in text over and update this section.
3. Release.
