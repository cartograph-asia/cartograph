import { addProtocol, AttributionControl, Map as MapLibreMap, MercatorCoordinate, NavigationControl, setWorkerUrl } from 'maplibre-gl';
import type { CameraOptions, LngLatBoundsLike } from 'maplibre-gl';
import workerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?url';
import { Protocol } from 'pmtiles';
import { cellToBoundary, cellToLatLng, getHexagonAreaAvg, getResolution, isValidCell, latLngToCell, UNITS } from 'h3-js';
import { loadApi, MOCK, type UploadResult } from './api';
import { DataStore } from './data';
import { PLOT_MIN_ZOOM, PLOT_RES, THAILAND_BBOX, planView, type BBox, type ViewPlan } from './grid';
import { addLayers, pulse, renderCoverage, renderPlots, setGroupState, setPlotState, showMode } from './layers';
import { Panel } from './panel';
import { loadBaseStyle } from './style';

setWorkerUrl(workerUrl);

const protocol = new Protocol();
addProtocol('pmtiles', protocol.tile);

const api = await loadApi();
// Where the base map lives (TILES_URL); without an answer, style.json's default.
const config = await api.getConfig().catch((err: unknown) => {
  console.warn('config:', err);
  return null;
});
const { style } = await loadBaseStyle(config?.tiles_url, protocol);
const data = new DataStore(api);
const statusEl = document.getElementById('map-status')!;

/**
 * The countries the map is about, Southeast and East Asia as the base map
 * was built: the view a visit without a position in the URL opens on. It is
 * fixed here rather than taken from the archive, so that a larger archive
 * does not move it.
 */
const HOME: LngLatBoundsLike = [[88, -12], [150, 55]];

// A visit with a position in the URL (#zoom/lat/lng, written by the map)
// keeps it; so does the cell from ?cell=, which load() flies to.
const linked = /^#\d/.test(location.hash) || new URLSearchParams(location.search).has('cell');

const map = new MapLibreMap({
  container: 'map',
  style,
  center: [119, 21.5],
  zoom: 2.5,
  maxZoom: 18,
  hash: true,
  // A flat map: plots are drawn north up, and a turned or tilted view only
  // showed the base map's edges (Adrian). No rotation, no pitch, no compass.
  dragRotate: false,
  pitchWithRotate: false,
  touchPitch: false,
  maxPitch: 0,
  attributionControl: false, // added below, to start it collapsed
});
map.touchZoomRotate.disableRotation();
map.keyboard.disableRotation();
// The credits start collapsed to their (i) button, where MapLibre would show
// them until the first move. A tap on the button opens them (OSM's attribution
// guidance accepts a visible, clickable (i)).
map.addControl(new AttributionControl({ compact: true }), 'bottom-right');
// MapLibre opens the control again while the style and its sources arrive, so
// it is collapsed once the map has loaded, not when it is added.
map.once('load', () => {
  const el = map.getContainer().querySelector('.maplibregl-ctrl-attrib');
  el?.classList.remove('maplibregl-compact-show');
  el?.removeAttribute('open');
});
map.addControl(new NavigationControl({ showCompass: false }), 'top-right');
map.on('error', (e) => console.warn('map error:', e.error?.message ?? e));

let plan: ViewPlan = { mode: 'none' };
let generation = 0;
let hovered: string | null = null;
let selected: string | null = null;
let pendingPulse: string | null = null;

function setStatus(text: string, kind: 'info' | 'error' = 'info'): void {
  statusEl.textContent = text;
  statusEl.dataset.kind = kind;
  statusEl.hidden = !text;
}

const keyEl = document.querySelector<HTMLElement>('.key')!;
const groupSizeEl = document.getElementById('group-size')!;

/** The key explains what the view shows: single plots, or groups of them at a coarser resolution. */
function setKey(next: ViewPlan): void {
  if (next.mode !== 'coverage') {
    keyEl.dataset.scale = 'plots';
    return;
  }
  keyEl.dataset.scale = 'groups';
  const plots = 7 ** (PLOT_RES - next.res);
  const km2 = Math.round(getHexagonAreaAvg(next.res, UNITS.km2));
  groupSizeEl.textContent = `${plots} plots, about ${km2.toLocaleString('en')} km²`;
}

/** Where plots can be surveyed, from the server; Thailand until it says. */
const surveyable: BBox = config?.surveyable
  ? { west: config.surveyable[0], south: config.surveyable[1], east: config.surveyable[2], north: config.surveyable[3] }
  : THAILAND_BBOX;

