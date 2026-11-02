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

3. **Base map and import data**: `map/publish.sh HOST RUNTIME` copies the archive and prints the `TILES_URL` line for `.env`. For the import, copy `map/data/thailand-latest.osm.pbf` and `map/data/sources/water-polygons-split-3857.zip` to `RUNTIME/map/data/` (same layout) and set in `.env`:

   ```sh
   IMPORT_PBF=/srv/map/data/thailand-latest.osm.pbf
   IMPORT_WATER=/srv/map/data/sources/water-polygons-split-3857.zip
   IMPORT_BBOX=
   ```

4. **Remote and first release**: `git remote add deploy HOST:REPO`, then `git push deploy main`. The hook builds, starts the database and the photo store, probes the api and starts the stack.

5. **Import Thailand** (once, about half an hour):

   ```sh
   ssh HOST 'RUNTIME/deploy/prod.sh --profile tools run --rm regions && RUNTIME/deploy/prod.sh --profile tools run --rm places'
   ```

6. **Host Caddy**: `deploy/cartograph.asia.caddy` to `/etc/caddy/sites/`, then `caddy reload` (see the file's header).

7. **Backups**:
   - on the host, in the deploy user's crontab: `30 3 * * * RUNTIME/deploy/backup.sh >> $HOME/.local/state/cartograph-backup.log 2>&1`
   - on the Mac, once: `deploy/install-backup-mac.sh /Volumes/<disk>/backups/cartograph HOST`. It creates the backup key (passphrase in the login keychain), copies its public half to the host and installs the daily launchd job.

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

Until the switch, the stack can run on another local port than the one the host Caddy proxies to, and be checked through an ssh tunnel. To go live:

1. `docs/pages/legal-notice.md` and `docs/pages/privacy.md`: the address, the supervisory authority and the date filled in, nothing in brackets left. Release.
2. `SMTP_PASSWORD` in `RUNTIME/deploy/.env`; the api's log says `smtp ok` at start.
3. `HTTP_PORT` in `.env` to the port in `deploy/cartograph.asia.caddy`, then `RUNTIME/deploy/prod.sh up -d`.
4. Check `https://cartograph.asia`: the front page, `/map`, a sign-in mail, an upload, `/robots.txt`.
