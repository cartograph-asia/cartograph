// Plot and coverage layers on top of the base style.
import type { ExpressionSpecification, GeoJSONSource, Map as MapLibreMap } from 'maplibre-gl';
import type { FeatureCollection, Polygon } from 'geojson';
import { cellRing } from './grid';
import type { DataStore } from './data';

// Greens from the logo palette (docs/HOME_BRIEF.md), light to dark.
const GREEN_LIGHT = '#A8CDB3';
const GREEN_MID = '#5FA57A';
const GREEN = '#2E7D4F';
const GRID = 'rgba(47, 58, 63, 0.32)'; // the pages' --ink-line

export const PLOT_FILL = 'plots-fill';
const PLOT_LINE = 'plots-line';
const COV_FILL = 'coverage-fill';
const COV_LINE = 'coverage-line';
const COV_PREV_FILL = 'coverage-prev-fill';
const COV_PREV_LINE = 'coverage-prev-line';

const ringCache = new Map<string, number[][]>();
function ring(cell: string): number[][] {
  let r = ringCache.get(cell);
  if (!r) {
    r = cellRing(cell);
    ringCache.set(cell, r);
  }
  return r;
}

type FC = FeatureCollection<Polygon>;
const empty = (): FC => ({ type: 'FeatureCollection', features: [] });

// Opacity expressions take a fade factor f (0–1) for the cross-fade between
// single plots and groups, and between the two group sizes. MapLibre has no
// layer opacity, and a zoom curve must stay the outermost expression, so f
// goes into every output.

function coverageFillOpacity(f: number): ExpressionSpecification {
  // Brief: opacity proportional to surveyed/total, 0 -> 0.05, 1 -> 0.6. Untouched parents stay clear.
  return ['case', ['>', ['get', 'surveyed'], 0], ['interpolate', ['linear'], ['get', 'ratio'], 0, 0.05 * f, 1, 0.6 * f], 0];
}

function coverageLineOpacity(f: number): ExpressionSpecification {
  const at = (blank: number): ExpressionSpecification => [
    'case',
    ['boolean', ['feature-state', 'hover'], false], f,
    ['>', ['get', 'surveyed'], 0], 0.75 * f,
    blank * f,
  ];
  // Blank parent cells only from zoom 8: further out their outlines
  // tile the whole view, sea included, and read as a pattern, not data.
  return ['interpolate', ['linear'], ['zoom'], 6.5, at(0), 8, at(0.45)];
}

function plotFillOpacity(f: number): ExpressionSpecification {
  return [
    '*',
    f,
    ['min', 1, ['+', ['case', ['==', ['get', 'surveyed'], 1], 0.7, 0], ['*', 0.45, ['coalesce', ['feature-state', 'pulse'], 0]]]],
  ];
}

function addCoverageLayers(map: MapLibreMap, source: string, fill: string, line: string): void {
  map.addLayer({
    id: fill,
    type: 'fill',
    source,
    paint: { 'fill-color': GREEN, 'fill-opacity': coverageFillOpacity(1) },
  });
  map.addLayer({
    id: line,
    type: 'line',
    source,
    paint: {
      // The group under the mouse is outlined in ink: a click zooms to it.
      'line-color': [
        'case',
        ['boolean', ['feature-state', 'hover'], false], '#2f3a3f',
        ['>', ['get', 'surveyed'], 0], GREEN,
        GRID,
      ],
      'line-opacity': coverageLineOpacity(1),
      'line-width': [
        'case',
        ['boolean', ['feature-state', 'hover'], false], 2,
        ['>', ['get', 'surveyed'], 0], 1.2,
        0.5,
      ],
    },
  });
}

export function addLayers(map: MapLibreMap): void {
  map.addSource('coverage', { type: 'geojson', data: empty(), promoteId: 'cell' });
  // The groups of the previous size while they fade out (see renderCoverage).
  map.addSource('coverage-prev', { type: 'geojson', data: empty() });
  map.addSource('plots', { type: 'geojson', data: empty(), promoteId: 'cell' });

  addCoverageLayers(map, 'coverage-prev', COV_PREV_FILL, COV_PREV_LINE);
  addCoverageLayers(map, 'coverage', COV_FILL, COV_LINE);

  map.addLayer({
    id: PLOT_FILL,
    type: 'fill',
    source: 'plots',
    paint: {
      // Three steps by photo count: 1 photo light, 2–3 mid, 4 or more dark.
      'fill-color': ['step', ['get', 'photo_count'], GREEN_LIGHT, 2, GREEN_MID, 4, GREEN],
      'fill-opacity': plotFillOpacity(1),
    },
  });
  map.addLayer({
    id: PLOT_LINE,
    type: 'line',
    source: 'plots',
    paint: {
      'line-color': [
        'case',
        ['boolean', ['feature-state', 'selected'], false], '#1d5e3a',
        ['boolean', ['feature-state', 'hover'], false], '#2f3a3f',
        ['==', ['get', 'surveyed'], 1], GREEN,
        GRID,
      ],
      'line-width': [
        'case',
        ['boolean', ['feature-state', 'selected'], false], 2.5,
        ['boolean', ['feature-state', 'hover'], false], 1.8,
        0.6,
      ],
    },
  });
  setVisible(map, [PLOT_FILL, PLOT_LINE, COV_FILL, COV_LINE, COV_PREV_FILL, COV_PREV_LINE], false);
}

