# Brief: Profile – more of a place, less of a counter

Read `CLAUDE.md`, `docs/PRINCIPLES.md` (3: named people with pages), the Home, Design and Display name entries in `docs/decisions.md` and `docs/GAMIFICATION.md` first. This brief gives the Surveyor page (`/surveyor/{name}`) the depth that Geograph's profile has and ours lacks: more figures, a breakdown by region and by year, ways to look through the photos, and exports of the plots. It is a comparison with `https://www.geograph.org.uk/profile/11144`; what that page has and does not fit Cartograph is listed under "Left out".

## Why

Today a profile says three numbers, shows a minimap and a newest-first grid. A Surveyor with 400 photos can neither see where they have been most active nor look at their oldest photos without paging through all of them. Principle 3 says a profile is "their plots, their map, their photos, their story"; Principle 5 says data can be exported and only the RSS feed exists.

## Decisions (Adrian, 2026-11-02)

1. **Home region stays a region.** The profile keeps the voluntary home region from `/settings` and does not get a home plot or any other more precise home location: a plot is about 0.74 km², in a village a handful of houses, and profiles are public without a login. Nothing in this brief reads or stores a more exact place for a person. (Geograph shows a 1 km grid square; we deliberately do not.)
2. **More figures, all derived on read.** Nothing new is stored.
3. **Breakdowns link into the photos.** Every region and every year in the breakdowns is a link that filters the photo list below, so the numbers are a way into the photos, not decoration.
4. **Chronological, chosen by the visitor.** The photo list is sorted by a named, visible order (newest uploaded, oldest uploaded, newest taken), the active order is shown, and the default is today's. There is no random and no "best" order (Principle 2).
5. **No pressure.** The year breakdown is a list of counts, not a calendar of streaks, and nothing shows how many days in a row anyone was active (Principle 6).
6. **Exports of plots, not of positions of people.** GPX and GeoJSON list the plot centres (not EXIF positions) of the plots a Surveyor has photos in, with the plot's name and link. They contain no photo, no time and no e-mail address.

## What the page shows

Route unchanged: `GET /surveyor/{name}`, plus the optional query below. The existing parts stay: header, minimap, photo grid.

- **Figures** (the `figures small` list): plots surveyed (with "n of them first", as today), photos, regions and **days**: the distinct dates on which they took photos, by the clock where the photo was taken (`taken_at_local`, else the UTC date of `taken_at`); shown when more than 0. Vocabulary as in `CLAUDE.md`.
- **Home region line**, when the home region is set: "{n} of {land} plots of {Region} surveyed by {Label}" with the existing coverage bar and `pct`. Plots in the region that others surveyed do not count.
- **By region** (section "Where"): the regions in which they have photos, most plots first, then by name: region name, "{n} plots · {m} photos", and the share "of {land}". Each name links to `?region={slug}`. Hidden when they have photos in one region only.
- **By year** (section "When"): the years in which their photos were taken, newest first, each "{year} · {n} photos", linking to `?year={year}`. Photos without a taken date are not listed. Hidden when fewer than two years.
- **Photos controls** above the grid, plain links, no script: order "Newest uploaded · Oldest uploaded · Newest taken" (the active one is `aria-current`), a toggle "One photo per plot / All photos", and, when a region or year filter is active, a line "Showing {n} photos in {Region}, {year}" with "Show all". "One photo per plot" shows the newest of their photos in each plot.
- **Exports** in the note line: RSS feed (existing), "GPX of the plots" and "GeoJSON of the plots".

### Query

`GET /surveyor/{name}?order=oldest|taken&region={slug}&year={yyyy}&one=plot&page=N`

- `order` absent means newest uploaded; `oldest`, `taken`; any other value is 404. `taken` sorts by `taken_at` descending, photos without one last, ties by upload time.
- `region` is the slug of any region (province, district, country); the plots of that region and of its districts count. An unknown slug is 404; a known region where they have no photos is an empty list with the line above, not 404.
- `year` is four digits between 2000 and the current year, else 404.
- `one=plot` is the only value; any other is 404.
- A request that spells the same view differently (`?page=1`, parameters in another order, upper case in the slug) is redirected permanently to the canonical form. Filtered or reordered views are `noindex` (their canonical link is their own normalised address) and pagination keeps the parameters.

### Exports

- `GET /surveyor/{name}.gpx`: GPX 1.1, one `wpt` per surveyed plot: `lat`/`lon` of the plot centre, `name` = plot name (`PlotRef.Name()`), `desc` = "{n} photos by {Label}", `link` = the plot's absolute URL.
- `GET /surveyor/{name}.geojson`: a `FeatureCollection` of `Point`s with the same name, `url`, `photos` (theirs) and `first` (whether they claimed it).
- Content types `application/gpx+xml; charset=utf-8` and `application/geo+json`, `Content-Disposition: inline; filename="{name}.gpx"`. The canonical redirect rules of the RSS feed apply (case of the name). Escaping is the encoder's (`encoding/xml`, `encoding/json`); nothing is concatenated by hand.

## Which plots and photos count

Everything reads `visible_photos`, as the profile does today; a hidden photo counts nowhere, and a plot whose only photo of theirs is hidden is not exported. Counts per region use `plots.region_id` (the province), the region filter also matches `plots.district_id`. A blocked Surveyor's page behaves as today.

