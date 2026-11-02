# Brief: Country import – a toolkit that opens one country at a time

Read `CLAUDE.md`, `docs/PRINCIPLES.md`, `docs/COUNTRY_PAGES_BRIEF.md` and the decisions on regions, districts, plot numbers and place naming in `docs/decisions.md` first.

The map already shows all of Southeast and East Asia, and since PR #26 every country in `internal/countries` has a page. Only Thailand is *open*: only Thailand has provinces, plot numbers and place names, so only there can a photo be uploaded. This brief builds the tools to open the other sixteen countries, one at a time, each as its own silo, without guessing at OSM data and without touching what Thailand already has.

Adrian (2026-11-02): a toolkit that, per country, works out what OSM holds there and what rules follow from it, and runs for each country on its own.

## Where things stand

The importers were written for Thailand; four things in them are Thai and would break or do harm on a second country:

1. **`places` replaces the whole table.** `importer.Places` → `db.ReplacePlaces` deletes every place not in the extract it just read. Importing Vietnam's places would delete Thailand's 18,145. (Plots keep the names they were given on first claim, so no user data is lost, but blank Thai plots would lose their names.)
2. **Province and district levels are fixed.** `osmfile.IsProvince` takes `admin_level=4` with an `ISO3166-2` tag starting with the country code; `IsDistrict` takes every `admin_level=6`. That fits Thailand and, to be checked, a few neighbours; it is not known to fit the Philippines (regions and provinces on different levels), city states (Singapore) or Brunei (districts only); the scan finds out.
3. **Names are Thai.** `osmfile.Names` takes the first non-Thai name of `name:en`, `int_name`, `name:th-Latn`, `name` as English and `name:th` as the second name; `RegionName` and `DistrictName` strip " Province", "จังหวัด", "Khet", "Amphoe". For Vietnam, `name` ("Hà Nội") would become the English name when `name:en` is missing, and the Vietnamese name would be lost. The columns are `regions.name_th`, `places.name_th`, `plots.place_name_th`.
4. **One country per run, fixed in compose.** The `regions` and `places` one-shot services read one `IMPORT_PBF` (default the Pattaya fixture) on every `docker compose up`.

What already works for any country: the `country` column on `regions`, `place_naming` and `place_overrides` per country, plot numbers per province (keyed by cell, never reused), the region index and `SurveyableBounds` that grow with each import, and the country pages.

## The toolkit

Three stages. Each stage runs per country and its result is looked at before the next one runs. The extracts are already on disk (`map/data/<name>-latest.osm.pbf`, fetched by `map/download.sh` for the base map).

### Stage 1 – Scan (read-only)

`cmd/countryscan -country vn -pbf map/data/vietnam-latest.osm.pbf` reads an extract and writes a report, `docs/countries/<slug>.md`, plus a proposed configuration (stage 2) as a Go snippet in the report. It writes nothing to the database and imports nothing.

To keep memory small on the large extracts (China 1.6 GB, Indonesia 1.7 GB, Japan 2.6 GB), the scan first cuts the extract down to `boundary=administrative` relations with their ways and nodes and to `place=*` nodes, with `osmium tags-filter` (already used by `map/build.sh`, run the same way: on the host if installed, else in a throwaway container), and reads only that.

The report answers, with numbers and examples:

- **Administrative levels:** for each `admin_level` 2–10 inside the country: number of relations, how many carry `ISO3166-2` and with which prefix, how many have `name:en`, `int_name`, a Latin `name`, the language of `name`; ten example names per level; whether the relations tile the country without gaps or overlaps (area sum vs. country area); rings that do not close in the extract.
- **A proposal** for the province level and the district level, with the reason ("level 4: 63 relations, all with ISO3166-2 VN-xx, cover 99.8 % of the country").
- **Names:** which tags give English names, which give the country's own names; the prefixes and suffixes that repeat ("Tỉnh ", " Province", "Huyện ", " District") and would be stripped; names that would produce the same slug, within the country and against slugs already in use (regions.slug is unique across all countries).
- **Places:** counts per `place=*` value; how many have an English or Latin name; the place kinds inside the largest cities (the input for `urban_kinds`); a proposed `place_naming` row.
- **Size:** estimated land plots (area / 0.737 km², minus water), the expected `plot_numbers` rows and an estimate of time and memory for stage 3 from the Thailand measurements (598,404 land plots, 25 s, ~550 MiB).

The scan is deterministic: the same extract gives the same report, so reports can be committed and compared after an extract update.

### Stage 2 – Configuration (data, reviewed in a PR)

Each country's import settings become data next to its entry in `internal/countries`:

- the extract file name (Geofabrik name, e.g. `malaysia-singapore-brunei`),
- the province rule: `admin_level` and the ISO 3166-2 prefix (or, where there is none, the parent relation),
- the district rule: `admin_level`, or none,
- name rules: which tags give the English name and which the country's own name (with its language), the prefixes and suffixes to strip,
- the kind word for the level below the country ("province", "prefecture", "state", "region"; `regions.kind` gains the values it needs).

