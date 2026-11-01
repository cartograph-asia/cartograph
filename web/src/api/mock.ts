// In-memory implementation of the four endpoints, used when VITE_MOCK=1.
// Behaves like the contract: only surveyed plots are returned, blank plot -> null (404),
// uploads need EXIF GPS or lat/lng, > 25 MB and non-JPEG/PNG are rejected with 400.
import { latLngToCell, cellToParent, cellToChildrenSize } from 'h3-js';
import { PLOT_RES } from '../grid';
import {
  ApiError,
  type Api,
  type CoverageCell,
  type PhotoInfo,
  type PlotDetail,
  type PlotSummary,
  type UploadInput,
  type UploadResult,
} from './types';

const MAX_BYTES = 25 * 1024 * 1024;
const LATENCY_MS = 150;
/** The mock is always signed in as this Surveyor. */
const MOCK_SURVEYOR = 'mock-surveyor';

interface MockPlot {
  cell: string;
  first_surveyor: string;
  claimed_at: string;
  photos: PhotoInfo[];
}

const plots = new Map<string, MockPlot>();

function delay<T>(value: T): Promise<T> {
  return new Promise((resolve) => setTimeout(() => resolve(value), LATENCY_MS));
}

function placeholder(label: string, hue: number, w: number, h: number): string {
  const svg =
    `<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}" viewBox="0 0 ${w} ${h}">` +
    `<defs><linearGradient id="g" x1="0" y1="0" x2="0" y2="1">` +
    `<stop offset="0" stop-color="hsl(${hue},45%,78%)"/><stop offset="1" stop-color="hsl(${hue + 30},35%,52%)"/>` +
    `</linearGradient></defs><rect width="100%" height="100%" fill="url(#g)"/>` +
    `<text x="50%" y="50%" font-family="system-ui,sans-serif" font-size="${Math.round(w / 14)}" ` +
    `fill="rgba(0,0,0,.55)" text-anchor="middle" dominant-baseline="middle">${label}</text></svg>`;
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`;
}

function addPhoto(
  lat: number,
  lng: number,
  surveyor: string,
  urls: { large: string; thumb: string },
  when: Date,
): { cell: string; claimed: boolean; id: string } {
  const cell = latLngToCell(lat, lng, PLOT_RES);
  const id = crypto.randomUUID();
  const photo: PhotoInfo = {
    id,
    surveyor,
    surveyor_label: surveyor,
    taken_at: when.toISOString(),
    uploaded_at: when.toISOString(),
    urls,
  };
  let plot = plots.get(cell);
  const claimed = !plot;
  if (!plot) {
    plot = { cell, first_surveyor: surveyor, claimed_at: when.toISOString(), photos: [] };
    plots.set(cell, plot);
  }
  plot.photos.push(photo);
  return { cell, claimed, id };
}

// Fixtures around Bangkok and Pattaya.
const fixtures: [number, number, string, string][] = [
  [13.7500, 100.4913, 'Somchai', 'Grand Palace'],
  [13.7465, 100.4927, 'Mali', 'Wat Pho'],
  [13.7307, 100.5418, 'Somchai', 'Lumphini Park'],
  [13.7310, 100.5422, 'Anan', 'Lumphini Park'],
  [13.7999, 100.5500, 'Mali', 'Chatuchak'],
  [13.7437, 100.4889, 'Anan', 'Wat Arun'],
  [12.9276, 100.8770, 'Pim', 'Pattaya Beach'],
  [12.9300, 100.8900, 'Pim', 'Pattaya'],
  [12.8870, 100.8730, 'Lek', 'Jomtien'],
  [12.9500, 100.8890, 'Lek', 'Naklua'],
];
fixtures.forEach(([lat, lng, who, label], i) => {
  const hue = (i * 47) % 360;
  addPhoto(
    lat,
    lng,
    who,
    { large: placeholder(label, hue, 1600, 1200), thumb: placeholder(label, hue, 480, 360) },
    new Date(Date.UTC(2026, 9, 1 + i, 4, 30)),
  );
});

function summary(p: MockPlot): PlotSummary {
  return {
    cell: p.cell,
    photo_count: p.photos.length,
    first_surveyor: p.first_surveyor,
    first_surveyor_label: p.first_surveyor,
    claimed_at: p.claimed_at,
    thumb: p.photos[0].urls.thumb,
  };
}

// --- minimal EXIF GPS reader (JPEG APP1 / TIFF), enough for the mock -------------

function readExifGps(buf: ArrayBuffer): { lat: number; lng: number } | null {
  const v = new DataView(buf);
  if (v.byteLength < 4 || v.getUint16(0) !== 0xffd8) return null;
  let off = 2;
  while (off + 4 <= v.byteLength) {
    const marker = v.getUint16(off);
    if ((marker & 0xff00) !== 0xff00) return null;
    const len = v.getUint16(off + 2);
    if (marker === 0xffe1 && v.getUint32(off + 4) === 0x45786966 /* "Exif" */) {
      return parseTiffGps(v, off + 10);
    }
    if (marker === 0xffda) return null; // start of scan, no EXIF before it
    off += 2 + len;
  }
  return null;
}

function parseTiffGps(v: DataView, tiff: number): { lat: number; lng: number } | null {
  const le = v.getUint16(tiff) === 0x4949;
  const u16 = (o: number) => v.getUint16(tiff + o, le);
  const u32 = (o: number) => v.getUint32(tiff + o, le);
  const findTag = (ifd: number, tag: number): number | null => {
    const n = u16(ifd);
    for (let i = 0; i < n; i++) {
      const e = ifd + 2 + i * 12;
      if (u16(e) === tag) return e;
    }
    return null;
  };
  const gpsPtr = findTag(u32(4), 0x8825);
  if (gpsPtr === null) return null;
  const gps = u32(gpsPtr + 8);
  const ref = (tag: number) => {
    const e = findTag(gps, tag);
    return e === null ? '' : String.fromCharCode(v.getUint8(tiff + e + 8));
  };
  const dms = (tag: number): number | null => {
    const e = findTag(gps, tag);
    if (e === null) return null;
    const at = u32(e + 8);
    const r = (k: number) => u32(at + k * 8) / (u32(at + k * 8 + 4) || 1);
    return r(0) + r(1) / 60 + r(2) / 3600;
  };
  const lat = dms(2);
  const lng = dms(4);
  if (lat === null || lng === null) return null;
  return { lat: ref(1) === 'S' ? -lat : lat, lng: ref(3) === 'W' ? -lng : lng };
}

// --- endpoints ---------------------------------------------------------------------

export const mockApi: Api = {
  async getConfig() {
    return { tiles_url: '/tiles/asia.pmtiles', surveyable: [97.3, 5.6, 105.6, 20.5] as [number, number, number, number] };
  },
  getPlots(cells: string[]): Promise<PlotSummary[]> {
    if (cells.length > 2000) return Promise.reject(new ApiError(400, 'too many cells (max 2000)'));
    const out: PlotSummary[] = [];
    for (const c of cells) {
      const p = plots.get(c);
      if (p) out.push(summary(p));
    }
    return delay(out);
  },

  getPlot(cell: string): Promise<PlotDetail | null> {
    const p = plots.get(cell);
    if (!p) return delay(null);
    return delay({
      cell: p.cell,
      photo_count: p.photos.length,
      first_surveyor: p.first_surveyor,
      first_surveyor_label: p.first_surveyor,
      claimed_at: p.claimed_at,
      name: 'Pattaya · Chon Buri 17',
      path: '/plot/chon-buri/17',
      number: 17,
      place: 'Pattaya',
      region: 'Chon Buri',
      photos: [...p.photos].reverse(),
    });
  },

  getCoverage(res: 5 | 6, cells: string[]): Promise<CoverageCell[]> {
    if (cells.length > 500) return Promise.reject(new ApiError(400, 'too many cells (max 500)'));
    const wanted = new Set(cells);
    const counts = new Map<string, number>();
    for (const cell of plots.keys()) {
      const parent = cellToParent(cell, res);
      if (wanted.has(parent)) counts.set(parent, (counts.get(parent) ?? 0) + 1);
    }
    const total = cellToChildrenSize(cells[0] ?? '85283473fffffff', PLOT_RES);
    return delay(
      [...counts].map(([cell, surveyed]) => ({ cell, surveyed, total: Number(total) })),
    );
  },

  async uploadPhoto(input: UploadInput): Promise<UploadResult> {
    await delay(null);
    const { photo } = input;
    if (photo.size > MAX_BYTES) throw new ApiError(400, 'file is larger than 25 MB');
    if (!['image/jpeg', 'image/png'].includes(photo.type)) {
      throw new ApiError(400, 'unsupported file type; use JPEG or PNG');
    }
    const gps = photo.type === 'image/jpeg' ? readExifGps(await photo.arrayBuffer()) : null;
    const loc = gps ?? (input.lat !== undefined && input.lng !== undefined
      ? { lat: input.lat, lng: input.lng }
      : null);
    if (!loc) throw new ApiError(400, 'no GPS location in photo and no lat/lng given');
    const url = URL.createObjectURL(photo);
    const urls = { large: url, thumb: url };
    const r = addPhoto(loc.lat, loc.lng, MOCK_SURVEYOR, urls, new Date());
    return { id: r.id, cell: r.cell, claimed: r.claimed, urls };
  },

  async getMe() {
    return delay({ name: MOCK_SURVEYOR, label: MOCK_SURVEYOR });
  },
};
