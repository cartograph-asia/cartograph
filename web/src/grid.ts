// Pure grid helpers: viewport -> H3 cells, batching, geometry.
// No MapLibre or DOM imports, so this module can be exercised from Node.
import {
  polygonToCells,
  cellToBoundary,
  cellToLatLng,
  getHexagonAreaAvg,
  UNITS,
} from 'h3-js';

export const PLOT_RES = 8;

/** Plot layer is shown from this zoom on (if the viewport is small enough). */
export const PLOT_MIN_ZOOM = 10;
/** Above this many cells in view we stay in coverage mode (raises the effective threshold zoom). */
export const MAX_VIEW_CELLS = 8000;
/**
 * Cells per GET /api/plots?cells= request. The API takes up to 2000, but at
 * 16 characters a cell that is a 32 KB URL, and Caddy refuses request lines
 * past about 20 KB with a 431 (so do Vite's dev proxy at 16 KB): those plots
 * then never got their colour. 800 cells keep the URL near 13 KB.
 */
export const PLOTS_BATCH = 800;
/** Contract limit for GET /api/coverage?cells= */
export const COVERAGE_BATCH = 500;

export interface BBox {
  west: number;
  south: number;
  east: number;
  north: number;
}

/**
 * Where plots can be surveyed when the server does not say (GET /api/config
 * "surveyable"): Thailand, the first imported country.
 */
export const THAILAND_BBOX: BBox = { west: 97.0, south: 5.4, east: 106.0, north: 20.8 };

/**
 * Blank groups are drawn only from this zoom on (layers.ts fades them in
 * from here); below it, groups are drawn only where plots can be surveyed,
 * since a blank group is invisible there anyway and the whole of Asia would
 * be tens of thousands of cells.
 */
export const BLANK_GROUP_MIN_ZOOM = 6.5;

/** Whether the centre of cell lies in b. */
export function cellIn(cell: string, b: BBox): boolean {
  const [lat, lng] = cellToLatLng(cell);
  return lat >= b.south && lat <= b.north && lng >= b.west && lng <= b.east;
}

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

/**
 * cells are drawn; dataCells, the part of them inside the surveyable area,
 * are the ones to ask the API about. dataCells is empty where no plot can be
 * surveyed yet.
 */
export type ViewPlan =
  | { mode: 'plots'; res: number; cells: string[]; dataCells: string[] }
  | { mode: 'coverage'; res: 5 | 6; cells: string[]; dataCells: string[] }
  | { mode: 'none' };

/**
 * Decide what to draw for a viewport. The grid is drawn everywhere;
 * surveyable is where plots can be claimed (the imported provinces).
 * Plots: zoom >= PLOT_MIN_ZOOM and at most MAX_VIEW_CELLS res-8 cells in view.
 * Coverage: res 5 below zoom 7, res 6 otherwise; falls back to res 5 if res 6
 * would be too many. Blank groups outside the surveyable area only from
 * BLANK_GROUP_MIN_ZOOM and when they fit in MAX_VIEW_CELLS.
 */
export function planView(viewport: BBox, zoom: number, surveyable: BBox = THAILAND_BBOX, padding = 0.1): ViewPlan {
  const area = padBBox(viewport, padding);
  const data = clipBBox(area, surveyable);
  // A group near the border can hold surveyable plots while its centre lies
  // just outside: ask about everything within half a degree.
  const ask = { west: surveyable.west - 0.5, south: surveyable.south - 0.5, east: surveyable.east + 0.5, north: surveyable.north + 0.5 };
  const inside = (cells: string[]) => (data ? cells.filter((c) => cellIn(c, ask)) : []);

  if (zoom >= PLOT_MIN_ZOOM && estimateCellCount(area, PLOT_RES) <= MAX_VIEW_CELLS) {
    const cells = cellsForBBox(area, PLOT_RES);
    if (cells.length <= MAX_VIEW_CELLS) return { mode: 'plots', res: PLOT_RES, cells, dataCells: inside(cells) };
  }

  let res: 5 | 6 = zoom < 7 ? 5 : 6;
  const everywhere = zoom >= BLANK_GROUP_MIN_ZOOM;
  const drawn = everywhere ? area : data;
  if (!drawn) return { mode: 'none' };
  if (res === 6 && estimateCellCount(drawn, 6) > MAX_VIEW_CELLS) res = 5;
  if (everywhere && estimateCellCount(area, res) > MAX_VIEW_CELLS) {
    // Too far out to draw blank groups everywhere: only where there is data.
    if (!data) return { mode: 'none' };
    const cells = cellsForBBox(data, res);
    return { mode: 'coverage', res, cells, dataCells: inside(cells) };
  }
  const cells = cellsForBBox(drawn, res);
  return { mode: 'coverage', res, cells, dataCells: inside(cells) };
}
