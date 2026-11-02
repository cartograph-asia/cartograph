# Brief: Country import, stages 2 and 3 – settings per country, an import that keeps each country in its silo

Read `CLAUDE.md`, `docs/PRINCIPLES.md`, `docs/COUNTRY_IMPORT_BRIEF.md` (its decisions stay valid), the country-import lines in `docs/decisions.md` and the scan reports in `docs/countries/` first. This brief implements stages 2 and 3 of `COUNTRY_IMPORT_BRIEF.md`: the import settings as Go data, and an import that opens one country at a time without touching any other country's rows.

**Before anything starts:** PR #29 (`country-import-name-local`, migration `0140_name_local.sql`, `name_th` → `name_local`) and PR #30 (`country-scan`, `cmd/countryscan`, `internal/countryscan`, the reports) are merged into `main`. Every name below assumes both (`NameLocal`, not `NameTh`).

## Scope

- **Stage 2:** an `Import` settings type in `internal/countries`, one file per country, Thailand's settings reproducing today's import exactly; the Thai name rules in `internal/geo/osmfile` become rules driven by those settings.
- **Stage 3:** `regions`, `places` and a new `import` command read the settings; the silo rules (a run touches only its country's rows, nothing stored is rewritten); a dry run with checks; per-province processing so the large countries fit in memory; compose, deploy docs, pages and the upload error follow.
- **Then:** Laos as the first new country in the toolkit PR, and one PR per country or group afterwards.

## Decisions (Adrian, 2026-11-02, "all as recommended")

1. **Order:** Laos first (in the toolkit PR), then Cambodia, then Vietnam; then Timor-Leste, Malaysia + Brunei + Singapore, Philippines, South Korea, North Korea, Taiwan, Japan, Myanmar, Indonesia, Mongolia, China.
2. **Philippines:** provinces at admin_level 4 (82 with ISO `PH` plus the Special Geographic Area through `Include`), no district level for now.
3. **Kind word:** a new nullable `regions.kind_name` holds the display word; `regions.kind` stays structural (`province`/`district`).
4. **Taiwan:** districts are levels 7 and 8 together; the settings take a list of levels.
5. **Myanmar:** provinces are the level-4 relations with ISO `MM` only (15); districts are the townships at level 7 (358).
6. **Singapore:** the five level-5 regions are provinces, recognised by lying inside the country relation r536780 (no ISO codes), the 55 level-6 planning areas are districts. **Brunei:** its four level-4 districts are provinces (word "district"), no district level.
7. **Duplicate slugs:** if stripping makes two siblings collide, both keep their affix (name and slug); if they still collide, the OSM id is appended. A stored slug never changes.
8. **China:** in scope, Xizang included, imported last. The map's fallback box widens from 88,-12,150,55 to 73,-12,154,54, in the China package, not now. China and Mongolia do not open in the alpha (`Hold`).

## Stage 2: settings

`internal/countries` stays plain data without OSM imports (the api imports it). `import.go` holds the types, `Imports map[string]Import` listing all 17 countries from the start, and `ImportOf(code) (Import, bool)`; each country has its own file `import_<code>.go` with one `var` (`importLA`, …), so a country PR edits only its file. Countries not yet opened carry `Hold`.

```go
type Import struct {
	Extract      string   // Geofabrik name: map/data/<Extract>-latest.osm.pbf ("malaysia-singapore-brunei", "east-timor")
	Province     Level
	District     Level    // zero value: no district level (Brunei, Philippines)
	English      []string // tags tried in order for name_en
	EnglishNot   []string // Unicode scripts a name_en candidate must not contain ({"Thai"})
	Local        []string // tags tried in order for name_local; plain "name" counts only if written in LocalScripts
	LocalScripts []string // {"Thai"}, {"Latin"}, {"Han", "Hiragana", "Katakana"}, …
	LocalLang    string   // BCP 47 for lang=""; empty: countries.Country.Lang
	SlugSuffix   bool     // province slug + "-" + country slug; false only for th
	PlaceKinds   []string // place=* values read; empty: today's six (Taiwan adds "hamlet")
	LandPlots    int      // the scan's figure, with report and extract date in a comment
	Partial      []int64  // province relations that may be built from their closed rings only (WP F)
	Hold         string   // non-empty: importers refuse with this reason
}

type Level struct {
	AdminLevels []int             // {4}; Taiwan districts {7, 8}
	ISOPrefix   string            // required ISO3166-2 prefix ("TH" matches "TH-20"); "" = none required
	Parent      int64             // without ISO: relation whose geometry must hold the centre (Singapore r536780)
	Include     []int64           // relations without the ISO code that count anyway (PH Special Geographic Area)
	Count       [2]int            // min, max relations; a dry-run check for provinces, reported for districts
	Kind        string            // display word: "province", "prefecture", "state", "region", "district", "commune"
	KindByAffix map[string]string // word by affix of the English or local name, matched before stripping ("市": "city", "縣": "county", "State": "state")
	Strip       Affixes
}

// Every entry is tried once, in order, suffixes first, then prefixes, trimming spaces between (as DistrictName does today).
type Affixes struct{ EnPrefix, EnSuffix, LocalPrefix, LocalSuffix []string }
```

**Thailand reproduces today's behaviour:** `Extract: "thailand"`; `Province{AdminLevels: {4}, ISOPrefix: "TH", Count: {77, 77}, Kind: "province", Strip: {EnSuffix: {" Province"}, LocalPrefix: {"จังหวัด"}}}`; `District{AdminLevels: {6}, Kind: "district", Strip: {EnSuffix: {" District"}, EnPrefix: {"Khet ", "Amphoe "}, LocalPrefix: {"เขต", "อำเภอ"}}}`; `English: {"name:en", "int_name", "name:th-Latn", "name"}`, `EnglishNot: {"Thai"}`, `Local: {"name:th", "name"}`, `LocalScripts: {"Thai"}`, `SlugSuffix: false`, `LandPlots: 598411`. These are exactly `osmfile.Names`, `RegionName`, `DistrictName` and `hasThai` on `main`.

**Name rules** (`internal/geo/osmfile/names.go`; all are no-ops for Thailand):

- `NamesFor(tags, Import) (en, local string)` and `StripName(en, local, Level)`; the old functions stay as thin wrappers over Thailand's settings until WP C has switched every caller, then go.
- *Folding:* `Slug` folds accents (NFD, marks dropped, đ → d, …) with `golang.org/x/text` (already in `go.mod`). `countryscan.foldSlug` moves into `osmfile` so scan and import cannot drift. Folding leaves ASCII unchanged, so Thai slugs stay as stored.
- *Equal names:* if `name_local` equals `name_en` after stripping, store NULL (Singapore, the Philippines, Vietnamese communes without `name:en`).
- *Kind word:* the first `KindByAffix` entry whose affix the raw English or local name carries, else `Level.Kind`. The importer writes it to `kind_name` only when it differs from `kind` (`province`/`district`), so Thailand's rows keep NULL.
- *Duplicates:* decision 7, within one parent (provinces: within the country). Example from the scan: Indonesia's two "Bogor" (kota and kabupaten) end with the OSM id; Vietnam's "Xã Tân Thành"/"Phường Tân Thành" keep their affixes.

**What stays in the database:** `place_naming` (one row per country, a migration each), `place_overrides`, the imported rows. Everything about *reading* OSM lives in the Go settings. `cmd/countryscan` takes the extract name from `ImportOf(code).Extract` instead of its own `extracts` map.

## Stage 3: import

**Regions silo**

- Only relations that match the settings are read (`AdminLevels`, `ISOPrefix` or `Parent`, `Include`). Districts are kept only when their centre lies in one of this run's provinces (as today).
- `UpsertRegions`: `on conflict (osm_id) do update … where regions.country = excluded.country`; a relation stored for another country returns no row and fails the run (the dry run checks it first). `slug` is left out of the update list; a district's slug is built from its province's *stored* slug.
- Province slug: `Slug(en)`, plus `"-" + Country.Slug` when `SlugSuffix` (`vientiane-laos`). A computed slug that equals a stored slug of another relation fails the dry run.

**Border plots**

- Land cells are decided by one `geo.RegionIndex` over all provinces in import order: the stored provinces by id, **with the run's own country's stored rows replaced in place by the run's shapes** (matched by `osm_id`), the run's new provinces appended at the end. A cell counts for the run only when the index returns one of the run's provinces. So the country imported first keeps a border plot, also when it is re-imported later, and Thailand's re-import finds its own cells.
- `AssignPlotNumbers` already skips cells that have a number; nothing changes there.
- The dry run reports cells held by another country; `overlaps.md` expects 1–3 per border.

**Places silo** (migration 0150 adds `places.country`)

- A place inside another country's province is skipped. A place inside one of the run's provinces gets the run's country. A place outside every province gets the run's country with `region_id` NULL.
- The upsert carries `where places.country = excluded.country or places.region_id is null`: a run never takes over another country's place, except one that country stored outside all provinces (a Thai extract holds Lao villages beyond the border).
- `ReplacePlaces(ctx, country, rows)` deletes only `where country = $1`.
- `PlaceKinds` comes from the settings; `osmfile.PlaceKinds` stays the default.

**Memory and input**

- Regions are upserted first; then provinces are processed in a bounded pool (`GOMAXPROCS`), each one: land cells, Hilbert sort, `AssignPlotNumbers`, `land_cells` update, cells dropped. Today every province's cells are held until the end. A failed run is completed by running it again (numbers only append).
- The importers read the osmium cut that `cmd/countryscan` leaves, `map/data/scan/<Extract>-latest.osm.pbf.cut.osm.pbf` (boundaries with their ways and nodes, place nodes), when it exists and is newer than the extract, else the full extract with a warning. `map/cut.sh <extract>` makes the same cut on a host without Go (osmium on the host, else a throwaway container, as `internal/countryscan/cut.go` does).

**Commands and flags**

- `regions` and `places` take `-country xx` (settings from `ImportOf`) and `-dry-run`. `-pbf`, `-water`, `-bbox` remain overrides (the fixture uses them); with `-bbox` the count and size checks are reported as skipped. `cmd/fixture` uses the settings' province and district rules instead of `IsProvince`/`IsDistrict`.
- New `cmd/import -countries la,kh [-dry-run] [-data /srv/map/data] [-water <zip>]` runs, per country in the order given, the checks, then regions, then places, in one process. A failure is logged and the next country still runs; a summary table (country, provinces, districts, land plots, new numbers, border cells skipped, places, result) is printed; exit code 1 if any country failed. `-pbf/-water/-bbox` are allowed only with a single country. It is Go because the api image is distroless (no shell), and one service because with two, a failed country would stop `places` for all.
- Flags are fixed here, so WP D can start before C is done.

**Dry run.** Reads only and does **not migrate** (today `connect()` migrates first): with a pending migration it refuses. It prints provinces and districts added or updated (old → new name), slugs, kind words, land plots, new plot numbers, border cells skipped, places added, updated, skipped and removed. Checks, run first by a real run as well, which refuses if any fails:

1. `Hold` set; 2. province count outside `Count`; 3. a slug that collides with a stored row of another relation, or a duplicate left after decision 7; 4. land plots more than 10 % off `LandPlots`; 5. a province without land plots; 6. a province-level relation in scope that does not assemble and is not in `Partial`; 7. a province relation stored for another country.

Reported, not enforced: districts dropped (no English name, centre outside), district cover, area lost by `Partial`, a cut older than its extract.

**Compose and deploy** (`deploy/`, `docs/DEPLOY.md`)

- One `import` service replaces `regions` and `places`: `-countries=${IMPORT_COUNTRIES:-th}`, `-data=/srv/map/data`, the fixture overrides `IMPORT_PBF`, `IMPORT_WATER`, `IMPORT_BBOX` as today (single country only). `seed` depends on `import`. In `compose.prod.yaml` `import` gets `profiles: ["tools"]`.
- `Dockerfile.api`: add `import` to the binary loop (`api regions places seed`). `.env.example` and `.env.prod.example` document `IMPORT_COUNTRIES`. `docs/DEPLOY.md`: copy each country's extract (or its cut) to `RUNTIME/map/data/`, then `prod.sh --profile tools run --rm import -countries la -dry-run`, read it, then the same without `-dry-run`; one country per run, in the decided order.

**Pages and upload error**

- `lang="th"` is hard-coded in `region.html`, `partials.html` (region-row) and `plot.html`. The language goes on `views.RegionRef.Lang` (region-row lists provinces of several countries on `/` and `/regions`) and `PlotRef.LocalLang`, from `LocalLang`, else `Country.Lang`. The CSS class `th` becomes `local` in `style.css` and the three templates.
- `Region.KindName()` uses `coalesce(kind_name, kind)` ("Province in Laos", "District in Brunei", "Region in Singapore").
- Upload error (`internal/api/photos.go`): "… covers so far (Laos and Thailand)". `db.OpenCountries`, in `internal/db/countries.go`, returns the distinct `regions.country` of provinces, cached and refreshed with `ReloadRegions`; names come from `internal/countries` in list order. With none: the sentence without the brackets.

## Migrations

| No. | Content | Package |
|---|---|---|
| 0150 | `places.country text`: backfill from the place's region, else `'th'`; then `not null`; index `(country)` | B |
| 0151 | `regions.kind_name text` (NULL: the word follows from `kind`) | B |
| 0152 | `place_naming` row (+ overrides) for Laos | G |
| 0160 kh · 0170 vn · 0180 tl · 0190 my/bn/sg · 0200 ph · 0210 kr · 0220 kp · 0230 tw (+ `hamlet` in `places_kind_check`) · 0240 jp · 0250 mm · 0260 id · 0270 mn · 0280 cn | `place_naming` rows and overrides, one block per country PR | H |

No values are added to `regions.kind`. Nothing existing is rewritten: both columns are additions, the backfill only fills the new column.

## Work packages

The toolkit is one integration branch, `country-import`, and one PR to `main` (A–E and G); each package is a session on its own branch off `country-import`, merged into it by the lead. F and every H are PRs of their own.

| WP | Content | Files | After | Parallel with | Model |
|---|---|---|---|---|---|
| A | **First commit, before B branches:** the golden test, generated by the unchanged code (`testdata/golden/pattaya-import.json`: regions, districts, slugs, names, land cells, plot numbers, places of the Pattaya fixture). Then the settings type, `import_th.go` and 16 files with `Hold` and the scan's figures, `osmfile` name/slug/kind rules, folding moved from `countryscan` (its accent-loss line compares against a non-folding slug), `cmd/countryscan` reads `Extract` | `internal/importer/golden_test.go`, `testdata/golden/`, `internal/countries/import*.go`, `internal/geo/osmfile/names.go` (+tests), wrappers in `osmfile.go`, `internal/countryscan/names.go`, `propose.go`, `cmd/countryscan/main.go` | #29, #30 | B (after A's first commit) | Opus |
| B | DB silo: 0150, 0151, upsert guard, slug kept, `kind_name` in `Region`, `ReplacePlaces(country)` with the places guard, `OpenCountries` | `migrations/0150*`, `0151*`, `internal/db/regions.go`, `internal/db/countries.go`, db tests | A's first commit | A | Opus |
| C | Importer: settings-driven reading, combined index, per-province pool, cut input, dry run and checks, `cmd/regions`, `cmd/places`, `cmd/import`, `cmd/fixture`; silo and dry-run tests; wrappers removed | `internal/importer/*`, `cmd/regions`, `cmd/places`, `cmd/import`, `cmd/fixture`, `testdata/synthetic-zz.osm` | A, B | D, E, F | Opus |
| D | Compose `import` service, env examples, Dockerfile loop, `DEPLOY.md`, `map/cut.sh` | `deploy/docker-compose.yml`, `compose.prod.yaml`, `.env.example`, `.env.prod.example`, `Dockerfile.api`, `docs/DEPLOY.md`, `map/cut.sh` | flags as fixed above | C, E, F | Sonnet |
| E | `lang` from the country, class `local`, kind word, upload error text | `internal/api/photos.go`, `internal/views/*`, `internal/db/views.go` (lang, kind word), 3 templates, `internal/web/static/style.css`, tests | A, B | C, D, F | Sonnet |
| F | Relations that do not assemble (see Risks): find out why for each of the nine, implement `Partial` (closed rings only, area lost logged) and, where a neighbour's extract holds the missing ways, building from a merged cut | `internal/geo/osmfile/osmfile.go` (`BuildBoundaries` and `ReadBoundaryData` only), `map/cut.sh` (merge), `docs/countries/` notes | A (C must not touch those two functions) | C, D, E | Opus |
| G | Laos in the toolkit PR: `import_la.go` without `Hold`, 0152, dry run against a restored copy of the production database (`docs/RESTORE.md`), import, measure time, RSS of the run and of the api afterwards, decisions lines | `internal/countries/import_la.go`, `migrations/0152*`, `docs/decisions.md` | C, D, E | – | Opus |
| H | One PR per country or group, in the decided order: re-scan, settings, migration block, dry run, measurement, report update | `import_<code>.go`, own migration block, `docs/countries/<slug>.md` | G; F first for vn, my/bn/sg, kp, tw, id, cn | each other (own files; `decisions.md` is append-only) | Sonnet: kh, tl, kr, kp, mn. Opus: vn, my/bn/sg, ph, tw, jp, mm, id, cn |

