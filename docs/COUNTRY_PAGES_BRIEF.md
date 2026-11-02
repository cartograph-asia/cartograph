# Brief: Country pages – `/thailand`, `/vietnam`, …

Read `CLAUDE.md`, `docs/PRINCIPLES.md` and `docs/DESIGN_BRIEF.md` first. This brief adds one page per country at `cartograph.asia/{country}`. Adrian's wish (2026-11-02): "we still need sub-pages per country: cartograph.asia/thailand, cartograph.asia/vietnam, etc."

Nothing here changes how plots, photos or Surveyors work. A country page is the region page one level up: the same ingredients (coverage, latest surveys, most active Surveyors, a coverage map), plus the list of the country's provinces.

## Where things stand

- `regions` holds provinces (`kind = 'province'`) and districts. **There is no country row**: a country exists only as the `country` column (ISO 3166-1 alpha-2, lower case) of its provinces. The `regions.kind` check already allows `'country'`, but nothing writes it, and the in-memory region index (`db.ReloadRegions`) deliberately holds provinces only.
- Only Thailand is imported (`cmd/regions -country th`). The map and the base map already span Southeast and East Asia.
- Nothing in the database knows a country's name or URL.

Deliberately not changed: no country rows in `regions`. A row would need a boundary, a `land_cells` it cannot sum without double counting and a place in every `kind = 'province'` query. A country's figures are the sum of its provinces; that is exact because every plot belongs to exactly one province.

## The list of countries

A small Go table, `internal/countries`: code, slug, English name, native name with its language tag. It is the one place that says which countries Cartograph has pages for, whether or not they are imported.

- Initial list, the area of the map (Southeast and East Asia): Brunei, Cambodia, China, Indonesia, Japan, Laos, Malaysia, Mongolia, Myanmar, North Korea, Philippines, Singapore, South Korea, Taiwan, Thailand, Timor-Leste, Vietnam. Hong Kong and Macau are not listed (see open questions).
- **Slugs** are the English short name, lower case, hyphens: `thailand`, `vietnam`, `laos`, `south-korea`, `timor-leste`. Not the ISO code (`/th` says nothing to a visitor) and not the long official name.
- A slug must never equal a first path segment the site already uses. A test walks the router and the reserved list (`/api`, `/healthz`, `/robots.txt`, `/map`, `/tiles`, `/img`, `/static`, `/admin`, `/signin`, `/signout`, `/settings`, and every other first segment of a registered route) against the list and fails on a clash. Adding a route whose first segment is a country's slug fails the same test.

Why Go and not a table: the list is reference data that changes with a release, like `StaticPages`; a test can check it without a database; and no migration is needed. A country's *naming rules* (`place_naming`) stay data, as decided.

## URLs

| URL | Result |
|---|---|
| `/{slug}` | the country page, 200 |
| `/{slug}?page=N` | page N of its latest surveys; a `page` that is not a positive integer or lies past the last page is 404 (same rule as `/region/{slug}`) |
| `/{Slug}` (any case), `/{slug}/`, `?page=1` | 301 to the canonical spelling |
| `/{slug}.rss` | the country's feed (principle 5: every region has an RSS feed); canonical `/{slug}.rss` |
| `/anything-else` | the 404 page, as today. Nothing is guessed or redirected: `/th`, `/tailand` and `/thailand-2` are 404 |

`/region/{slug}` is unchanged and stays the address of provinces. `/regions` stays the list of provinces and gains the countries on top (see below), so a visitor can reach every country page by links alone.

The route is `/{country}` registered after every static route; chi prefers static routes over a parameter, so `/about`, `/latest` and the rest are not affected. Caddy already sends everything except `/map*`, `/tiles/*` and `/img/*` to the api, so no proxy change is needed.

## What a country page shows

Server-rendered, no inline styles, no new CSS beyond classes in `style.css`; it reuses the region page's head, figures, coverage bar, grid, activity lists and mini map.

1. **Head.** Kicker "Country", title `Thailand` with the native name in its own language tag, figures (coverage with "x of y plots", photos, Surveyors, provinces with activity), coverage bar, a note with the last survey and the RSS link.
2. **Coverage map.** The mini map, bounds = the union of the provinces' boxes; "Open on the map →".
3. **Provinces.** The `region-row` list (name, coverage, bar, most active Surveyors), newest activity first, then by name, exactly as on `/regions` but only this country's.
4. **Latest surveys.** The grid of the country's newest photos, paginated like a region (24 per page). Chronological, nothing else (principle 2).
5. **Most active Surveyors.** Last 30 days and all time, as on the region page.

