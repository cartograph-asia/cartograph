// Base map style: web/style.json (owned by the map pipeline) if present, else a plain background.
// The archive's address comes from the server (TILES_URL via GET /api/config);
// the URL in style.json is the default for when it cannot be asked.
import type { StyleSpecification } from 'maplibre-gl';

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
