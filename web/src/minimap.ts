// Small maps on the server-rendered pages (profile, plot, region).
//
// The pages import this module (built as /map/assets/minimap.js) only when a
// [data-minimap] element scrolls into view. Each element already holds a
// complete no-JS fallback (an SVG of the plots or a link); the map replaces it.
//
//   data-minimap="plots"     data-cells="c1,c2,…" [data-photos="6,0,…"] [data-focus="c1"]
//   data-minimap="coverage"  data-bounds="minLng,minLat,maxLng,maxLat"  (fetches /api/coverage)
import maplibreCss from 'maplibre-gl/dist/maplibre-gl.css?inline';
import { addProtocol, Map as MapLibreMap, NavigationControl, setWorkerUrl } from 'maplibre-gl';
import type { LngLatBoundsLike, MapLayerMouseEvent } from 'maplibre-gl';
import workerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?url';
import type { Feature, FeatureCollection, Polygon } from 'geojson';
import { Protocol } from 'pmtiles';
import { httpApi } from './api/http';
import { type BBox, cellRing, cellsForBBox, chunk, COVERAGE_BATCH, estimateCellCount, MAX_VIEW_CELLS } from './grid';
import { baseStyle } from './style';

// The three greens of /map (src/layers.ts): 1 photo, 2–3, 4 or more.
const GREEN_LIGHT = '#A8CDB3';
const GREEN = '#2E7D4F';
const GREEN_MID = '#5FA57A';
const SLATE = '#2F3A3F';

setWorkerUrl(workerUrl);
addProtocol('pmtiles', new Protocol().tile);

// MapLibre's controls in the pages' look, as on /map (src/styles.css): square,
// no shadow, the paper colour and the page's serif for the attribution. The
// last rule keeps the map filling its box whichever of MapLibre's
// .maplibregl-map {position: relative} and the pages' .minimap-canvas loads last.
const OVERRIDES = `
.maplibregl-ctrl-group,.maplibregl-ctrl-group:not(:empty){border-radius:0;box-shadow:none;border:1px solid rgba(47,58,63,.32);background:#f6f2ea}
.maplibregl-ctrl-group button+button{border-top:1px solid rgba(47,58,63,.16)}
.maplibregl-ctrl-group button:focus:first-child,.maplibregl-ctrl-group button:focus:last-child{border-radius:0}
.maplibregl-ctrl.maplibregl-ctrl-attrib{font:.8125rem/1.4 Charter,'Bitstream Charter','Sitka Text',Cambria,'Noto Serif',Georgia,serif;color:#5d676b;background:rgba(246,242,234,.88)}
.maplibregl-ctrl-attrib a{color:#5d676b}
.maplibregl-ctrl-attrib.maplibregl-compact{border-radius:0}
.maplibregl-ctrl-attrib-button{border-radius:0;background-color:transparent}
.minimap-canvas.maplibregl-map{position:absolute;inset:0}`;

// Where the base map lives (TILES_URL); without an answer, style.json's default.
const config = await httpApi.getConfig().catch((err: unknown) => {
  console.warn('minimap config:', err);
  return null;
});

const css = document.createElement('style');
css.textContent = maplibreCss + OVERRIDES;
document.head.append(css);

type Props = { cell: string; surveyed: number; photos: number; focus: number; ratio: number; path?: string };

function feature(cell: string, props: Omit<Props, 'cell'>): Feature<Polygon, Props> {
  return { type: 'Feature', properties: { cell, ...props }, geometry: { type: 'Polygon', coordinates: [cellRing(cell)] } };
}

function boundsOf(features: Feature<Polygon>[]): LngLatBoundsLike {
  let w = 180, s = 90, e = -180, n = -90;
  for (const f of features) {
    for (const [lng, lat] of f.geometry.coordinates[0]) {
      w = Math.min(w, lng); e = Math.max(e, lng);
      s = Math.min(s, lat); n = Math.max(n, lat);
    }
  }
  return [[w, s], [e, n]];
}

function parseBBox(raw: string | undefined): BBox | null {
  const v = (raw ?? '').split(',').map(Number);
  if (v.length !== 4 || v.some((x) => !Number.isFinite(x))) return null;
  return { west: v[0], south: v[1], east: v[2], north: v[3] };
}

function createMap(el: HTMLElement, bounds: LngLatBoundsLike, maxZoom: number): MapLibreMap {
  const box = document.createElement('div');
  box.className = 'minimap-canvas';
  el.append(box);
  const map = new MapLibreMap({
    container: box,
    style: baseStyle(config?.tiles_url),
    bounds,
    fitBoundsOptions: { padding: 24, maxZoom },
    minZoom: 4.5,
    maxZoom: 17,
    scrollZoom: false, // never trap the page scroll
    dragRotate: false,
    pitchWithRotate: false,
    attributionControl: { compact: true },
  });
  map.touchZoomRotate.disableRotation();
  map.addControl(new NavigationControl({ showCompass: false }), 'top-right');
  map.on('error', (e) => console.warn('minimap:', e.error?.message ?? e));
  return map;
}

