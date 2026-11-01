import 'maplibre-gl/dist/maplibre-gl.css';
import './styles.css';
import { addProtocol, Map as MapLibreMap, NavigationControl, setWorkerUrl } from 'maplibre-gl';
import workerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?url';
import { Protocol } from 'pmtiles';
import { cellToLatLng, getResolution, isValidCell, latLngToCell } from 'h3-js';
import { loadApi, MOCK, type UploadResult } from './api';
import { DataStore } from './data';
import { PLOT_RES, planView, type ViewPlan } from './grid';
import { addLayers, pulse, renderCoverage, renderPlots, setPlotState, showMode } from './layers';
import { Panel } from './panel';
import { baseStyle } from './style';

setWorkerUrl(workerUrl);

// Layout variants for design review (docs/design/round-3): ?layout=a|b|c,
// remembered for the session. a is the default until Adrian picks one.
const LAYOUTS = ['a', 'b', 'c'];
const asked = new URLSearchParams(location.search).get('layout');
let layout = 'a';
try {
  if (asked && LAYOUTS.includes(asked)) sessionStorage.setItem('map-layout', asked);
  layout = sessionStorage.getItem('map-layout') ?? layout;
} catch {
  if (asked && LAYOUTS.includes(asked)) layout = asked;
}
document.body.dataset.layout = layout;
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

function viewBBox() {
  const b = map.getBounds();
  return { west: b.getWest(), south: b.getSouth(), east: b.getEast(), north: b.getNorth() };
}

async function refresh(): Promise<void> {
  const gen = ++generation;
  const t0 = performance.now();
  plan = planView(viewBBox(), map.getZoom());
  showMode(map, plan.mode);

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

let debounce: number | undefined;
function scheduleRefresh(): void {
  window.clearTimeout(debounce);
  debounce = window.setTimeout(() => void refresh(), 200);
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
 * or a height along the bottom (the sheet on phones). Nothing where the
 * panel stands beside the map (layout b). Layout positions ignore the
 * panel's slide transform, so this is right while it is still moving in.
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
map.on('moveend', scheduleRefresh);

map.on('mousemove', (e) => {
  if (plan.mode !== 'plots') {
    map.getCanvas().style.cursor = plan.mode === 'coverage' ? 'zoom-in' : '';
    return;
  }
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
});

map.on('click', (e) => {
  if (plan.mode === 'coverage') {
    map.easeTo({ center: e.lngLat, zoom: Math.max(map.getZoom() + 3, 12) });
    return;
  }
  if (plan.mode !== 'plots') return;
  const cell = latLngToCell(e.lngLat.lat, e.lngLat.lng, PLOT_RES);
  select(cell);
  void panel.openPlot(cell);
  revealCell(cell);
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
