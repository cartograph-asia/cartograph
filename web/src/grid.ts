// Pure grid helpers: viewport -> H3 cells, batching, geometry.
// No MapLibre or DOM imports, so this module can be exercised from Node.
import {
  polygonToCells,
  cellToBoundary,
  getHexagonAreaAvg,
  UNITS,
} from 'h3-js';

export const PLOT_RES = 8;

/** Plot layer is shown from this zoom on (if the viewport is small enough). */
export const PLOT_MIN_ZOOM = 10;
/** Above this many cells in view we stay in coverage mode (raises the effective threshold zoom). */
export const MAX_VIEW_CELLS = 8000;
/** Contract limit for GET /api/plots?cells= */
export const PLOTS_BATCH = 2000;
/** Contract limit for GET /api/coverage?cells= */
export const COVERAGE_BATCH = 500;

export interface BBox {
  west: number;
  south: number;
  east: number;
  north: number;
}

/** Prototype region. Viewports are clipped to this before computing cells. */
export const THAILAND_BBOX: BBox = { west: 97.0, south: 5.4, east: 106.0, north: 20.8 };

export function padBBox(b: BBox, fraction: number): BBox {
  const dx = (b.east - b.west) * fraction;
  const dy = (b.north - b.south) * fraction;
  return { west: b.west - dx, south: b.south - dy, east: b.east + dx, north: b.north + dy };
}

export function clipBBox(b: BBox, to: BBox): BBox | null {
  const c = {
    west: Math.max(b.west, to.west),
    south: Math.max(b.south, to.south),
    east: Math.min(b.east, to.east),
    north: Math.min(b.north, to.north),
  };
  return c.west < c.east && c.south < c.north ? c : null;
}

/** Approximate area in km² (good enough for a count estimate at Thai latitudes). */
export function bboxAreaKm2(b: BBox): number {
  const kmPerDeg = 111.32;
  const midLat = ((b.north + b.south) / 2) * (Math.PI / 180);
  return (b.east - b.west) * kmPerDeg * Math.cos(midLat) * (b.north - b.south) * kmPerDeg;
}

/** Cheap pre-check so we never run polygonToCells for a huge viewport. */
export function estimateCellCount(b: BBox, res: number): number {
  return Math.ceil(bboxAreaKm2(b) / getHexagonAreaAvg(res, UNITS.km2));
}

export function cellsForBBox(b: BBox, res: number): string[] {
  const ring = [
    [b.north, b.west],
    [b.north, b.east],
    [b.south, b.east],
    [b.south, b.west],
  ];
  return polygonToCells(ring, res);
}

export function chunk<T>(items: readonly T[], size: number): T[][] {
  const out: T[][] = [];
  for (let i = 0; i < items.length; i += size) out.push(items.slice(i, i + size));
  return out;
}

/** GeoJSON ring ([lng, lat], closed) for a cell. */
export function cellRing(cell: string): number[][] {
  const ring = cellToBoundary(cell, true);
  ring.push(ring[0]);
  return ring;
}

export type ViewPlan =
  | { mode: 'plots'; res: number; cells: string[] }
  | { mode: 'coverage'; res: 5 | 6; cells: string[] }
  | { mode: 'none' };

/**
 * Decide what to draw for a viewport.
 * Plots: zoom >= PLOT_MIN_ZOOM and at most MAX_VIEW_CELLS res-8 cells in view.
 * Coverage: res 5 below zoom 7, res 6 otherwise; falls back to res 5 if res 6 would be too many.
 */
export function planView(viewport: BBox, zoom: number, padding = 0.1): ViewPlan {
  const area = clipBBox(padBBox(viewport, padding), THAILAND_BBOX);
  if (!area) return { mode: 'none' };

  if (zoom >= PLOT_MIN_ZOOM && estimateCellCount(area, PLOT_RES) <= MAX_VIEW_CELLS) {
    const cells = cellsForBBox(area, PLOT_RES);
    if (cells.length <= MAX_VIEW_CELLS) return { mode: 'plots', res: PLOT_RES, cells };
  }

  let res: 5 | 6 = zoom < 7 ? 5 : 6;
  if (res === 6 && estimateCellCount(area, 6) > MAX_VIEW_CELLS) res = 5;
  return { mode: 'coverage', res, cells: cellsForBBox(area, res) };
}
