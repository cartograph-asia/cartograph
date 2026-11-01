# Brief: Design – from prototype to alpha

Read `CLAUDE.md` and `docs/PRINCIPLES.md` first. This brief gives the pages their look. Structure, content order and routes stay as they are after the Home work package; nothing here adds a feature.

Adrian's verdict on the current state: the structure is right, the look is not. The pages read as a competent default, not as a place. Principle 7 says what the target is: big photos, generous whitespace, good type, dense with content and empty of decoration. Early-web spirit, current-day craft.

## What is wrong today

Observed on 2026-11-01 with the Pattaya fixture, at 1400 px and at 400 px. Fix the causes, not these symptoms one by one.

1. **No hierarchy.** Masthead, counters, section headings and captions sit at nearly the same size and weight; `h2` is 1.3 rem against 1.06 rem body text. The page is one grey column in which nothing comes first.
2. **Green does three jobs.** Every plot name and Surveyor name is a green underlined link, the primary button is green, the "first survey" mark is green italic, the bars and map plots are green. In a photo grid the captions become a tangle of green underlines and the colour means nothing any more.
3. **Photos do not dominate.** Five tiles per row at 1400 px, each 13 rem wide, with a caption of three to four lines underneath; the caption takes as much height as the photo. Principle 7 asks for the opposite.
4. **Everything has the same spacing.** Sections are stacked with identical gaps; "Plot of the week" looks like one more list; the regions block is a thin row; the intro is an afterthought at the bottom.
5. **The masthead is generic.** Wordmark, italic claim, a solid green button with a 3 px radius: this is a blog header. The logo is the one distinctive thing on the page and it is 36 px.
6. **Two colour worlds.** Small maps are grey OpenMapTiles-grey on a cream page; the `/map` page is a different product altogether (white Inter UI, pill buttons, its own "Cartograph" badge, a bright "Upload photo" pill).
7. **Phone.** The counters line wraps into four ragged lines; the two-column grid is fine; the masthead stacks acceptably.
8. **Forms and sign-in pages** are usable but were never styled; the error page likewise.

## Direction

Quiet, dense, photographic. The reference is not a start-up landing page and not a photo-sharing app with cards; it is a well-set magazine page or a good community site from before "engagement": a clear masthead, strong headings, photos as large as the column allows, captions in small type, lots of air between blocks and very little inside them.

Rules that follow from it:

- **Type scale with intent.** Define a scale in CSS custom properties and use nothing outside it: display (page titles, the counters' numbers), heading (section heads), body, meta (captions, dates, counters' labels). Section heads must be unmistakably heads: large, or small caps with a rule, but not "slightly bigger body". Prose measure stays at 38 rem.
- **Colour roles.** Slate for text and for links in running text (underlined, underline in a lighter ink). Green is reserved for *actions* (the primary button, form submit) and for *data* (plot marks on maps, coverage bars, the first-survey mark). Plot names and Surveyor names in grids and captions are slate without underline, underlined on hover and focus. The muted ink is for meta only.
- **Photos first.** Desktop: four tiles per row at the full 76 rem width, three in narrower layouts, two on phones. Captions take two lines at meta size: line 1 the plot name, line 2 Surveyor · relative time, with the first-survey mark as a small glyph or short tag on that line, never a third line. Tiles keep 4:3 crops and the thumb variant.
- **Rhythm.** A spacing scale (e.g. 0.5 / 1 / 2 / 4 / 6 rem) and a rule: large between sections, small inside them. Each block on the front page gets a shape of its own: the counters as one statement line with large numbers; the featured block as a full-width photo with its caption in a side column; regions as a compact table-like list where the coverage bar is the row; new Surveyors as a row of small portraits-of-plots with the name under each.
- **Masthead.** Logo and wordmark as one mark, larger than today; the claim as a quiet line under or beside it; actions as text with a single understated primary ("Start surveying" as an outlined or text action with an arrow, not a filled block). Signed-in state shows the Surveyor name as a link and "Sign out" as plain text.
- **One map look.** Retint the base map style (`web/style.json`) to the page: land in the cream family, water a pale grey-blue, roads and labels in light slate, no saturated colours; the plot greens then stand on it. Small maps on the pages and the `/map` page use the same style. The `/map` page gets the page masthead (same template markup, served by the Vite app's HTML), the page typography and the page's button styles for its controls, panel and upload form; MapLibre's own CSS is overridden where it shows (controls, attribution).
- **Nothing added.** No icons set, no shadows, no cards, no carousels, no animation beyond a hover underline, no dark mode in this round. No JavaScript for styling.

## Typography: two directions for round 1

Adrian picks after seeing them on the front page. Build both as a switch in CSS (two token sets), not as two codebases.

- **A – system serif, character by scale.** Keep the current system stack (Charter, Sitka, Cambria, Noto Serif, Georgia); get the identity from size, weight, spacing and the masthead mark. Zero font bytes.
- **B – one self-hosted display face.** One open-licence serif or humanist sans for headings, wordmark and the counters' numbers, Latin subset, woff2, ≤ 40 KB, `font-display: swap`; body stays the system stack. Candidates: Fraunces, Newsreader, Literata, Source Serif 4 (serif) or Inter Tight, Public Sans (sans). Propose two and show them. Thai text keeps falling back per glyph.

Either way the front page stays under 100 KB of HTML + CSS (+ font).

## Process

Design happens in code, reviewed as screenshots. Adrian is the taste; the agent proposes.

1. **Round 1 – the front page.** Tokens (`:root` custom properties for colour roles, type scale, spacing), masthead, counters, grid and captions, featured block, regions, new Surveyors, footer, in both typographic directions. Deliver screenshots at 400 px and 1400 px for each direction into `docs/design/round-1/` (JPEG, ≤ 300 KB each) and a short note on what was decided and why. Open a PR but mark it draft; Adrian picks a direction or asks for changes in the PR.
2. **Round 2 – every page.** Roll the chosen direction out to `/latest`, plot, photo, Surveyor, region, `/regions`, `/surveyors`, static pages, sign-in flow, forms and the error page. Screenshots of every page at both widths into `docs/design/round-2/`. One PR.
3. **Round 3 – the map.** Base map retint, `/map` chrome, small maps. Screenshots of `/map` at both widths and of a plot page's small map. One PR.

A screenshot script (`tools/screenshots`, Playwright against `cmd/webpreview` on 127.0.0.1:8093) is part of round 1 so that every later PR can regenerate the set with one command; `web/package.json` already has Node, so it may live there as a dev dependency. The script, not the agent's description, is the evidence that a page was looked at.

## Sub-agent split

- **Task 1 – Tokens and front page (lead):** `internal/web/static/style.css` rebuilt on tokens, the front page templates adjusted where the markup needs it (caption structure, masthead), both typographic directions, the round-1 note.
- **Task 2 – Screenshot tooling:** `tools/screenshots` with a page list, two viewports, deterministic output names, run against `cmd/webpreview`; a README line on how to run it. Deliver first so task 1 can use it.
- **Task 3 – Map retint:** `web/style.json` in the page palette, checked at zoom 6, 10 and 15 over Chon Buri; the `/map` chrome and MapLibre overrides. This task starts only after Adrian picked a direction in round 1 and runs as round 3.

Tasks 1 and 2 run in parallel; task 2's script must exist before task 1 hands in.

## Done means

Round 1: the front page in two directions, screenshots at two widths in `docs/design/round-1/`, a draft PR, page weight within budget, vocabulary and page tests green. Round 2 and 3: every page and the map in the chosen direction, screenshots regenerated by the script, no inline styles, no new dependency beyond an optional font file and Playwright as a dev dependency, `docs/decisions.md` updated with the typographic choice and the colour roles.