const NOT_YET = 'Plots here cannot be surveyed yet.';

function viewBBox() {
  const b = map.getBounds();
  return { west: b.getWest(), south: b.getSouth(), east: b.getEast(), north: b.getNorth() };
}

async function refresh(): Promise<void> {
  const gen = ++generation;
  const t0 = performance.now();
  plan = planView(viewBBox(), map.getZoom(), surveyable);
  showMode(map, plan.mode);
  setKey(plan);

  if (plan.mode === 'none') {
    setStatus(NOT_YET);
    return;
  }
  if (plan.mode === 'coverage') {
    const p = plan;
    renderCoverage(map, p.res, p.cells, data);
    setStatus(p.dataCells.length ? 'Zoom in to see individual plots' : NOT_YET);
    if (!p.dataCells.length) return;
    try {
      await data.ensureCoverage(p.res, p.dataCells);
      if (gen !== generation) return;
      renderCoverage(map, p.res, p.cells, data);
    } catch (err) {
      if (gen === generation) setStatus(`Could not load coverage: ${(err as Error).message}`, 'error');
    }
    return;
  }

  const p = plan;
  renderPlots(map, p.cells, data); // grid immediately, known plots coloured
  if (!p.dataCells.length) {
    setStatus(NOT_YET);
    return;
  }
  try {
    const { requests, requested } = await data.ensurePlots(p.dataCells);
    if (gen !== generation) return;
    const surveyed = renderPlots(map, p.cells, data);
    setStatus(`${surveyed} of ${p.dataCells.length} plots in view surveyed`);
    if (import.meta.env.DEV) {
      console.debug(
        `plots: ${p.cells.length} cells in view, ${requested} fetched in ${requests} request(s), ` +
          `${(performance.now() - t0).toFixed(0)} ms`,
      );
    }
    if (pendingPulse && p.cells.includes(pendingPulse)) {
      pulse(map, pendingPulse);
      pendingPulse = null;
    }
  } catch (err) {
    if (gen === generation) setStatus(`Could not load plots: ${(err as Error).message}`, 'error');
  }
}

/**
 * Draw the grid for the current view from what is already loaded, without
 * requests. Runs every MOVE_REDRAW_MS while the map moves, so that zooming
 * and panning do not leave an empty margin until the movement ends.
 */
function redraw(): void {
  const t0 = performance.now();
  const next = planView(viewBBox(), map.getZoom(), surveyable);
  showMode(map, next.mode);
  setKey(next);
  if (next.mode === 'coverage') renderCoverage(map, next.res, next.cells, data);
  else if (next.mode === 'plots') renderPlots(map, next.cells, data);
  if (import.meta.env.DEV && next.mode !== 'none') {
    console.debug(`redraw: ${next.cells.length} cells, ${(performance.now() - t0).toFixed(0)} ms`);
  }
}

const MOVE_REDRAW_MS = 150;
let moveTimer: number | undefined;
// The plot and coverage layers exist from 'load' on; the map moves before
// that (the hash, the first resize), and those moves are ignored.
let layersReady = false;
function onMove(): void {
  if (!layersReady || moveTimer !== undefined) return;
  moveTimer = window.setTimeout(() => {
    moveTimer = undefined;
    redraw();
  }, MOVE_REDRAW_MS);
}

/** The movement ended: draw at once and fetch what the view lacks. */
function onMoveEnd(): void {
  if (!layersReady) return;
  window.clearTimeout(moveTimer);
  moveTimer = undefined;
  void refresh();
}

function select(cell: string | null): void {
  if (selected) setPlotState(map, selected, { selected: false });
  selected = cell;
  if (cell) setPlotState(map, cell, { selected: true });
}

const panel = new Panel(api, {
  onClose: () => select(null),
  onUploaded: (r: UploadResult) => onUploaded(r),
});

const panelEl = document.getElementById('panel')!;

/**
 * How much of the map the open panel covers: a width along the right edge,
 * or a height along the bottom (the sheet on phones). Layout positions
 * ignore the panel's slide transform, so this is right while it is still
 * moving in.
 */
function panelCover(): { x: number; y: number } {
  const m = map.getContainer().getBoundingClientRect();
  const left = panelEl.offsetLeft;
  const top = panelEl.offsetTop;
  const w = Math.max(0, Math.min(m.right, left + panelEl.offsetWidth) - Math.max(m.left, left));
  const h = Math.max(0, Math.min(m.bottom, top + panelEl.offsetHeight) - Math.max(m.top, top));
  if (!w || !h) return { x: 0, y: 0 };
  return w >= m.width - 1 ? { x: 0, y: h } : { x: w, y: 0 };
}

