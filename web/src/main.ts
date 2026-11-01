import 'maplibre-gl/dist/maplibre-gl.css';
import './styles.css';
import { addProtocol, Map as MapLibreMap, NavigationControl, setWorkerUrl } from 'maplibre-gl';
import workerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?url';
import { Protocol } from 'pmtiles';
import { cellToBoundary, cellToLatLng, getHexagonAreaAvg, getResolution, isValidCell, latLngToCell, UNITS } from 'h3-js';
import { loadApi, MOCK, type UploadResult } from './api';
import { DataStore } from './data';
import { PLOT_RES, planView, type ViewPlan } from './grid';
import { addLayers, pulse, renderCoverage, renderPlots, setGroupState, setPlotState, showMode } from './layers';
import { Panel } from './panel';
import { baseStyle } from './style';

setWorkerUrl(workerUrl);

addProtocol('pmtiles', new Protocol().tile);

const api = await loadApi();
const data = new DataStore(api);
const statusEl = document.getElementById('map-status')!;

const map = new MapLibreMap({
  container: 'map',
  style: baseStyle(),
  center: [100.9, 13.2],
  zoom: 6,
  minZoom: 4.5,
  maxZoom: 18,
  maxBounds: [[88, -2], [114, 28]],
  hash: true,
  attributionControl: { compact: true },
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

function viewBBox() {
  const b = map.getBounds();
  return { west: b.getWest(), south: b.getSouth(), east: b.getEast(), north: b.getNorth() };
}

async function refresh(): Promise<void> {
  const gen = ++generation;
  const t0 = performance.now();
  plan = planView(viewBBox(), map.getZoom());
  showMode(map, plan.mode);
  setKey(plan);

  if (plan.mode === 'none') {
    setStatus('Cartograph covers Thailand for now.');
    return;
  }
  if (plan.mode === 'coverage') {
    const p = plan;
    renderCoverage(map, p.res, p.cells, data);
    setStatus('Zoom in to see individual plots');
    try {
      await data.ensureCoverage(p.res, p.cells);
      if (gen !== generation) return;
      renderCoverage(map, p.res, p.cells, data);
    } catch (err) {
      if (gen === generation) setStatus(`Could not load coverage: ${(err as Error).message}`, 'error');
    }
    return;
  }

  const p = plan;
  renderPlots(map, p.cells, data); // grid immediately, known plots coloured
  try {
    const { requests, requested } = await data.ensurePlots(p.cells);
    if (gen !== generation) return;
    const surveyed = renderPlots(map, p.cells, data);
    setStatus(`${surveyed} of ${p.cells.length} plots in view surveyed`);
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
  const next = planView(viewBBox(), map.getZoom());
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
map.on('move', onMove);
map.on('moveend', onMoveEnd);

/** The group of plots (coverage cell) under a point, if the view shows groups and it is one of them. */
function groupAt(lat: number, lng: number): string | null {
  if (plan.mode !== 'coverage') return null;
  const cell = latLngToCell(lat, lng, plan.res);
  return plan.cells.includes(cell) ? cell : null;
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
  const cell = latLngToCell(e.lngLat.lat, e.lngLat.lng, PLOT_RES);
  if (cell === hovered) return;
  if (hovered) setPlotState(map, hovered, { hover: false });
  hovered = cell;
  setPlotState(map, cell, { hover: true });
  map.getCanvas().style.cursor = 'pointer';
});
map.getCanvas().addEventListener('mouseleave', () => {
  if (hovered) setPlotState(map, hovered, { hover: false });
  hovered = null;
  hoverGroup(null);
});

/** Zoom so that a group of plots fills the view, clear of the labels at the edges. */
function zoomToGroup(cell: string): void {
  let w = 180, s = 90, e = -180, n = -90;
  for (const [lat, lng] of cellToBoundary(cell)) {
    w = Math.min(w, lng); e = Math.max(e, lng);
    s = Math.min(s, lat); n = Math.max(n, lat);
  }
  hoverGroup(null);
  const pad = Math.round(Math.min(innerWidth, innerHeight) * 0.08);
  map.fitBounds([[w, s], [e, n]], { padding: pad, duration: 700 });
}

map.on('click', (e) => {
  if (plan.mode === 'coverage') {
    const group = groupAt(e.lngLat.lat, e.lngLat.lng);
    if (group) zoomToGroup(group);
    return;
  }
  if (plan.mode !== 'plots') return;
  const cell = latLngToCell(e.lngLat.lat, e.lngLat.lng, PLOT_RES);
  select(cell);
  void panel.openPlot(cell);
  revealCell(cell);
});

// On phones the key is the status line; its legend opens on a tap.
const keyToggle = document.getElementById('key-toggle')!;
keyToggle.addEventListener('click', () => {
  keyToggle.setAttribute('aria-expanded', String(keyEl.classList.toggle('open')));
});

document.getElementById('open-upload')!.addEventListener('click', () => {
  select(null);
  panel.openUpload();
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
