# Brief: Development profiles with free photos across Thailand

Read `CLAUDE.md` (Alpha: seeds and fixtures are for development only, a production stack never seeds) and `docs/PRINCIPLES.md` first. This brief gives a development database several Surveyor profiles with 10–20 real, geotagged photos each, spread over all of Thailand, so pages such as the profile (`docs/PROFILE_BRIEF.md`), the region pages and the map can be looked at with believable data. The synthetic Pattaya seed stays.

## Decisions (Adrian, 2026-11-02)

1. **Source: Wikimedia Commons.** Its API (`action=query&list=geosearch`) finds files by coordinates, and every file carries its licence and author. Only **CC0, CC BY and CC BY-SA** files are used, JPEG only, at least 1600 px wide. Nothing else is downloaded.
2. **Nothing from Commons enters the repository.** Photos and the manifest live in `testdata/commons/`, which is git-ignored. They are fetched by a command, not stored.
3. **Credit stays with the author.** The manifest keeps author, licence, licence URL and the Commons page of every photo, and a `CREDITS.md` is written next to the photos. The fake Surveyors are labels for development data; the photo pages therefore show "© {fake Surveyor}", which is wrong for these photos. The data is for local development and screenshots of the dev stack only and is never loaded into a production database or shown publicly.
4. **Polite to Wikimedia.** One request at a time, a pause between requests, `maxlag`, an honest `User-Agent` naming the project's repository, files fetched once and then served from the cache directory.
5. **Positions come from Commons' coordinates.** The downloaded thumbnails carry no EXIF, so the seed writes GPS and the taken date into each JPEG (as the synthetic seed does) and runs it through the normal upload pipeline.

## What gets built

- `cmd/commons`: `go run ./cmd/commons -out testdata/commons`. It searches around about 24 places across Thailand (Bangkok, Ayutthaya, Kanchanaburi, Hua Hin, Chiang Mai, Chiang Rai, Pai, Mae Hong Son, Nan, Sukhothai, Phitsanulok, Khon Kaen, Udon Thani, Nakhon Ratchasima, Ubon, Trat, Chanthaburi, Surat Thani, Samui, Phuket, Krabi, Nakhon Si Thammarat, Songkhla and Pattaya), keeps up to four photos per place, at least 400 m apart, downloads a 2000 px version of each and writes `manifest.json` and `CREDITS.md`. Running it again downloads only what is missing.
- `cmd/seed -commons testdata/commons`: six Surveyors with their own bio, a home region and in some cases a display name; the photos are dealt out so that each Surveyor has photos in many regions (10–20 each). Uploads are spread over the last 60 days, the taken date is the one Commons gives, else a date before the upload. Ids are derived from the file name, so a second run does nothing. A photo outside every region (sea, border) is skipped with a warning. `-photos=0` skips the Pattaya seed.
- The regions and places of **all of Thailand** must be imported first (`regions` and `places` with `map/data/thailand-latest.osm.pbf` and the water polygons), otherwise most photos have no plot.

## Tests

The manifest is read and validated (licence, coordinates, file present); the dealing-out gives every Surveyor 10–20 photos and every place a photo; a JPEG with a JFIF header gets EXIF GPS and date back through `photos.ReadMeta`; the seed run twice creates nothing the second time (database test, `TEST_DATABASE_URL`).

## Open questions

- Credit on the photo page for dev data (a "Photo: {author}, {licence}" line) would need a change to the photo template for data that only exists in development; not done.
- More countries need their own anchors and imports; Thailand only for now, as the importers are.
