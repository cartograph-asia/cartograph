// Base map style: web/style.json (owned by the map pipeline) if present, else a plain background.
// The archive's address comes from the server (TILES_URL via GET /api/config);
// the URL in style.json is the default for when it cannot be asked.
import type { ExpressionSpecification, StyleSpecification } from 'maplibre-gl';
import { PMTiles, type Protocol } from 'pmtiles';

// Glob import so the build still works while web/style.json does not exist yet.
const found = import.meta.glob<StyleSpecification>('../style.json', {
  eager: true,
  import: 'default',
});

const FALLBACK: StyleSpecification = {
  version: 8,
  sources: {},
  layers: [{ id: 'background', type: 'background', paint: { 'background-color': '#ecebe7' } }],
};

/** west, south, east, north in degrees. */
export type Bounds = [number, number, number, number];

/**
 * What the tile archive covers when its header cannot be read: map/build.sh
 * BOUNDS. Normally the bounds come from the archive itself, so a rebuild
 * with other BOUNDS needs no change here.
 */
export const ARCHIVE_BOUNDS: Bounds = [88, -12, 150, 55];

/** The page's paper (styles.css --paper): the margin beyond the archive. */
const PAPER = '#f6f2ea';

/** How long the map waits for the archive's header before it uses ARCHIVE_BOUNDS. */
const HEADER_TIMEOUT_MS = 4000;

/** Make root-relative URLs absolute; MapLibre workers cannot resolve them otherwise. */
function absolutize(url: string, origin: string): string {
  if (url.startsWith('pmtiles:///')) return `pmtiles://${origin}/${url.slice('pmtiles:///'.length)}`;
  if (url.startsWith('/') && !url.startsWith('//')) return origin + url;
  return url;
}

/** Make a tiles URL from GET /api/config ("/tiles/asia.pmtiles" or a full URL) a pmtiles:// source URL. */
function pmtilesSource(tilesUrl: string, origin: string): string {
  return 'pmtiles://' + (tilesUrl.startsWith('/') ? origin + tilesUrl : tilesUrl);
}

export function baseStyle(tilesUrl?: string | null, origin = location.origin): StyleSpecification {
  const raw = found['../style.json'];
  if (!raw) {
    console.warn('web/style.json not found; using fallback background style');
    return FALLBACK;
  }
  const style = structuredClone(raw);
  for (const src of Object.values(style.sources ?? {})) {
    if ('url' in src && typeof src.url === 'string') {
      src.url = tilesUrl && src.url.startsWith('pmtiles:') ? pmtilesSource(tilesUrl, origin) : absolutize(src.url, origin);
    }
    if ('tiles' in src && Array.isArray(src.tiles)) src.tiles = src.tiles.map((t) => absolutize(t, origin));
  }
  if (typeof style.glyphs === 'string') style.glyphs = absolutize(style.glyphs, origin);
  if (typeof style.sprite === 'string') style.sprite = absolutize(style.sprite, origin);
  return style;
}

/**
 * The area the style's tile archive covers, from the archive's header. The
 * archive is registered with the protocol, so the map reuses the header
 * instead of fetching it again.
 */
async function archiveBounds(style: StyleSpecification, protocol: Protocol): Promise<Bounds> {
  const src = Object.values(style.sources).find(
    (s) => 'url' in s && typeof s.url === 'string' && s.url.startsWith('pmtiles://'),
  );
  if (!src || !('url' in src) || typeof src.url !== 'string') return ARCHIVE_BOUNDS;
  const archive = new PMTiles(src.url.slice('pmtiles://'.length));
  protocol.add(archive);
  let timer: number | undefined;
  const timeout = new Promise<null>((resolve) => (timer = window.setTimeout(() => resolve(null), HEADER_TIMEOUT_MS)));
  try {
    const h = await Promise.race([archive.getHeader(), timeout]);
    if (!h || !(h.minLon < h.maxLon && h.minLat < h.maxLat)) return ARCHIVE_BOUNDS;
    return [h.minLon, h.minLat, h.maxLon, h.maxLat];
  } catch (err) {
    console.warn('tile archive header:', err);
    return ARCHIVE_BOUNDS;
  } finally {
    window.clearTimeout(timer);
  }
}

/**
 * Cover everything beyond the archive with the page's paper. Past its
 * bounds the archive ends at tile edges that differ by zoom: the sea stops
 * in a straight line and the base map's background (the land colour, since
 * land is implicit in OpenMapTiles) continues as if it were land, and the
 * land at the western and northern edges (India, Tibet, Siberia) stops just
 * as abruptly. The paper hides all of that at every zoom. Towards the edge
 * the map fades into it over a band of a few dozen pixels, so that neither
 * land nor sea ends in a line; the land colour is close to the paper, so
 * mostly the sea fades, like an atlas whose sea tint thins out at the margin.
 * The layers go on top of the base map and below what the app adds.
 */
function addMargin(style: StyleSpecification, [w, s, e, n]: Bounds): void {
  const hole = [[w, s], [w, n], [e, n], [e, s], [w, s]]; // clockwise: a hole in the world
  // The fade is drawn as four separate edges, each running on into the
  // paper past the corners, so the corners are where two fades overlap; one
  // joined ring left a lighter wedge in the corners' joins.
  const x = 5;
  const sx = Math.max(-85, s - x);
  const nx = Math.min(85, n + x);
  const edges = [
    [[w, sx], [w, nx]],
    [[e, sx], [e, nx]],
    [[w - x, s], [e + x, s]],
    [[w - x, n], [e + x, n]],
  ];
  const world = [[-180, -85.06], [180, -85.06], [180, 85.06], [-180, 85.06], [-180, -85.06]];
  style.sources.margin = {
    type: 'geojson',
    data: {
      type: 'FeatureCollection',
      features: [
        { type: 'Feature', properties: {}, geometry: { type: 'Polygon', coordinates: [world, hole] } },
        ...edges.map((coordinates) => ({ type: 'Feature' as const, properties: {}, geometry: { type: 'LineString' as const, coordinates } })),
      ],
    },
  };
  // The fade is a blurred line along the edge: opaque on it, clear at half
  // its width inside. Narrow when zoomed out, where a degree is a few pixels
  // and Indonesia's southern islands lie one degree inside the edge.
  const fade = (k: number): ExpressionSpecification => ['interpolate', ['linear'], ['zoom'], 2, 48 * k, 4, 96 * k, 7, 160 * k];
  style.layers.push(
    {
      id: 'margin',
      type: 'fill',
      source: 'margin',
      filter: ['==', ['geometry-type'], 'Polygon'],
      paint: { 'fill-color': PAPER, 'fill-antialias': false },
    },
    {
      id: 'margin-fade',
      type: 'line',
      source: 'margin',
      filter: ['==', ['geometry-type'], 'LineString'],
      layout: { 'line-cap': 'butt' },
      paint: { 'line-color': PAPER, 'line-width': fade(1), 'line-blur': fade(0.5) },
    },
  );
}

/**
 * The base style with the paper margin beyond the archive, and the area the
 * archive covers.
 */
export async function loadBaseStyle(
  tilesUrl: string | null | undefined,
  protocol: Protocol,
): Promise<{ style: StyleSpecification; bounds: Bounds }> {
  const style = baseStyle(tilesUrl);
  if (style === FALLBACK) return { style, bounds: ARCHIVE_BOUNDS };
  const bounds = await archiveBounds(style, protocol);
  // A world archive (BOUNDS=world) has no edge to hide.
  const world = bounds[0] <= -179.9 && bounds[2] >= 179.9;
  if (!world) addMargin(style, bounds);
  return { style, bounds };
}
