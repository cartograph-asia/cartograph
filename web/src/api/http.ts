// Real API: same-origin fetch to /api/...
import {
  ApiError,
  type Api,
  type CoverageCell,
  type PlotDetail,
  type PlotSummary,
  type UploadInput,
  type UploadResult,
} from './types';

async function errorFrom(res: Response): Promise<ApiError> {
  let message = `Request failed (${res.status})`;
  try {
    const body = (await res.json()) as { error?: unknown };
    if (typeof body.error === 'string' && body.error) message = body.error;
  } catch {
    // not JSON; keep the generic message
  }
  return new ApiError(res.status, message);
}

async function getJson<T>(url: string): Promise<T> {
  const res = await fetch(url, { headers: { Accept: 'application/json' } });
  if (!res.ok) throw await errorFrom(res);
  return (await res.json()) as T;
}

export const httpApi: Api = {
  async getPlots(cells: string[]): Promise<PlotSummary[]> {
    if (cells.length === 0) return [];
    const body = await getJson<{ plots: PlotSummary[] | null }>(
      `/api/plots?cells=${cells.join(',')}`,
    );
    return body.plots ?? [];
  },

  async getPlot(cell: string): Promise<PlotDetail | null> {
    const res = await fetch(`/api/plots/${encodeURIComponent(cell)}`, {
      headers: { Accept: 'application/json' },
    });
    if (res.status === 404) return null;
    if (!res.ok) throw await errorFrom(res);
    const body = (await res.json()) as PlotDetail;
    return { ...body, photos: body.photos ?? [] };
  },

  async getCoverage(res: 5 | 6, cells: string[]): Promise<CoverageCell[]> {
    if (cells.length === 0) return [];
    const body = await getJson<{ cells: CoverageCell[] | null }>(
      `/api/coverage?res=${res}&cells=${cells.join(',')}`,
    );
    return body.cells ?? [];
  },

  async uploadPhoto(input: UploadInput): Promise<UploadResult> {
    const form = new FormData();
    form.append('photo', input.photo);
    form.append('surveyor', input.surveyor);
    if (input.lat !== undefined && input.lng !== undefined) {
      form.append('lat', String(input.lat));
      form.append('lng', String(input.lng));
    }
    const res = await fetch('/api/photos', { method: 'POST', body: form });
    if (!res.ok) throw await errorFrom(res);
    return (await res.json()) as UploadResult;
  },
};
