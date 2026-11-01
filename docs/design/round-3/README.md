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

## Open points for Adrian

1. **Neighbouring countries** are plain land without names or roads, because the OSM extract is Thailand's. Merging the neighbours' extracts would add them at the cost of a larger archive and build.
2. **The key on phones** takes two lines at the bottom. It could collapse to the status line alone, with the legend behind a tap; that would need a few lines of script.
3. **Small maps** keep their own colours (surveyed plots in the middle green, the plot itself outlined in ink) instead of the three steps by photo count. The pages' small maps do not know photo counts yet.