function setVisible(map: MapLibreMap, ids: string[], visible: boolean): void {
  for (const id of ids) map.setLayoutProperty(id, 'visibility', visible ? 'visible' : 'none');
}

type Group = 'plots' | 'coverage' | 'coverage-prev';
const LAYERS: Record<Group, string[]> = {
  plots: [PLOT_FILL, PLOT_LINE],
  coverage: [COV_FILL, COV_LINE],
  'coverage-prev': [COV_PREV_FILL, COV_PREV_LINE],
};

function setFade(map: MapLibreMap, group: Group, f: number): void {
  if (group === 'plots') {
    map.setPaintProperty(PLOT_FILL, 'fill-opacity', plotFillOpacity(f));
    map.setPaintProperty(PLOT_LINE, 'line-opacity', f);
    return;
  }
  const [fill, line] = LAYERS[group];
  map.setPaintProperty(fill, 'fill-opacity', coverageFillOpacity(f));
  map.setPaintProperty(line, 'line-opacity', coverageLineOpacity(f));
}

/** Cross-fade length when the view changes between single plots and groups, or between group sizes. */
const FADE_MS = 350;
let fadeFrame = 0;

/** Fade `inGroup` in and `outGroup` out; a new fade replaces one still running. */
function crossFade(map: MapLibreMap, inGroup: Group | null, outGroup: Group | null): void {
  cancelAnimationFrame(fadeFrame);
  for (const g of Object.keys(LAYERS) as Group[]) {
    if (g !== inGroup && g !== outGroup) setVisible(map, LAYERS[g], false);
  }
  if (inGroup) {
    setFade(map, inGroup, 0);
    setVisible(map, LAYERS[inGroup], true);
  }
  const start = performance.now();
  const step = (now: number) => {
    const t = Math.min(1, (now - start) / FADE_MS);
    if (inGroup) setFade(map, inGroup, t);
    if (outGroup) setFade(map, outGroup, 1 - t);
    if (t < 1) fadeFrame = requestAnimationFrame(step);
    else if (outGroup) setVisible(map, LAYERS[outGroup], false);
  };
  fadeFrame = requestAnimationFrame(step);
}

type Mode = 'plots' | 'coverage' | 'none';
let shownMode: Mode = 'none';

/** Show single plots, groups or nothing; a change of mode cross-fades. */
export function showMode(map: MapLibreMap, mode: Mode): void {
  if (mode === shownMode) return;
  const from = shownMode;
  shownMode = mode;
  // Groups coming back after single plots fade in on their own; there is
  // no group size to fade from.
  if (mode === 'coverage') coverageRes = null;
  crossFade(map, mode === 'none' ? null : mode, from === 'none' ? null : from);
}

/** Draw all cells in view; unknown cells are drawn blank until their data arrives. */
export function renderPlots(map: MapLibreMap, cells: string[], data: DataStore): number {
  let surveyed = 0;
  const fc: FC = {
    type: 'FeatureCollection',
    features: cells.map((cell) => {
      const p = data.plot(cell);
      if (p) surveyed++;
      return {
        type: 'Feature',
        properties: { cell, surveyed: p ? 1 : 0, photo_count: p?.photo_count ?? 0 },
        geometry: { type: 'Polygon', coordinates: [ring(cell)] },
      };
    }),
  };
  (map.getSource('plots') as GeoJSONSource).setData(fc);
  return surveyed;
}

let coverageRes: 5 | 6 | null = null;
let coverageData: FC = empty();

/**
 * Draw the groups in view. When their size changes (zoom 7), the groups of
 * the old size move to the coverage-prev layers and fade out while the new
 * ones fade in.
 */
export function renderCoverage(map: MapLibreMap, res: 5 | 6, cells: string[], data: DataStore): void {
  const fc: FC = {
    type: 'FeatureCollection',
    features: cells.map((cell) => {
      const c = data.coverageOf(res, cell);
      const surveyed = c?.surveyed ?? 0;
      const ratio = c && c.total > 0 ? surveyed / c.total : 0;
      return {
        type: 'Feature',
        properties: { cell, surveyed, ratio },
        geometry: { type: 'Polygon', coordinates: [ring(cell)] },
      };
    }),
  };
  if (coverageRes !== null && res !== coverageRes && shownMode === 'coverage') {
    (map.getSource('coverage-prev') as GeoJSONSource).setData(coverageData);
    setFade(map, 'coverage-prev', 1);
    setVisible(map, LAYERS['coverage-prev'], true);
    crossFade(map, 'coverage', 'coverage-prev');
  }
  coverageRes = res;
  coverageData = fc;
  (map.getSource('coverage') as GeoJSONSource).setData(fc);
}

export function setPlotState(map: MapLibreMap, cell: string, state: Record<string, unknown>): void {
  map.setFeatureState({ source: 'plots', id: cell }, state);
}

export function setGroupState(map: MapLibreMap, cell: string, state: Record<string, unknown>): void {
  map.setFeatureState({ source: 'coverage', id: cell }, state);
}

/** Brief fill pulse on a freshly surveyed plot. */
export function pulse(map: MapLibreMap, cell: string, ms = 1600): void {
  const start = performance.now();
  const step = (now: number) => {
    const t = Math.min(1, (now - start) / ms);
    // two soft beats fading out
    const v = (1 - t) * (0.5 + 0.5 * Math.cos(t * Math.PI * 4));
    setPlotState(map, cell, { pulse: t < 1 ? v : 0 });
    if (t < 1) requestAnimationFrame(step);
  };
  requestAnimationFrame(step);
}
