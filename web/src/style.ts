// Base map style: web/style.json (owned by the map pipeline) if present, else a plain background.
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

export function baseStyle(origin = location.origin): StyleSpecification {
  const raw = found['../style.json'];
  if (!raw) {
    console.warn('web/style.json not found; using fallback background style');
    return FALLBACK;
  }
  const style = structuredClone(raw);
  for (const src of Object.values(style.sources ?? {})) {
    if ('url' in src && typeof src.url === 'string') src.url = absolutize(src.url, origin);
    if ('tiles' in src && Array.isArray(src.tiles)) src.tiles = src.tiles.map((t) => absolutize(t, origin));
  }
  if (typeof style.glyphs === 'string') style.glyphs = absolutize(style.glyphs, origin);
  if (typeof style.sprite === 'string') style.sprite = absolutize(style.sprite, origin);
  return style;
}
