# Design round 3 – the map

**Decision (Adrian, 2026-11-01): layout c**, the map edge to edge with the masthead and the key as paper labels on it. Brief: [docs/DESIGN_BRIEF.md](../../DESIGN_BRIEF.md). The decisions are logged in [docs/decisions.md](../../decisions.md) under "Design".

## Screenshots

Taken with `cd web && npm run screenshots -- round-3`. The `/map` page and the small maps need tiles and the API, so this set is read from the compose stack on 127.0.0.1:8080 (`docker compose up` in `deploy/`, with the web image rebuilt after a change in `web/`). It shows the local data, not a fixture.

| Page | 400 px | 1400 px |
|---|---|---|
| `/map`, Thailand | [map-400.jpg](map-400.jpg) | [map-1400.jpg](map-1400.jpg) |
| `/map`, plots around Pattaya | [map-plots-400.jpg](map-plots-400.jpg) | [map-plots-1400.jpg](map-plots-1400.jpg) |
| `/map`, a plot's panel open | [map-panel-400.jpg](map-panel-400.jpg) | [map-panel-1400.jpg](map-panel-1400.jpg) |
| A plot page's small map | [plot-smallmap-400.jpg](plot-smallmap-400.jpg) | [plot-smallmap-1400.jpg](plot-smallmap-1400.jpg) |
| A region page's small map | [region-smallmap-400.jpg](region-smallmap-400.jpg) | [region-smallmap-1400.jpg](region-smallmap-1400.jpg) |

The three layouts that were compared are in [variants/](variants/), named `{a,b,c}-{overview,plots,panel}-{400,1400}.jpg` (taken with a one-off script while the `?layout=` switch existed):

- **a – masthead bar:** the pages' masthead as a bar above the map, legend and status in a strip below.
- **b – atlas column:** masthead, legend and status in a column beside the map; the panel takes the column's place.
- **c – full-bleed (chosen):** the map edge to edge, masthead and key as labels on it.

## What changed, and why

- **The map was cut off.** The tiles covered only the OSM extract's box, so the Andaman Sea and everything beyond Thailand's box showed as bare background with a hard edge. The archive now covers the map's whole bounds; ocean and neighbouring land come from the global water polygons and Natural Earth.
- **One look.** `/map` uses the pages' tokens, serif, masthead and buttons. No shadows, no pills, no Inter. MapLibre's controls and attribution are restyled to match, on `/map` and on the small maps.
- **Base map retint.** Land in the paper family, water a pale grey-blue, roads and labels in light slate, so that the plot greens are the only colour.
- **Quieter far out.** Below zoom 8 the outlines of blank parent cells are hidden; they used to cover the whole view, sea included.
- **Map key.** The three greens (1, 2–3, 4 or more photos) and a blank plot, drawn as small plot shapes, plus one line on what a plot is.

## After the first review (Adrian, 2026-11-01)

1. **Neighbouring countries** get OSM detail: the extracts of Myanmar, Laos, Cambodia, Vietnam and Malaysia/Singapore/Brunei are merged with Thailand's before the tile build.
2. **The key on phones** is one line, the status and a "Key +" toggle; the legend and the line about plots open above it. From 768 px it stays open.
3. **Small maps** colour plots in the three greens by photo count (1, 2–3, 4 or more), like `/map`; the no-JS outline does the same.
4. **The key follows the zoom:** single plots close in; further out, where the map groups plots, it says how many plots a shape holds (49 or 343) and how it is shaded.
5. **Groups with photos are clickable:** under the mouse such a group is outlined, and a click zooms until it fills the view. Blank groups only pan.

Also fixed on the way:

- The grid is redrawn while the map moves (every 150 ms, from what is already loaded), and data is fetched as soon as the movement stops, instead of 200 ms after it, so zooming no longer leaves an empty margin.
- Plot requests ask for 800 cells, not 2000: Caddy answered the 32 KB URLs with a 431, so in large views many plots never got their colour.