/** Offset that centres a point in the part of the map the open panel does not cover. */
function panelOffset(): [number, number] {
  const c = panelCover();
  return [-c.x / 2, -c.y / 2];
}

/** Move the map only if the cell would end up under the panel or off-screen. */
function revealCell(cell: string): void {
  const [lat, lng] = cellToLatLng(cell);
  const pt = map.project([lng, lat]);
  const margin = 48;
  const c = panelCover();
  const box = map.getContainer();
  const freeW = box.clientWidth - c.x;
  const freeH = box.clientHeight - c.y;
  if (pt.x < margin || pt.y < margin || pt.x > freeW - margin || pt.y > freeH - margin) {
    map.easeTo({ center: [lng, lat], offset: panelOffset() });
  }
}

function onUploaded(r: UploadResult): void {
  data.invalidate(r.cell);
  pendingPulse = r.cell;
  select(r.cell);
  void panel.openPlot(r.cell, false);
  const [lat, lng] = cellToLatLng(r.cell);
  const visible = plan.mode === 'plots' && map.getBounds().contains([lng, lat]);
  if (visible) {
    void refresh();
    revealCell(r.cell);
  } else {
    map.flyTo({ center: [lng, lat], zoom: Math.max(map.getZoom(), 14), offset: panelOffset(), essential: true });
  }
}

/** The plot from /map?cell=… (the "Survey this plot" link on plot pages), if valid. */
function linkedCell(): string | null {
  const cell = new URLSearchParams(location.search).get('cell')?.toLowerCase() ?? null;
  return cell && isValidCell(cell) && getResolution(cell) === PLOT_RES ? cell : null;
}

map.on('load', () => {
  addLayers(map);
  layersReady = true;
  const cell = linkedCell();
  if (cell) {
    // Drop the cell so that a reload keeps the user's own view from the hash.
    const q = new URLSearchParams(location.search);
    q.delete('cell');
    history.replaceState(null, '', location.pathname + (q.size ? `?${q}` : '') + location.hash);
    const [lat, lng] = cellToLatLng(cell);
    select(cell);
    void panel.openPlot(cell);
    map.jumpTo({ center: [lng, lat], zoom: Math.max(map.getZoom(), 15) });
    map.panBy(panelOffset().map((v) => -v) as [number, number], { duration: 0 });
  }
  void refresh();
});
// The key matches the starting zoom before the first tiles are drawn.
setKey(planView(viewBBox(), map.getZoom(), surveyable));

/**
 * The view that shows HOME whole and centred in the part of the map the
 * labels leave free: below the masthead and above the key. On a low window
 * (a phone turned sideways) that band would be a sliver, so there the
 * countries may run under the key.
 */
function homeCamera(): CameraOptions | undefined {
  const box = map.getContainer().getBoundingClientRect();
  const key = keyEl.getBoundingClientRect();
  const mast = document.querySelector('.mast')!.getBoundingClientRect();
  const gap = box.width < 768 ? 12 : 24;
  const attribution = 28;
  const top = mast.bottom - box.top + gap;
  let bottom = box.bottom - key.top + gap;
  if (box.height - top - bottom < box.height / 2) bottom = attribution + gap;
  if (box.height - top - bottom < box.height / 3 || box.width <= 2 * gap) return undefined;
  return map.cameraForBounds(HOME, { padding: { top, bottom, left: gap, right: gap } });
}

/**
 * Air around the countries: the map zooms out half a step beyond the home
 * view and no further, and pans only as far as that view reaches. Beyond the
 * archive it shows the paper margin (style.ts), so no edge of the base map
 * is ever in sight. Recomputed when the window changes size or turns.
 */
function frame(): CameraOptions | undefined {
  const cam = homeCamera();
  if (!cam?.center || cam.zoom === undefined) return undefined;
  const minZoom = Math.max(0, cam.zoom - 0.5);
  const c = MercatorCoordinate.fromLngLat(cam.center);
  const world = 512 * 2 ** minZoom;
  const dx = map.getContainer().clientWidth / 2 / world;
  const dy = map.getContainer().clientHeight / 2 / world;
  const sw = new MercatorCoordinate(c.x - dx, Math.min(1, c.y + dy)).toLngLat();
  const ne = new MercatorCoordinate(c.x + dx, Math.max(0, c.y - dy)).toLngLat();
  map.setMinZoom(minZoom);
  map.setMaxBounds([sw, ne]);
  return cam;
}

