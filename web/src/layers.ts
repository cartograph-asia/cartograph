// Plot and coverage layers on top of the base style.
import type { ExpressionSpecification, GeoJSONSource, Map as MapLibreMap } from 'maplibre-gl';
import type { FeatureCollection, Polygon } from 'geojson';
import { cellRing } from './grid';
import type { DataStore } from './data';

// Greens from the logo palette (docs/HOME_BRIEF.md), light to dark:
// #A8CDB3, #5FA57A and #2E7D4F, written as rgba() below where the alpha
// varies per cell.
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

// What differs from cell to cell (share surveyed, photo count, hover) is in
// the colours, with their alpha; the layers' own opacity is a plain number.
// A change of that number is cheap and MapLibre eases it by itself, which
// gives the cross-fade between single plots and groups, and between the two
// group sizes. (Writing the fade into data-driven opacity expressions made
// MapLibre rebuild every cell on every frame, and the old cells lingered.)

/** Cross-fade length when the view changes between single plots and groups, or between group sizes. */
const FADE_MS = 350;
const FADE = { duration: FADE_MS, delay: 0 };

const green = (a: ExpressionSpecification | number): ExpressionSpecification => ['rgba', 46, 125, 79, a];
const ink = (a: number): ExpressionSpecification => ['rgba', 47, 58, 63, a];
const HOVER: ExpressionSpecification = ['boolean', ['feature-state', 'hover'], false];
const SURVEYED: ExpressionSpecification = ['>', ['get', 'surveyed'], 0];

function addCoverageLayers(map: MapLibreMap, source: string, fill: string, line: string): void {
  map.addLayer({
    id: fill,
    type: 'fill',
    source,
    paint: {
      // Brief: opacity proportional to surveyed/total, 0 -> 0.05, 1 -> 0.6. Untouched parents stay clear.
      'fill-color': ['case', SURVEYED, green(['interpolate', ['linear'], ['get', 'ratio'], 0, 0.05, 1, 0.6]), green(0)],
      'fill-opacity': 0,
      'fill-opacity-transition': FADE,
    },
  });
  // The group under the mouse is outlined in ink: a click zooms to it.
  // Blank parent cells only from zoom 8: further out their outlines tile the
  // whole view, sea included, and read as a pattern, not data.
  const lineAt = (blank: number): ExpressionSpecification => ['case', HOVER, ink(1), SURVEYED, green(0.75), ink(0.32 * blank)];
  map.addLayer({
    id: line,
    type: 'line',
    source,
    paint: {
      'line-color': ['interpolate', ['linear'], ['zoom'], 6.5, lineAt(0), 8, lineAt(0.45)],
      'line-width': ['case', HOVER, 2, SURVEYED, 1.2, 0.5],
      'line-opacity': 0,
      'line-opacity-transition': FADE,
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

  // A plot's fill: 0.7 when surveyed, plus the pulse after an upload.
  const alpha: ExpressionSpecification = [
    'min',
    1,
    ['+', ['case', ['==', ['get', 'surveyed'], 1], 0.7, 0], ['*', 0.45, ['coalesce', ['feature-state', 'pulse'], 0]]],
  ];
  map.addLayer({
    id: PLOT_FILL,
    type: 'fill',
    source: 'plots',
    paint: {
      // Three steps by photo count: 1 photo light, 2–3 mid, 4 or more dark.
      'fill-color': ['step', ['get', 'photo_count'], ['rgba', 168, 205, 179, alpha], 2, ['rgba', 95, 165, 122, alpha], 4, green(alpha)],
      'fill-opacity': 0,
      'fill-opacity-transition': FADE,
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
        HOVER, '#2f3a3f',
        ['==', ['get', 'surveyed'], 1], GREEN,
        GRID,
      ],
      'line-width': [
        'case',
        ['boolean', ['feature-state', 'selected'], false], 2.5,
        HOVER, 1.8,
        0.6,
      ],
      'line-opacity': 0,
      'line-opacity-transition': FADE,
    },
  });
}

type Group = 'plots' | 'coverage' | 'coverage-prev';
const LAYERS: Record<Group, [fill: string, line: string]> = {
  plots: [PLOT_FILL, PLOT_LINE],
  coverage: [COV_FILL, COV_LINE],
  'coverage-prev': [COV_PREV_FILL, COV_PREV_LINE],
};

/** Ease a group's opacity to v over FADE_MS, or set it at once. */
function setOpacity(map: MapLibreMap, group: Group, v: number, ease = true): void {
  const [fill, line] = LAYERS[group];
  const t = ease ? FADE : { duration: 0, delay: 0 };
  map.setPaintProperty(fill, 'fill-opacity-transition', t);
  map.setPaintProperty(line, 'line-opacity-transition', t);
  map.setPaintProperty(fill, 'fill-opacity', v);
  map.setPaintProperty(line, 'line-opacity', v);
}

type Mode = 'plots' | 'coverage' | 'none';
let shownMode: Mode = 'none';

/** Show single plots, groups or nothing; a change of mode cross-fades. */
export function showMode(map: MapLibreMap, mode: Mode): void {
  if (mode === shownMode) return;
  shownMode = mode;
  setOpacity(map, 'plots', mode === 'plots' ? 1 : 0);
  setOpacity(map, 'coverage', mode === 'coverage' ? 1 : 0);
  // Groups coming back after single plots fade in on their own; there is
  // no group size to fade from.
  if (mode === 'coverage') coverageRes = null;
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
    // The old size stands in the prev layers at full opacity and fades out;
    // the new size starts clear and fades in.
    (map.getSource('coverage-prev') as GeoJSONSource).setData(coverageData);
    setOpacity(map, 'coverage-prev', 1, false);
    setOpacity(map, 'coverage', 0, false);
    requestAnimationFrame(() => {
      setOpacity(map, 'coverage-prev', 0);
      setOpacity(map, 'coverage', 1);
    });
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