The `place_naming` row stays in the database as decided on 2026-11-01 and comes in a migration per country, numbered by the lead.

The importers read these settings instead of the fixed rules; Thailand's settings reproduce today's behaviour exactly, and a test holds that (Pattaya fixture: same regions, districts, slugs, names, land cells and plot numbers as before).

### Stage 3 – Import (one country, one run)

`regions -country vn` and `places -country vn` read their extract and settings from stage 2. Silo rules:

- **A run touches only its country's rows.** `places` replaces only the places of that country (by the region they lie in, or a `country` column on `places`); `regions` upserts only its country's provinces and districts. Re-running Thailand changes nothing in Vietnam and the reverse.
- **Nothing existing is rewritten.** Plot numbers are only added (as today); slugs already in use never change; plots keep their region, place name and number. A border plot already numbered by one country is not numbered again by the neighbour (the `plot_numbers` primary key is the cell; the scan reports how many such cells there are).
- **Dry run first.** `-dry-run` does everything except writing and prints what would change: provinces and districts added or updated, land plots, new plot numbers, places added, updated and removed, slugs. A real run refuses when the dry run's checks fail: province count outside the range in the settings, slug collision, land plots more than 10 % off the scan's estimate, a province without land plots.
- **One failing country does not stop the others.** The compose one-shots take a list (`IMPORT_COUNTRIES=th,vn,la`) and run each country on its own; a failure is logged and the next one runs. A production stack runs only what `.env` lists.

After an import the api picks up the new provinces by itself within a minute (`ReloadRegions`), uploads there are accepted, `/map` shows the new plots as surveyable and the country page opens. The upload error for a point outside every province names the open countries instead of "(Thailand)".

## Names in more than one language

Schema change, proposed here first (alpha: no rewrite without asking):

- Rename `name_th` to `name_local` on `regions`, `places` and `plots` (`place_name_th` → `place_name_local`), and add `name_local_lang` where it is not implied by the country. A rename keeps every value; Thailand's Thai names stay where they are. Templates get the language from the country (`lang="vi"`), where they hard-code `lang="th"` today.
- Alternative without a rename: keep `name_th` for Thai and add `name_local` for the others. Two columns for one idea; not recommended.

## Order

Small and close countries first, each one PR: first the scan for all sixteen (reports only, one PR), then the toolkit's stages 2 and 3 with Thailand reproduced and Vietnam as the first new country, then one PR per country or group of similar countries. Proposed order, to be changed by what the scan finds:

1. Vietnam, Laos, Cambodia (level 4 with ISO codes expected, small extracts),
2. Myanmar, Malaysia (states and regions; Malaysia's extract also holds Singapore and Brunei),
3. Philippines, Indonesia, Timor-Leste,
4. Taiwan, South Korea, North Korea, Japan,
5. Brunei, Singapore (no province level in the usual sense; settings decided from the scan),
6. Mongolia, China last: together about 15 of the roughly 22 million land plots of the whole list (China alone about 13 million at 0.74 km² per plot). Before they are imported, measure `plot_numbers`, the region index in the api and `/api/coverage` with the countries before them, and decide whether numbering all land plots up front still fits or whether large countries number plots on first claim only.

## Tests

- Scan: against the Pattaya fixture, the report finds level 4 = Chon Buri with `TH-20`, level 6 = its districts, the place kinds of the fixture; the report is byte-identical on a second run.
- Settings: every country in `internal/countries` has settings or is marked "not scanned"; Thailand's settings reproduce today's import on the Pattaya fixture exactly (regions, districts, slugs, names, land cells, plot numbers).
- Silo: importing a second, synthetic country into the fixture database leaves every Thai row unchanged (regions, places, plot numbers, plots); re-importing Thailand leaves the synthetic country unchanged.
- Dry run: writes nothing (row counts and a checksum of the tables before and after).
- Names: a Vietnamese and a Japanese example keep their own names in `name_local` with the right language; the vocabulary and page tests still pass.

## Out of scope

Translating the site, per-country plot of the week, per-country Surveyor ranks, cities as regions, any change to the base map build, importing anything on the production host. Running imports on production is Adrian's step, by hand, per country, after the dry run (`docs/DEPLOY.md` gets the commands).

## Open questions

1. **Name columns:** rename `name_th` to `name_local` (recommended) or add a second column?
2. **Slug collisions across countries** (a province in two countries with the same English name): append the country's slug (`{name}-{country}`)? Existing Thai slugs never change.
3. **Large countries:** number every land plot on import (as Thailand) or only on first claim for China and Mongolia? Decided after measuring with the smaller countries.
4. **Border plots:** a plot whose centre lies in two countries' simplified boundaries belongs to the country imported first (today's rule, applied across countries). Fine, or should the true boundary decide?
5. **Disputed areas:** OSM maps de facto boundaries; the scan lists overlaps between countries' relations. Who decides how they are shown?
6. **Province level for the Philippines:** its 17 regions or its roughly 80 provinces? The scan shows which OSM level holds which.
7. **Order:** is the order above right, and should China and Mongolia be opened at all in alpha?