const home = frame();
if (home && !linked) map.jumpTo(home);
let frameTimer: number | undefined;
window.addEventListener('resize', () => {
  clearTimeout(frameTimer);
  frameTimer = window.setTimeout(frame, 200);
});

map.on('move', onMove);
map.on('moveend', onMoveEnd);

/**
 * The group of plots (coverage cell) under a point, if the view shows groups
 * and that group has photos. Blank groups are not clickable: zooming into
 * them shows nothing new (Adrian).
 */
function groupAt(lat: number, lng: number): string | null {
  if (plan.mode !== 'coverage') return null;
  const cell = latLngToCell(lat, lng, plan.res);
  return (data.coverageOf(plan.res, cell)?.surveyed ?? 0) > 0 ? cell : null;
}

/** The plot under a point, if the view shows single plots and it has photos (Adrian: only those are clickable). */
function surveyedPlotAt(lat: number, lng: number): string | null {
  if (plan.mode !== 'plots') return null;
  const cell = latLngToCell(lat, lng, PLOT_RES);
  return data.plot(cell) ? cell : null;
}

let hoveredGroup: string | null = null;
function hoverGroup(cell: string | null): void {
  if (cell === hoveredGroup) return;
  if (hoveredGroup) setGroupState(map, hoveredGroup, { hover: false });
  hoveredGroup = cell;
  if (cell) setGroupState(map, cell, { hover: true });
}

map.on('mousemove', (e) => {
  if (plan.mode !== 'plots') {
    // Groups: outline the one under the mouse; a click zooms to it.
    const group = groupAt(e.lngLat.lat, e.lngLat.lng);
    hoverGroup(group);
    map.getCanvas().style.cursor = group ? 'pointer' : '';
    return;
  }
  hoverGroup(null);
  // Only plots with photos react; a blank plot is surveyed with "Upload a photo".
  const cell = surveyedPlotAt(e.lngLat.lat, e.lngLat.lng);
  map.getCanvas().style.cursor = cell ? 'pointer' : '';
  if (cell === hovered) return;
  if (hovered) setPlotState(map, hovered, { hover: false });
  hovered = cell;
  if (cell) setPlotState(map, cell, { hover: true });
});
map.getCanvas().addEventListener('mouseleave', () => {
  if (hovered) setPlotState(map, hovered, { hover: false });
  hovered = null;
  hoverGroup(null);
});

/** Zoom so that a group of plots fills the view, clear of the labels at the edges, and at least to single plots. */
function zoomToGroup(cell: string): void {
  let w = 180, s = 90, e = -180, n = -90;
  for (const [lat, lng] of cellToBoundary(cell)) {
    w = Math.min(w, lng); e = Math.max(e, lng);
    s = Math.min(s, lat); n = Math.max(n, lat);
  }
  hoverGroup(null);
  const pad = Math.round(Math.min(innerWidth, innerHeight) * 0.08);
  const cam = map.cameraForBounds([[w, s], [e, n]], { padding: pad });
  if (!cam) return;
  // Always land on single plots: on a phone a large group (343 plots) fits
  // only at about zoom 9, where the map would still show groups.
  map.easeTo({ ...cam, zoom: Math.max(cam.zoom ?? 0, PLOT_MIN_ZOOM), duration: 700 });
}

map.on('click', (e) => {
  if (plan.mode === 'coverage') {
    const group = groupAt(e.lngLat.lat, e.lngLat.lng);
    if (group) zoomToGroup(group);
    return;
  }
  const cell = surveyedPlotAt(e.lngLat.lat, e.lngLat.lng);
  if (!cell) return;
  select(cell);
  void panel.openPlot(cell);
  revealCell(cell);
});

// On phones the key is the status line; its legend opens on a tap.
const keyToggle = document.getElementById('key-toggle')!;
keyToggle.addEventListener('click', () => {
  keyToggle.setAttribute('aria-expanded', String(keyEl.classList.toggle('open')));
});

document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape' && panel.escape()) e.preventDefault();
});

// The closed panel parks off-screen to the right on desktop and below on
// mobile; without this, crossing the breakpoint animates it across the map.
let resizeTimer: number | undefined;
window.addEventListener('resize', () => {
  document.body.classList.add('resizing');
  clearTimeout(resizeTimer);
  resizeTimer = window.setTimeout(() => document.body.classList.remove('resizing'), 150);
});

if (MOCK) document.body.classList.add('mock');
