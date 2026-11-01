// Plot and coverage layers on top of the base style.
import type { GeoJSONSource, Map as MapLibreMap } from 'maplibre-gl';
import type { FeatureCollection, Polygon } from 'geojson';
import { cellRing } from './grid';
import type { DataStore } from './data';

const GREEN = '#2e8b57';
const GRID = 'rgba(55, 65, 75, 0.32)';

export const PLOT_FILL = 'plots-fill';
const PLOT_LINE = 'plots-line';
const COV_FILL = 'coverage-fill';
const COV_LINE = 'coverage-line';

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

export function addLayers(map: MapLibreMap): void {
  map.addSource('coverage', { type: 'geojson', data: empty(), promoteId: 'cell' });
  map.addSource('plots', { type: 'geojson', data: empty(), promoteId: 'cell' });

  map.addLayer({
    id: COV_FILL,
    type: 'fill',
    source: 'coverage',
    paint: {
      'fill-color': GREEN,
      // Brief: opacity proportional to surveyed/total, 0 -> 0.05, 1 -> 0.6. Untouched parents stay clear.
      'fill-opacity': [
        'case',
        ['>', ['get', 'surveyed'], 0],
        ['interpolate', ['linear'], ['get', 'ratio'], 0, 0.05, 1, 0.6],
        0,
      ],
    },
  });
  map.addLayer({
    id: COV_LINE,
    type: 'line',
    source: 'coverage',
    paint: {
      'line-color': ['case', ['>', ['get', 'surveyed'], 0], GREEN, GRID],
      'line-opacity': ['case', ['>', ['get', 'surveyed'], 0], 0.75, 0.5],
      'line-width': ['case', ['>', ['get', 'surveyed'], 0], 1.2, 0.5],
    },
  });

  map.addLayer({
    id: PLOT_FILL,
    type: 'fill',
    source: 'plots',
    paint: {
      'fill-color': GREEN,
      'fill-opacity': [
        '+',
        ['case', ['==', ['get', 'surveyed'], 1], 0.35, 0],
        ['*', 0.45, ['coalesce', ['feature-state', 'pulse'], 0]],
      ],
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
        ['boolean', ['feature-state', 'hover'], false], '#2b3540',
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
}

function setVisible(map: MapLibreMap, ids: string[], visible: boolean): void {
  for (const id of ids) map.setLayoutProperty(id, 'visibility', visible ? 'visible' : 'none');
}

export function showMode(map: MapLibreMap, mode: 'plots' | 'coverage' | 'none'): void {
  setVisible(map, [PLOT_FILL, PLOT_LINE], mode === 'plots');
  setVisible(map, [COV_FILL, COV_LINE], mode === 'coverage');
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
  (map.getSource('coverage') as GeoJSONSource).setData(fc);
}

export function setPlotState(map: MapLibreMap, cell: string, state: Record<string, unknown>): void {
  map.setFeatureState({ source: 'plots', id: cell }, state);
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
