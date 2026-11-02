# Brief: Blank plots – a way to the next plot that waits for a photo

Read `CLAUDE.md`, `docs/PRINCIPLES.md`, the Home and Design entries in `docs/decisions.md` and `docs/GAMIFICATION.md` (core loop) first. This brief adds one page that shows a few **blank plots** to a Surveyor, drawn at random from the Surveyor's own region. It is the first of the ways in that Geograph has and Cartograph lacks; search is a separate brief.

## Why

The core loop is "see a blank plot near you, go there, take the photo, see the plot turn surveyed" (`GAMIFICATION.md`). Today a Surveyor finds blank plots only by panning `/map`. A page that says "these plots in your region still wait for a first photo" closes the loop without a map, works without JavaScript and is the answer to Geograph's "61,226 squares with fewer than four photos, add yours now".

## Decisions (Adrian, 2026-11-02)

1. **The region comes from the Surveyor, never from the visitor's connection.** In this order:
   1. the **home region** the Surveyor chose in `/settings` (`surveyors.home_region_id`, voluntary, already there);
   2. otherwise the region **derived from the Surveyor's own visible photos**: the province holding most of the plots they have surveyed, ties broken by the province of their latest photo;
   3. otherwise no region: plots are drawn from the whole surveyable area.
2. **A suggestion, not a filter.** The page says where its plots come from and lets the Surveyor change it: "Plots in Chon Buri (your home region)", "Plots in Chon Buri (where you surveyed most)", "Plots anywhere". Nobody is bound to a region; a traveller can pick another one.
3. **Random and said so.** The plots are drawn at random from the region's blank plots on every load, and the page states it ("Chosen at random from the blank plots of Chon Buri. Reload for others."). No ranking, no "for you" (Principle 2).
4. **No IP lookup, no stored guess.** The derived region is computed on read from the Surveyor's photos and stored nowhere. No GeoIP database, no cookie for visitors, no location request. A "near me" variant would use the browser's geolocation on a click; it comes after v1 (see below).
5. **No pressure, no points.** Showing a plot scores nothing and nothing is counted or remembered about what was shown (Principles 6 and 8).

## What the page shows

Route `GET /blank` (open to everyone; `?region={slug}` picks a region, `?where=anywhere` the whole area). Title "Blank plots", kicker "Waiting for a first photo".

- **Source line** (lede): where the region comes from, as in decision 2, with the change links "Another region" (links to `/regions`) and "Anywhere". A signed-in Surveyor without a home region and without photos sees "Anywhere" and a note: "Set your home region in your profile to see plots near you."
- **Six blank plots** in the existing photo-grid pattern, but with the plot's small outline map (the server-side SVG, `h3.Boundary`) in place of a photo: plot name (`PlotRef.Name()` via `heading`, short form in the caption), region, a link to `/plot/{slug}/{number}` and "Open on the map →" (`/map?cell=…`).
- **"Show other plots"**: a plain link to the same URL (new draw). Fewer than six blank plots left in the region: show what there is; none: "Every plot of Chon Buri has a photo. Thank you." with the link to Anywhere.
- Links to the page: the footer ("Latest · Regions · Surveyors · Map · Blank plots"), each region page ("Find a blank plot here", `/blank?region={slug}`) and the Surveyor's own profile. The front page stays as it is (Principle 1).

## Which plots count

A plot is **blank** when it is a land plot of the region (`plot_numbers`, the region import's land cells) and has no visible photo (no `plots` row, or `photo_count = 0`; a plot whose only photo is hidden counts as blank again, as in the Admin brief). Numbers run 1..n without gaps per province (decisions, Home), so the draw picks random numbers and drops the claimed ones, repeating a few rounds, instead of sorting the table at random; a province that is nearly complete falls back to listing its remaining blank numbers. Plots in water never appear (they have no number).

## Data

No migration expected. If a query needs an index, the lead assigns the number (0110 is reserved for the comments work package). The derived region is one query over `visible_photos` joined to `plots.region_id`, and it only reads the signed-in Surveyor's own rows.

## Code

- `internal/views`: `BlankPage` embedding `Page` (Region `*RegionRef`, `Source` as a typed string `home`/`photos`/`chosen`/`anywhere`, up to six `BlankPlot` with `PlotRef`, cell and the outline for the SVG, plus the region choices); `Source` gets `BlankPlots(ctx, regionID *int, n int)` and `DerivedRegion(ctx, surveyorID string)`. Fixtures in `viewstest.Fake`.
- `internal/db`: the two methods in `db.Views`. The draw and the derived region live next to the other region queries; nothing outside `internal/db/admin.go` reads `photos` directly (a test enforces it), so use `visible_photos`.
- `internal/web`: handler for `/blank`, template `blank.html`, route in the router, entry in the footer partial and on the region and profile pages. Responses are `Cache-Control: private, no-store` and `noindex` (random content, no canonical URL worth indexing).
- The region comes from `views.ViewerFrom(ctx)` and the Surveyor's id, not from a parameter the visitor could forge: `?region=` only picks the region to draw from, it reads nothing about anybody.

## Look

No new components: the photo grid, the outline map, the source line in the lede style, plain text links. No inline styles, no one-off CSS (`DESIGN_BRIEF.md`). Screenshots at 400 and 1400 px of: a Surveyor with a home region, one with a derived region, a visitor, a region that is complete, in `docs/design/blank/`.

## Tests

- **db:** only land plots without a visible photo are drawn, never a claimed one, never a water cell; a plot whose only photo is hidden is a candidate; the draw returns fewer than six when fewer remain, none for a complete region; the derived region is the province with most surveyed plots, ties go to the latest photo, and photos of other Surveyors or hidden photos do not count.
- **web:** the source line for each of the four cases (home, derived, chosen, anywhere); unknown region slug is 404; the page sets no cookie for a visitor; vocabulary and page tests (the words "blank" and "plot", never "square" or "hex"); no `style` attribute; page weight within budget; headers are `no-store` and `noindex`.

## Done means

From a clean `docker compose up` with the Pattaya fixtures: signed in as a Surveyor whose home region is Chon Buri, `/blank` shows six blank Chon Buri plots with outlines and links; reloading shows others; clearing the home region in `/settings` makes the page derive Chon Buri from the Surveyor's photos and say so; a new Surveyor without photos gets "Anywhere" and the hint; a visitor can use `/blank?region=chon-buri`. `go test ./...` green with `TEST_DATABASE_URL` set; `/security-review` run.

## Decided later questions (Adrian, 2026-11-02)

- **Near me: later.** v1 takes the region from the home region, the Surveyor's photos or the visitor's choice only. A button that asks the browser for its position once (on a click, stored nowhere) is a small step of its own after v1.
- **Plots with one photo: with the points.** "Plots that would like a second look" comes with the points system (`GAMIFICATION.md`, the second photo scores 2), not in v1.
- **Name and address: `/blank`, "Blank plots".** The working names stay.
- **Below province: later.** `?region=` takes province slugs only; districts become a choice once they have pages.