Sequence: A (first commit) → A ∥ B → C ∥ D ∥ E ∥ F → G → H. Production imports stay sequential in Adrian's order whatever order the H PRs merge in.

**Acceptance for C and G:** `import -countries th -dry-run` against a restored copy of the production database, with the extract its Thailand import used, reports **0 changes** to provinces, districts, slugs, land cells, plot numbers and places; with a newer extract, every change is explained by an OSM edit.

## Tests

- **Settings:** every code in `countries.All` has an `Imports` entry; Thailand's settings equal the old functions on a table of tag sets (Thai `name`, Thai `name:en`, missing `name:th`, "Khet …", "Amphoe …", "จังหวัด…"). The golden test passes unchanged after A, B and C.
- **Silo:** a synthetic country `zz` (`testdata/synthetic-zz.osm`: one province with `ISO3166-2=ZZ-01` beside Chon Buri, overlapping it by at least one cell, two place nodes, one of them inside Chon Buri; settings injected through `RegionOptions.Settings`, so `zz` needs no entry in `countries.All`). Importing it leaves every Thai row of `regions`, `places`, `plot_numbers` and `plots` unchanged; re-importing Thailand leaves `zz` unchanged; the shared cell keeps its Thai number; the place inside Chon Buri stays Thai.
- **Dry run:** row counts and `md5(string_agg(… order by …))` of `regions`, `places`, `plot_numbers`, `plots` and `schema_migrations` identical before and after; with a pending migration it refuses instead of migrating.
- **Names and slugs:** Vietnamese ("Tỉnh Cao Bằng"/"Cao Bằng Province" → en "Cao Bằng", slug `cao-bang-vietnam`, local NULL since equal); a commune with only `name`; Japanese (Kyōtango, 京丹後市 → `name_local`, lang `ja`); folding: Cần Thơ → `can-tho`, Đồng Tháp → `dong-thap`, Ürümqi → `urumqi`, Higashiōsaka → `higashiosaka`, Ōmihachiman → `omihachiman`; all 77 Thai province slugs unchanged; duplicates keep affixes, then the OSM id; a renamed relation keeps its stored slug; kind word by affix (`市` → city, `縣` → county).
- **Levels:** districts from two levels (Taiwan 7 + 8); provinces without ISO by `Parent` (Singapore); `Include` (Philippines); no district level (Brunei).
- **Incomplete relations (F):** a relation with a missing member way is not built and fails the dry run; listed in `Partial`, it is built from its closed rings and the lost area is logged.
- **Pages:** `lang` and the kind word per country; upload error lists the open countries, plain sentence with none; vocabulary and page tests pass.

