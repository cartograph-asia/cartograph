# Map room: air around the countries

Screenshots of `/map` at 400 and 1400 px (viewport, not full page), before and after branch `map-room`. "Before" is the compose stack on 127.0.0.1:8080; "after" is the branch's build served by `vite preview` against the same stack's tiles and API.

- `before-open-*`: without a position in the URL the map opened on Thailand at zoom 6.
- `before-south-*`: zoomed out as far as allowed (4.5) and panned south as far as allowed. `maxBounds` equalled the archive's bounds (-12° in the south), so the view stopped at the data's edge with Timor under the key.
- `before-archive-edge-1400`: north Australia at zoom 6 with `maxBounds` lifted (a test build). Below the tile edge past -12° the sea turns into the land-coloured background: the edge that a turned map showed past the corners of `maxBounds`.
- `after-open-*`: the opening view: Southeast and East Asia whole and centred between masthead and key, paper all around.
- `after-south-*`: the same position as `before-south-*`: the islands clear of the key, the sea fading into paper below them.
- `after-edge-*`: the archive's north-east corner (55° N, 150° E) at zoom 5: land and sea fade into the paper; no tile edge.