## Data

No migration. `photos_surveyor_idx (surveyor_id, uploaded_at desc)` serves the lists; the breakdowns are one grouped query each over the Surveyor's own rows. If a profile with several thousand photos turns out slow, an index on `(surveyor_id, taken_at)` is the answer and the lead assigns the number (0110 is reserved for the comments work package).

## Code

- `internal/views`: `SurveyorQuery{Page int; Order string; Region string; Year int; OnePerPlot bool}` with constants for the orders; `Source.Surveyor(ctx, name, SurveyorQuery)` replaces the `page int` argument. `Surveyor` gets `Regions []SurveyorRegion` (Region `RegionRef`, Plots, Photos, Land), `Years []SurveyorYear`, `Query SurveyorQuery` (as parsed), `FilterRegion *RegionRef` for the "Showing" line, `HomeCoverage *Coverage`; `SurveyorStats` gets `Days`. New `Source.SurveyorPlots(ctx, name) (*SurveyorPlotList, error)` for the exports (Surveyor `SurveyorRef`, plots with `PlotRef`, centre, their photo count, first-survey flag).
- `internal/db`: the extended `Surveyor` and `SurveyorPlots` in `db.Views`. Filters are built from validated values and bound parameters, never concatenated; `nothing outside internal/db/admin.go reads photos directly` still holds, use `visible_photos`.
- `internal/web`: the handler parses and validates the query (404 on bad values), the `.gpx` and `.geojson` suffixes next to the `.rss` one, `internal/web/export.go` writes both formats. The helper `canonical` compares the parameters it knows (`page`, `order`, `region`, `year`, `one`) in canonical form instead of only `page`; other pages behave as before.
- `viewstest.Fake` and the fixtures show every state (several regions and years, no taken date, no home region, filtered).

## Look

No new components: the existing `figures`, `cov`/`covbar`, `cols`, `head-row` and `note` patterns; the two breakdowns use the `activity` list style. The controls are one line of text links with `aria-current` on the active one, styled in `style.css` with existing tokens. No inline styles, no one-off CSS (`DESIGN_BRIEF.md`). The page stays readable and fully usable without JavaScript.

## Tests

- **db:** stats include days; the home-region share counts only their own plots; the region and year breakdowns count only visible photos of that Surveyor and sort as stated; each order sorts as stated (photos without `taken_at` last); `one=plot` returns the newest photo per plot and the total used for paging matches; the region filter matches a district; unknown slug is `ErrNotFound`; a plot with only hidden photos of theirs is neither listed nor exported.
- **web:** canonical redirects (`?page=1`, parameter order, slug case); 404 for bad `order`, `year`, `one` and unknown region; filtered pages are `noindex` and pagination keeps the parameters; the figures, the "Where" and "When" sections and the controls appear and disappear as stated; GPX and GeoJSON parse, carry the stated fields, escape a display name such as `Ada "<b>" Lovelace`, and contain no e-mail address or time; vocabulary and page tests (no "square", no "hex"); no `style` attribute; page weight within budget.

## Done means

On the Pattaya fixtures: `/surveyor/adrian` shows the figures, "Where" and, if the fixture has two years, "When"; `?order=oldest`, `?region=chon-buri`, `?year=…` and `?one=plot` change the list and keep paging; the GPX opens in a GPX viewer and the GeoJSON in geojson.io with the right plots; clearing the home region in `/settings` removes the home line. `go test ./...` green with `TEST_DATABASE_URL` set; `/security-review` run; screenshots at 400 and 1400 px in `docs/design/profile/` (full profile, filtered, a Surveyor with one region and no taken dates).

## Left out, and why

- **A home plot or grid reference** (Geograph: "Home grid reference"): decision 1.
- **Profile picture / Gravatar**: Gravatar makes every visitor's browser ask a third party (Principle 5, no tracking). An own upload would need a design for storage, moderation and the admin area; a separate brief.
- **Send message**: there are no direct messages; conversation lives at the photo and the plot (Principle 4).
- **Points, TPoints, rank, titles**: `GAMIFICATION.md` is still design only. The "Where" list is the place a later "Surveyor of {country}" title would sit.
- **Random order, slideshow, "Georiver"**: random is outside Principle 2's "you always know why you see this"; the others are views of the same photos that the grid already gives.
- **Calendar of days**: replaced by the year list; a month calendar would invite streaks (Principle 6).
- **KML**: GPX and GeoJSON cover the same tools; KML can follow if somebody asks.
- **Marked images, tags, "geographical context" breakdown**: Cartograph has no tags or favourites yet.

## Open questions

- **Table view.** Geograph lists recent photos as a table (grid reference, title, submitted, type, points, taken). A `?view=list` table with plot, taken, uploaded and "first" would suit Surveyors with thousands of photos; include now or later?
- **Years by upload or by taken date.** The list uses the taken date, as that is what a Surveyor means by "2024"; the uploaded date is the other possible reading.
- **Region filter on districts.** Districts have no pages of their own yet; the filter accepts their slugs, but nothing links to them. Offer the district list under each province in "Where" once they do.