## Order of countries

Figures from the scan reports (`docs/countries/`); stage-3 time and memory are the reports' linear estimates from Thailand (598,404 plots, 25 s, ~550 MiB), not measurements.

| # | Country (extract) | Provinces | Districts | Land plots | Estimate | Notes | Migr. |
|---:|---|---|---|---:|---|---|---|
| 1 | Laos (`laos`) | L4, LA, 18 | L6, 148, 100 % | 275,740 | 11 s, 253 MiB | strip " Province" (9 of 18), " District", local "ເມືອງ"; 1 accent name | 0152 |
| 2 | Cambodia (`cambodia`) | L4, KH, 25 | L6, 200, 92.8 % | 213,169 | 8 s, 195 MiB | local prefix "ខេត្ត"; "Kampong" is part of the name | 0160 |
| 3 | Vietnam (`vietnam`) | L4, VN, 34 | L6 communes, 3,254 | 390,593 | 16 s, 358 MiB | folding (29 + 3,020 names), Xã/Phường/Commune/Ward, Khánh Hòa r1887959 (F) | 0170 |
| 4 | Timor-Leste (`east-timor`) | L4, TL, 13 + 1 `Include` | L5, 70 | 19,546 | ~1 s | | 0180 |
| 5 | Malaysia, Brunei, Singapore (`malaysia-singapore-brunei`) | MY L4 16; BN L4 4; SG L5 5 (`Parent`) | MY L6 130; BN none; SG L6 55 | 318,281; 7,864; 941 | 13 s; ~1 s; ~1 s | Sabah r3879783 (F); one cut for three countries | 0190 |
| 6 | Philippines (`philippines`) | L4, PH, 82 + SGA | none | 367,912 | 15 s, 338 MiB | check how Metro Manila is held at level 4 | 0200 |
| 7 | South Korea (`south-korea`) | L4, KR, 15 of 16 with ISO | L6, 229 | 191,458 | 7 s | the 16th needs `Include` (Jeonnam-Gwangju is a candidate) | 0210 |
| 8 | North Korea (`north-korea`) | L4, KP, 13 | L6, 171 | 214,569 | 8 s | North Pyongan r356540 (F) | 0220 |
| 9 | Taiwan (`taiwan`) | L4, TW, 22 | L7 + L8, 328 | 51,028 | 2 s | Kaohsiung r2127079 (F); kind word from 縣/市, English city names have no suffix; hamlets | 0230 |
| 10 | Japan (`japan`) | L4, JP, 47 | L7, 1,739 | 572,035 | 23 s, 525 MiB | 27 district names need folding; 12 duplicate district slugs | 0240 |
| 11 | Myanmar (`myanmar`) | L4, MM, 15 | L7 townships, 358 | 804,966 | 33 s, 739 MiB | | 0250 |
| 12 | Indonesia (`indonesia`) | L4, ID, 38 | L5, 502 | 2,316,624 | 96 s, 2.1 GiB | Riau Islands r3797244, North Kalimantan r5449460 (F); 6.3 M cells inside the outline (sea); measure here | 0260 |
| 13 | Mongolia (`mongolia`) | L4, MN, 22 | L6, 340 | 2,225,456 | 92 s, 2.0 GiB | `Hold` until beta | 0270 |
| 14 | China (`china`) | L4, CN, 31 | open: L5 316 or L6 2,674 | 11,976,594 (too low) | 500 s, 11 GiB | Xizang r153292, Fujian r553303, Hainan r2128285 (F); `Hold`; BOUNDS change | 0280 |