Not on the page: **Plot of the week**. It is one editorial pick for the whole site (`featured`), not per area; showing it here would either repeat the front page or need a per-country pick nobody writes. Left out until Adrian wants per-country picks.

Coverage figures are sums over the country's provinces (`land_cells`, `claimed_cells`), so they match the provinces listed beneath.

Province pages gain a link back: the kicker reads "Province in Thailand" with the country linked.

### `/regions`

A "Countries" block above the provinces: one row per country in the list, in list order by name. Countries with data show the same row as a province (name, coverage, bar, top Surveyors); countries without data show the name and "Not open yet" and still link to their page. The provinces list below is unchanged except that its note no longer says "Thailand's".

### A country that is in the list but has no data

`/vietnam` before Vietnam is imported answers **200**, with `noindex`, as a blank plot does:

- Head: the name, kicker "Country", one sentence: "Cartograph does not cover Vietnam yet. Plots open here once its map has been imported."
- No figures, no map, no feed link (`/vietnam.rss` is 404 until there is data), no empty grids.
- Link to `/about` for what Cartograph is and to the contact in the footer.

Once the importer has run, the same URL shows the full page; nothing else changes, no deploy.

A country that is imported but where nobody has surveyed a plot yet is the normal page at 0 %, with "Nobody has surveyed a plot in Vietnam yet. Be the first." in place of the grid (as regions do).

## Adding a country

1. A line in `internal/countries` (code, slug, names). The slug clash test runs with it.
2. If plots there are to get names: a migration with the country's row in `place_naming` (as migration 0075 for Thailand). Without a row, `regions` and the country page still work; plots are named by their province only.
3. `regions -pbf <extract> -water <water> -country vn`, then `places` for the same extract. `regions` refuses a country code that is not in the list, so a typo cannot create regions no page shows. The first province import for a country needs no deploy: the api reloads its region index within a minute.

Not part of this brief, and the importer's limit to know: `regions` finds provinces as OSM `admin_level=4` relations with an `ISO3166-2` tag starting with the country code. That fits Thailand and, to be checked per country, Vietnam, Laos, Cambodia, Myanmar and Malaysia; the Philippines, Singapore, Brunei and others use other levels. Each country therefore needs a look at its OSM data before its import, and may need a per-country admin level in the importer (a follow-up brief).

## Tests

- `internal/countries`: slugs are valid (`^[a-z]+(-[a-z]+)*$`), codes and slugs unique, names non-empty, native names carry a language tag.
- `internal/web`: slug-versus-route clash test; template tests for the country page (full, no data, 0 %), the vocabulary test over the new pages (no "hex", "square", "Cartograph Asia"), the page test with `viewstest.Fake` (200, 404 for an unknown slug, 301 for case and trailing slash, RSS, `?page=` rules, `noindex` and no feed link when empty), the province page's country link, the countries block on `/regions`.
- `internal/db` (needs `TEST_DATABASE_URL`): country figures equal the sum of the provinces in the Pattaya fixture; a country with no regions yields the empty page; the feed holds only that country's photos.
- `internal/importer`: `regions` rejects a code that is not in the list. The Pattaya fixture runs as before (Thailand).
- `go vet`, `go test ./...`, `/security-review` on the branch before the PR.

## Out of scope

Per-country plot-of-the-week, country flags, a country index page of its own (`/countries`; `/regions` carries it), city pages, `sitemap.xml`, per-country importer settings, translating the site, any change to the `/map` page. No migration, no change to stored data.

## Open questions (also in the PR)

1. **The list.** Is the list above right? East Asia is a political question (Taiwan, Hong Kong, Macau, North Korea); each is one line to add or drop.
2. **Slugs.** `laos` (not `lao-pdr`), `myanmar` (not `burma`), `timor-leste` (not `east-timor`), `south-korea`, `north-korea`, `taiwan`. Right?
3. **Unopened countries** answer 200 with an explanation. The alternative is 404 until imported. A 200 lets people share `/vietnam` early and costs nothing; say if you prefer 404.
4. **Names for the level below.** The page says "Provinces". Japan has prefectures, Malaysia states, Myanmar states and regions. `regions.kind` has only `province` today; a per-country word (`countries` table column or Go field) is easy once the first non-province country is imported.
5. **Upload message.** `internal/api/photos.go` says "outside the area Cartograph covers so far (Thailand)". It stays true until a second country is imported and should then be built from the imported countries. Not changed here.
6. **Front page.** The regions block still lists provinces of all countries together once there are several. Group by country, or show countries first? Left for when a second country exists.
