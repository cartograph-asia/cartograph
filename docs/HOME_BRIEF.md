# Brief: Home – the community front page

Read `CLAUDE.md` and `docs/PRINCIPLES.md` first. This brief builds the public front page of cartograph.asia and the pages it links to, on top of the merged prototype. The map stays as it is and moves to `/map`.

The front page is the Cartograph of 2005's community sites in today's craft: it shows what people did, who they are and how far the survey has come. It is not a landing page with a hero image and three feature boxes, and it is not a map.

## Scope

In: front page, Surveyor profile page, plot page, photo page, region page, RSS feeds, server-side rendering, the minimal account needed to attach photos to a person.
Out: comments, ranks, country titles, "where was this" mode, moderation tools, e-mail digest, mobile app. Leave hooks in the data model where cheap; build nothing of these.

## Pages

### `/` – Home
Order top to bottom; every block is server-rendered HTML.

1. **Masthead.** Logo, "Cartograph", claim. Right: "Start surveying" → `/map`, "Sign in".
2. **Counters.** Plots surveyed, photos, Surveyors, regions with activity. Plain numbers, one line. Click a number → the matching list.
3. **Latest surveys.** Grid of the 24 newest photos: thumbnail, plot name (nearest place from OSM, see "Place names"), region, Surveyor name, relative time, a small "first survey" marker when the photo claimed its plot. Chronological. "More" → `/latest` with pagination (no infinite scroll).
4. **Plot of the week.** One photo large with a short text. Editorial, set by hand in the DB (`featured` table); falls back to the most recent first survey if nothing is set.
5. **Regions.** Thailand's provinces (later: countries, then provinces): name, coverage percentage, bar, three most active Surveyors in the last 30 days. Sorted by recent activity, not by size.
6. **New Surveyors.** The eight most recently joined, each with their first photo. "Welcome" tone, no numbers.
7. **Footer.** About, Principles, data & licence, RSS, GitHub, contact. No social-media bar; the handles are a line of text.

### `/surveyor/{name}` – Profile
Name, joined date, home region (optional, self-chosen), a short bio (plain text, 500 chars). Counters: plots claimed, photos, regions. A small map (MapLibre, lazy-loaded below the fold) showing their surveyed plots. Then their photos, chronological grid, paginated. RSS link.

### `/plot/{cell}` – Plot
Place name and region, first surveyor and date, photo count. All photos chronological. Small map centred on the plot with its neighbours' status. Link "Survey this plot" → `/map?cell=…`. Seasonal note later; keep the layout open for it.

### `/photo/{id}` – Photo
The large image, Surveyor, plot (linked), taken/uploaded dates, licence line, "first survey of this plot" badge when true. Previous/next within the plot. Open Graph tags so a shared link shows the photo.

### `/region/{slug}` – Region
Province (and later country): coverage, counters, a map of coverage at parent resolution, latest surveys there, most active Surveyors (30 days, all time). RSS link.

### `/latest`, `/latest.rss`, `/region/{slug}.rss`, `/surveyor/{name}.rss`
Plain chronological lists and their feeds. Feeds carry the thumbnail and a link; nothing more.

## Accounts (minimum)

E-mail + magic link, no passwords. A Surveyor name is chosen once (unique, 3–30 chars, letters/digits/hyphen). The upload form on `/map` requires sign-in. Existing prototype photos with free-text surveyor names are migrated to placeholder accounts that Adrian can claim by e-mail. No OAuth, no roles, no admin UI; `featured` is edited in SQL.

## Place names

Each plot gets a display name on first claim: the nearest OSM place (village/town/city) within 5 km from the Thailand extract, else the province. Precompute into a `places` table from the same extract used for the map; one Go command in `cmd/places`. The plot page shows "near {place}", never a bare cell id in user-facing text.

## Regions

Thailand's 77 provinces from OSM admin boundaries (admin_level 4) in the extract, imported once into a `regions` table with a simplified polygon. A plot belongs to the region containing its centre. Coverage = plots claimed / res-8 cells whose centre lies on land within the region (land from the same boundary; sea excluded). Compute once, update incrementally on claim.

## Rendering and stack

Server-side rendered HTML from the Go backend with `html/template`; the map page keeps its Vite bundle. No frontend framework for these pages; a few lines of vanilla JS for lazy maps and the sign-in form are fine. Pages must be complete and readable with JavaScript disabled. Images via `img.cartograph.asia` keys as before; add a `medium` variant (1200 px) for photo pages. Target: front page under 100 KB of HTML and CSS, first photo visible within one second on a phone.

Typography and layout: generous whitespace, one serif or humanist sans for text, photos dominate, no cards-with-shadows, no icon rows, no carousels. Colour palette from the logo: cream #F6F2EA, slate #2F3A3F, greens #2E7D4F / #5FA57A / #A8CDB3.

## Sub-agent split

- **Task 1 – Data:** migrations (accounts, places, regions, featured, photo medium variant), `cmd/places` and `cmd/regions` importers, coverage computation, queries for every page. Deliver with tests on a fixture extract (a small bounding box around Pattaya).
- **Task 2 – Accounts:** magic-link sign-in (SMTP via env; in dev, log the link), session cookie, name choice, migration of prototype photos to placeholder accounts, upload form gated on `/map`.
- **Task 3 – Pages:** templates and CSS for all pages above, RSS, Open Graph. Develop against fixture data from task 1's test set.
- **Task 4 – Integration (lead):** wire, run with the Pattaya fixtures, check every page on phone width and desktop, measure page weight, update README and `docs/decisions.md`.

Tasks 1–3 run in parallel; the page templates depend on the query shapes, so task 1 publishes its result types first (a `internal/views` package with plain structs) and tasks 2 and 3 build against them.

## Done means

From a clean `docker compose up` with the Pattaya fixtures: the front page shows real counters, the latest surveys, a featured photo, the provinces with coverage and the new Surveyors; every photo, plot, Surveyor and region links to its own page; a signed-in Surveyor can upload from `/map` and sees their photo on the front page on reload; feeds validate; the front page is under 100 KB and renders without JavaScript.
