# Restoring from a backup

The production backups reach the Mac when it asks for them (`cartograph-pull-backups`, from `deploy/pull-backups.sh`): encrypted database dumps in `<backup dir>/db/cartograph-YYYYMMDD-HHMMSS.dump.gpg` and a mirror of the photo bucket in `<backup dir>/photos/`. This is how to bring them back, into a fresh stack on the Mac (the restore test) or onto a new host.

## 1. Decrypt the dump

The key lives in its own keyring, its passphrase in the login keychain:

```sh
export GNUPGHOME=~/.config/cartograph/gnupg
gpg --batch --pinentry-mode loopback \
  --passphrase "$(security find-generic-password -s cartograph-backup -w)" \
  --decrypt <backup dir>/db/cartograph-YYYYMMDD-HHMMSS.dump.gpg > /tmp/cartograph.dump
```

Without the Mac, there is no way to read the dumps: keep a copy of `~/.config/cartograph/gnupg` and the passphrase somewhere safe (a password manager).

## 2. A fresh stack

On the Mac, from a checkout of the release that made the backup:

```sh
cd deploy
docker compose -p cgrestore up -d postgres garage garage-init
```

(On a new host, the same with `deploy/prod.sh` and the production `.env`.)

## 3. The database

```sh
docker compose -p cgrestore exec -T postgres pg_restore -U cartograph -d cartograph --clean --if-exists < /tmp/cartograph.dump
rm /tmp/cartograph.dump
```

## 4. The photos

Copy the mirror back into the bucket with the api's key, through Garage's S3 port inside the stack's network:

```sh
docker run --rm --network cgrestore_default -v "<backup dir>/photos:/data:ro" \
  -e RCLONE_CONFIG_G_TYPE=s3 -e RCLONE_CONFIG_G_PROVIDER=Other \
  -e RCLONE_CONFIG_G_ENDPOINT=http://garage:3900 -e RCLONE_CONFIG_G_REGION=us-east-1 \
  -e RCLONE_CONFIG_G_ACCESS_KEY_ID=<S3_ACCESS_KEY> -e RCLONE_CONFIG_G_SECRET_ACCESS_KEY=<S3_SECRET_KEY> \
  -e RCLONE_CONFIG_G_FORCE_PATH_STYLE=true \
  rclone/rclone:1.71.2 copy /data g:cartograph
```

The mirror keeps photos whose files an admin removed; after a restore, remove those objects again (their keys are in the `photo.remove_files` rows of `admin_actions`).

## 5. Start and check

```sh
docker compose -p cgrestore up -d
```

The front page, a plot page, a photo and `/admin` (signed in as an admin) must show the same as production on the day of the dump.