/** Remove the no-JS fallback once the map has drawn. */
function replaceFallback(el: HTMLElement, map: MapLibreMap): void {
  map.once('idle', () => {
    for (const child of Array.from(el.children)) if (!child.classList.contains('minimap-canvas')) child.remove();
  });
}

function linkPlots(map: MapLibreMap, layer: string): void {
  map.on('click', layer, (e: MapLayerMouseEvent) => {
    const { cell, path } = e.features?.[0]?.properties ?? {};
    if (typeof path === 'string' && path) location.href = path;
    else if (typeof cell === 'string') location.href = `/plot/${cell}`;
  });
  map.on('mouseenter', layer, () => (map.getCanvas().style.cursor = 'pointer'));
  map.on('mouseleave', layer, () => (map.getCanvas().style.cursor = ''));
}

function plotsMap(el: HTMLElement): void {
  const cells = (el.dataset.cells ?? '').split(',').filter(Boolean);
  if (cells.length === 0) return;
  // Without data-photos (a page cached from before), every plot counts as one photo.
  const photos = (el.dataset.photos ?? '').split(',').map((v) => Number(v || 1));
  const focus = el.dataset.focus ?? '';
  const paths = (el.dataset.paths ?? '').split(',');
  const features = cells.map((cell, i) =>
    feature(cell, {
      surveyed: photos[i] > 0 ? 1 : 0,
      photos: photos[i] ?? 1,
      focus: cell === focus ? 1 : 0,
      ratio: 0,
      path: paths[i] || undefined,
    }),
  );
  const fc: FeatureCollection<Polygon, Props> = { type: 'FeatureCollection', features };
  const map = createMap(el, boundsOf(features), focus ? 15 : 14);
  map.on('load', () => {
    map.addSource('cells', { type: 'geojson', data: fc });
    map.addLayer({
      id: 'cells-fill',
      type: 'fill',
      source: 'cells',
      paint: {
        'fill-color': ['step', ['get', 'photos'], GREEN_LIGHT, 2, GREEN_MID, 4, GREEN],
        'fill-opacity': ['case', ['==', ['get', 'surveyed'], 1], 0.7, 0.04],
      },
    });
    map.addLayer({
      id: 'cells-line',
      type: 'line',
      source: 'cells',
      paint: {
        'line-color': ['case', ['==', ['get', 'focus'], 1], SLATE, ['==', ['get', 'surveyed'], 1], GREEN, 'rgba(47, 58, 63, 0.45)'],
        'line-width': ['case', ['==', ['get', 'focus'], 1], 3, 1.2],
      },
    });
    linkPlots(map, 'cells-fill');
    replaceFallback(el, map);
  });
}

function coverageMap(el: HTMLElement): void {
  const b = parseBBox(el.dataset.bounds);
  if (!b) return;
  // Parents at res 6 unless the region is too large for one request series.
  const res: 5 | 6 = estimateCellCount(b, 6) > MAX_VIEW_CELLS / 4 ? 5 : 6;
  const parents = cellsForBBox(b, res);
  const map = createMap(el, [[b.west, b.south], [b.east, b.north]], 11);
  const loaded = new Promise<void>((resolve) => map.on('load', () => resolve()));
  const counts = Promise.all(chunk(parents, COVERAGE_BATCH).map((c) => httpApi.getCoverage(res, c))).then((r) => r.flat());
  Promise.all([loaded, counts])
    .then(([, cov]) => {
      const fc: FeatureCollection<Polygon, Props> = {
        type: 'FeatureCollection',
        features: cov
          .filter((c) => c.surveyed > 0)
          .map((c) => feature(c.cell, { surveyed: c.surveyed, photos: 0, focus: 0, ratio: c.total > 0 ? c.surveyed / c.total : 0 })),
      };
      map.addSource('coverage', { type: 'geojson', data: fc });
      map.addLayer({
        id: 'coverage-fill',
        type: 'fill',
        source: 'coverage',
        // Any surveyed parent is clearly visible; fuller parents are darker.
        paint: { 'fill-color': GREEN, 'fill-opacity': ['interpolate', ['linear'], ['get', 'ratio'], 0, 0.3, 0.1, 0.55, 1, 0.8] },
      });
      map.addLayer({ id: 'coverage-line', type: 'line', source: 'coverage', paint: { 'line-color': GREEN, 'line-width': 1 } });
      replaceFallback(el, map);
    })
    .catch((err: unknown) => console.warn('minimap coverage:', err));
}

for (const el of Array.from(document.querySelectorAll<HTMLElement>('[data-minimap]'))) {
  if (el.dataset.minimapReady) continue;
  el.dataset.minimapReady = '1';
  if (el.dataset.minimap === 'plots') plotsMap(el);
  else if (el.dataset.minimap === 'coverage') coverageMap(el);
}
