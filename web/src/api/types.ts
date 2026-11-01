// Types mirror the fixed API contract in docs/PROTOTYPE_BRIEF.md.

export interface PhotoUrls {
  large: string;
  thumb: string;
}

/** Item of GET /api/plots?cells= (only surveyed plots are returned). */
export interface PlotSummary {
  cell: string;
  photo_count: number;
  /** Surveyor name (the handle, for links). */
  first_surveyor: string;
  /** Display name, else the Surveyor name: the text to show. */
  first_surveyor_label: string;
  claimed_at: string;
  thumb: string;
}

export interface PhotoInfo {
  id: string;
  surveyor: string;
  surveyor_label: string;
  taken_at: string | null;
  uploaded_at: string;
  urls: PhotoUrls;
}

/** GET /api/plots/{cell} */
export interface PlotDetail {
  cell: string;
  photo_count: number;
  first_surveyor: string;
  first_surveyor_label: string;
  claimed_at: string;
  /** User-facing name, "{place} · {province} {number}". */
  name: string;
  /** Link to the plot page: /plot/{region-slug}/{number}, else /plot/{cell}. */
  path: string;
  /** Running number in the province; 0 if none yet. */
  number: number;
  /** Nearest place, else the region; empty when unknown. */
  place: string;
  region: string;
  photos: PhotoInfo[];
}

/** Item of GET /api/coverage */
export interface CoverageCell {
  cell: string;
  surveyed: number;
  total: number;
}

/** 201 of POST /api/photos */
export interface UploadResult {
  id: string;
  cell: string;
  claimed: boolean;
  urls: PhotoUrls;
}

/** 200 of GET /api/me: the signed-in Surveyor. */
export interface Me {
  /** Surveyor name (the handle, for links). */
  name: string;
  /** Display name, else the Surveyor name. */
  label: string;
}

/** The Surveyor comes from the session; there is no name field. */
export interface UploadInput {
  photo: File;
  lat?: number;
  lng?: number;
}

/** 200 of GET /api/config: what the map needs from the server's environment. */
export interface ClientConfig {
  /** The base map's PMTiles archive (TILES_URL): a path on this site or a full URL. */
  tiles_url: string;
}

export interface Api {
  getConfig(): Promise<ClientConfig>;
  /** At most PLOTS_BATCH cells per call; returns surveyed plots only. */
  getPlots(cells: string[]): Promise<PlotSummary[]>;
  /** null for a blank plot (404). */
  getPlot(cell: string): Promise<PlotDetail | null>;
  /** At most COVERAGE_BATCH cells per call. */
  getCoverage(res: 5 | 6, cells: string[]): Promise<CoverageCell[]>;
  uploadPhoto(input: UploadInput): Promise<UploadResult>;
  /** The signed-in Surveyor, or null when signed out or no name is chosen yet (401). */
  getMe(): Promise<Me | null>;
}

export class ApiError extends Error {
  readonly status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
    this.name = 'ApiError';
  }
}