## Risks

- **Incomplete extracts:** nine province relations do not assemble (table above). Likely causes, to be confirmed in F: a border whose ways the neighbouring extract holds (Sabah/North Kalimantan, North Pyongan, Fujian, Xizang) → build from a merged cut of the extracts already in `map/data/`; sea claims (Spratlys in Khánh Hòa and Kaohsiung, Hainan's Sansha) → `Partial`. If Xizang needs India, Nepal or Bhutan, those extracts are read for boundaries only and never join the base map (South Asia stays out). China's outline misses all three provinces, so its `LandPlots` must come from a corrected scan.
- **Memory and time:** the linear estimates give Indonesia 2.1 GiB and China 11 GiB; the largest Chinese province alone holds 1,958,373 land plots. Per-province processing and the cut input are what make this fit; measure after every country. Production api: `mem_limit` 1 GiB, `GOMEMLIMIT` 768 MiB; the district index grows (Vietnam 3,254, Japan 1,739, China up to 2,674), so the api's RSS is measured after every import.
- **Storage:** 20.5 M land plots in all, 14.2 M of them China and Mongolia; `plot_numbers` and the backups grow with them. Indonesia's measurement decides whether large countries number all plots up front or on first claim (original brief, question 3).
- **Extract drift:** the scan, the import and production may read different Geofabrik dates. Each country PR re-scans and updates `LandPlots`; the 10 % check catches the rest.

## Out of scope

Everything the original brief excludes (translation, per-country plot of the week and ranks, cities as regions, base-map changes, imports on production by an agent). Also: district pages; per-country plural headings ("Provinces" on country pages stays); opening China or Mongolia; the BOUNDS change before the China package (`ARCHIVE_BOUNDS` in `web/src/style.ts`, `HOME` in `web/src/main.ts`, the example box in `map/build.sh`).

## Done means

The integration PR: golden, silo and dry-run tests green with `TEST_DATABASE_URL`; `docker compose up` imports the Pattaya fixture through `import`; the Thailand dry run against the restored production copy shows 0 changes; Laos imported locally, its country page open, an upload in Laos accepted and named, Thai rows unchanged; time and RSS of the run and of the api in `docs/decisions.md`; `/security-review` run, findings fixed or listed in the PR. Open questions go into the PR description.

## Corrections to the plan

Checked against `main`, the branches `country-scan` and `country-import-name-local`, and the reports.

1. **Laos is not affix-free:** 9 of 18 province names end in " Province", all 148 districts in " District", 111 local district names start with "ເມືອງ". Laos needs strip rules.
2. **Combined index:** "stored provinces in id order, then this run's" would let a re-import of Thailand find its stored rows first and get no land cells, and would hand border plots to a later country. The run's own country's rows are replaced in place (section Border plots).
3. **Places guard:** with only `where places.country = excluded.country`, villages a Thai import stored outside every province (beyond the border) could never pass to Laos; a place with `region_id` NULL may be taken over.
4. **Kind word by suffix does not work:** Taiwan's English city names have no suffix (Keelung, Tainan) and Myanmar's regions often none (Yangon, Mandalay); Vietnam's cities differ by the local prefix "Thành phố". `KindByAffix` matches English or local prefixes and suffixes.
5. **Hamlets:** adding `hamlet` to `places_kind_check` is not enough; `osmfile.PlaceKinds` is global, so the kinds become a setting (otherwise Thailand's places change).
6. **`lang` placement:** region-row (`partials.html`) renders `RegionRef` for several countries, so the language belongs on `RegionRef`, not only `views.Region`. The class `th` goes too.
7. **Cut path:** `cmd/countryscan` writes `map/data/scan/<extract>-latest.osm.pbf.cut.osm.pbf`, per extract, not `<slug>-cut.osm.pbf`.
8. **Migration numbers** follow the decided order (Laos 0152, Cambodia 0160, Vietnam 0170, …), not the plan's (vn 0160, kh 0170). The runner applies unrecorded files in lexical order, so a late lower number would still apply, but the order is easier to read.
9. **More callers:** `cmd/fixture` uses `IsProvince`/`IsDistrict`, and `countryscan`'s accent-loss figure compares against `osmfile.Slug`, which will fold; both are in the file sets now. Removing the Thai functions in A alone would break the importer, so A leaves wrappers and C removes them.
10. **BOUNDS:** besides `ARCHIVE_BOUNDS` (`web/src/style.ts`), `HOME` in `web/src/main.ts` holds 88,-12,150,55, and the 2026-11-02 decision line said "BOUNDS stays"; both are named for the China package.
11. **Figures:** China's district index of 2,674 is level 6, but the scan proposes level 5 (316); China's district level is open. The largest Chinese province holds 1,958,373 land plots, not "about 2.2 M cells". South Korea has 16 level-4 relations, 15 with ISO; the plan did not mention the 16th. Cambodia's districts cover 92.8 %, not "clean".
12. **Golden file timing:** it must be generated before A or B change anything, so it is A's first commit and B branches after it.
13. **0151** is no longer conditional (decision 3).

## Added after the first measurements

WP F found that eight of the nine province relations that do not assemble from their own extract are whole when the country's cut is completed with a neighbour's cut (`docs/countries/incomplete-relations.md`); WP C's Thailand acceptance found one duplicate district slug that a fresh import would give differently. Both are settings now (WP C2).

**`Import.Neighbours []string`:** Geofabrik extract names whose cuts, `map/data/scan/<name>-latest.osm.pbf.cut.osm.pbf`, complete the country's boundaries. The importer reads the relations from the country's own cut only and their member ways and nodes from the own cut and the neighbours' cuts (`osmfile.ReadBoundaryDataWith`); a neighbour's own relations are never read.

- A neighbour's file is found by the same rules as the country's own (`InputFor`: the cut when it is newer than the extract, else the extract with a note). A neighbour with neither fails check 6, naming the missing cut and `map/cut.sh <name>`. With `-pbf` and no `-data` the neighbours are not looked for (the file given may be a merged cut); the dry run says so.
- The dry run lists every neighbour file it read ("Neighbour read for boundaries only: …"); a country without neighbours gets no such line.
- **The neighbours never feed places**, and neither does a merged cut (`map/merge-cut.sh`): it holds the neighbours' place nodes, and a place outside every province takes the run's country. `places` reads the own cut only, and a test holds it to that. Neighbours never join the base map's `EXTRACTS` either.
- `cmd/countryscan` reads the same neighbours (`-neighbours`, default: the country's settings, `""` for none), cutting a neighbour's extract like the country's own, so scan and import build the same boundaries. A report without neighbours is unchanged; one with them names them in its first paragraph.
- C2 fills `Neighbours` for no country; each country PR sets its own from this table (WP F):

| Country | `Neighbours` | `Partial` |
|---|---|---|
| Vietnam | `philippines` | – |
| Malaysia (Brunei, Singapore share the extract) | `philippines` | – |
| Indonesia | `malaysia-singapore-brunei` | – |
| Taiwan | `china`, `philippines` | – |
| North Korea | `china` | – |
| China | `taiwan`, `philippines`, `bhutan` (Xizang; `bhutan` is not downloaded yet and is read for boundaries only) | – |

- **`LandPlots` from the merged cut:** each country PR takes `LandPlots` from a scan with its neighbours (`cmd/countryscan -country xx`, which now reads them from the settings, or a scan of the merged cut), not from the report of the own extract; against the old figures a whole import of Malaysia (318,281 → 419,805) and North Korea (214,569 → 239,213) would fail check 4. A settings test requires every neighbour to be the `Extract` of some country's settings; China's PR adds `bhutan` to the test's list of boundary-only extracts.

**`Import.LegacyDuplicateSlug bool`, Thailand only:** production stores Thailand's two "Bang Sai" districts of Phra Nakhon Si Ayutthaya as `phra-nakhon-si-ayutthaya-bang-sai` (r10985069) and `phra-nakhon-si-ayutthaya-bang-sai-11024893`, by the old importer's rule; decision 7 would give a fresh import `…-bang-sai-district-10985069` and `…-bang-sai-district-11024893`. Since a stored slug never changes and the development stack, the fixture and production must agree, Thailand keeps the old rule for districts: a stored slug is kept; new districts, in OSM id order, get `<province slug>-<Slug(stripped English name)>`, and `-<osm id>` appended when that slug is taken; names stay stripped, no affix comes back. Provinces, and every other country, follow decision 7. A fresh import of Thailand with this setting gives the same regions, slugs, names, region ids, plot numbers and places as the importers of `main`.
