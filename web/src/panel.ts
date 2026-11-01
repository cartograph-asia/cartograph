// Side panel (bottom sheet on narrow screens): plot details, photos, upload form, lightbox.
import { ApiError, type Api, type PlotDetail, type UploadResult } from './api';

const MAX_BYTES = 25 * 1024 * 1024;

function $<T extends HTMLElement>(id: string): T {
  const el = document.getElementById(id);
  if (!el) throw new Error(`#${id} missing`);
  return el as T;
}

function fmtDate(iso: string | null | undefined): string {
  if (!iso) return '';
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleDateString(undefined, { dateStyle: 'medium' });
}

export function isHeic(file: File): boolean {
  return /^image\/hei[cf]/i.test(file.type) || /\.(heic|heif)$/i.test(file.name);
}

export interface PanelEvents {
  onClose(): void;
  onUploaded(result: UploadResult): void;
}

export class Panel {
  private el = $<HTMLElement>('panel');
  private title = $<HTMLHeadingElement>('panel-title');
  private info = $<HTMLElement>('plot-info');
  private form = $<HTMLFormElement>('upload-form');
  private formTitle = $<HTMLHeadingElement>('upload-title');
  private file = $<HTMLInputElement>('photo-input');
  private surveyingAs = $<HTMLElement>('surveying-as');
  private signinPrompt = $<HTMLElement>('signin-prompt');
  private locBtn = $<HTMLButtonElement>('use-location');
  private locClear = $<HTMLButtonElement>('clear-location');
  private locText = $<HTMLElement>('location-text');
  private submit = $<HTMLButtonElement>('upload-submit');
  private status = $<HTMLElement>('form-status');
  private lightbox = $<HTMLDialogElement>('lightbox');
  private lightboxImg = $<HTMLImageElement>('lightbox-img');
  private lightboxCaption = $<HTMLElement>('lightbox-caption');

  private cell: string | null = null;
  private token = 0;
  private loc: { lat: number; lng: number } | null = null;
  private busy = false;

  constructor(private api: Api, private events: PanelEvents) {
    void this.loadMe();
    $<HTMLButtonElement>('panel-close').addEventListener('click', () => this.close());
    this.file.addEventListener('change', () => this.checkFile());
    this.locBtn.addEventListener('click', () => this.useLocation());
    this.locClear.addEventListener('click', () => this.setLocation(null));
    this.form.addEventListener('submit', (e) => {
      e.preventDefault();
      void this.upload();
    });
    $<HTMLButtonElement>('lightbox-close').addEventListener('click', () => this.lightbox.close());
    this.lightbox.addEventListener('click', (e) => {
      if (e.target === this.lightbox) this.lightbox.close();
    });
    this.lightbox.addEventListener('close', () => {
      this.lightboxImg.removeAttribute('src');
    });
    if (!('geolocation' in navigator)) this.locBtn.hidden = true;
  }

  /** Uploads need a signed-in Surveyor: show the form, or the sign-in link. */
  private async loadMe(): Promise<void> {
    let name: string | null = null;
    try {
      name = (await this.api.getMe())?.name ?? null;
    } catch {
      // treat as signed out; the upload would be refused anyway
    }
    this.setSurveyor(name);
  }

  private setSurveyor(name: string | null): void {
    this.form.hidden = name === null;
    this.signinPrompt.hidden = name !== null;
    this.surveyingAs.textContent = name ? `Surveying as ${name}` : '';
  }

  get isOpen(): boolean {
    return this.el.classList.contains('open');
  }

  get currentCell(): string | null {
    return this.cell;
  }

  /** Esc: lightbox first, then the panel. Returns true if something was closed. */
  escape(): boolean {
    if (this.lightbox.open) {
      this.lightbox.close();
      return true;
    }
    if (this.isOpen) {
      this.close();
      return true;
    }
    return false;
  }

  private show(): void {
    this.el.classList.add('open');
    this.el.inert = false;
    this.el.setAttribute('aria-hidden', 'false');
    document.body.classList.add('panel-open');
  }

  close(): void {
    if (!this.isOpen) return;
    this.el.classList.remove('open');
    this.el.inert = true;
    this.el.setAttribute('aria-hidden', 'true');
    document.body.classList.remove('panel-open');
    this.cell = null;
    this.token++;
    this.events.onClose();
  }

  /** Panel without a plot: upload only (the photo's location decides the plot). */
  openUpload(): void {
    this.cell = null;
    this.token++;
    this.title.textContent = 'Add a photo';
    this.info.replaceChildren();
    this.info.hidden = true;
    this.formTitle.textContent = 'Survey a plot';
    this.show();
    (this.form.hidden ? this.signinPrompt.querySelector('a') : this.file)?.focus();
  }

  async openPlot(cell: string, focus = true): Promise<void> {
    this.cell = cell;
    const token = ++this.token;
    this.title.textContent = 'Plot';
    this.info.hidden = false;
    this.info.replaceChildren(el('p', 'muted', 'Loading…'));
    this.formTitle.textContent = 'Add a photo';
    this.show();
    if (focus) $<HTMLButtonElement>('panel-close').focus();
    try {
      const detail = await this.api.getPlot(cell);
      if (token !== this.token) return;
      if (detail?.place) this.title.textContent = `Near ${detail.place}`;
      this.renderInfo(cell, detail);
    } catch (err) {
      if (token !== this.token) return;
      this.info.replaceChildren(el('p', 'error', `Could not load this plot: ${message(err)}`));
    }
  }

  private renderInfo(cell: string, d: PlotDetail | null): void {
    const nodes: Node[] = [];
    const meta = document.createElement('dl');
    meta.className = 'meta';
    const row = (k: string, v: string) => {
      if (!v) return;
      meta.append(el('dt', '', k), el('dd', '', v));
    };
    const pill = el('span', d ? 'pill surveyed' : 'pill blank', d ? 'Surveyed' : 'Blank');
    nodes.push(pill);
    if (d) {
      if (d.region && d.region !== d.place) row('Region', d.region);
      row('First surveyor', d.first_surveyor);
      row('Claimed', fmtDate(d.claimed_at));
      row('Photos', String(d.photo_count));
    }
    // The plot page, not the cell id: user-facing text never shows a bare cell.
    const link = el('a', '', d ? 'All photos and neighbours' : 'Plot page');
    link.setAttribute('href', `/plot/${cell}`);
    const pageDd = document.createElement('dd');
    pageDd.append(link);
    meta.append(el('dt', '', 'Page'), pageDd);
    nodes.push(meta);

    if (d && d.photos.length) {
      const grid = el('ul', 'thumbs');
      for (const p of d.photos) {
        const li = document.createElement('li');
        const btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'thumb';
        const label = `Photo by ${p.surveyor}${p.taken_at ? `, taken ${fmtDate(p.taken_at)}` : ''}`;
        btn.setAttribute('aria-label', `Open ${label}`);
        const img = document.createElement('img');
        img.src = p.urls.thumb;
        img.alt = label;
        img.loading = 'lazy';
        img.decoding = 'async';
        btn.append(img);
        btn.addEventListener('click', () => this.openLightbox(p.urls.large, label));
        li.append(btn);
        grid.append(li);
      }
      nodes.push(grid);
    } else {
      nodes.push(el('p', 'muted', 'Nobody has surveyed this plot yet. Be the first.'));
    }
    this.info.replaceChildren(...nodes);
  }

  private openLightbox(src: string, caption: string): void {
    this.lightboxImg.src = src;
    this.lightboxImg.alt = caption;
    this.lightboxCaption.textContent = caption;
    this.lightbox.showModal();
  }

  private setStatus(text: string, kind: 'info' | 'error' | 'ok' = 'info'): void {
    this.status.textContent = text;
    this.status.dataset.kind = kind;
  }

  /** Client-side checks before upload. Returns the file if acceptable. */
  private checkFile(): File | null {
    const f = this.file.files?.[0];
    if (!f) {
      this.setStatus('');
      return null;
    }
    if (isHeic(f)) {
      this.setStatus('HEIC photos are not supported yet. Please export the photo as JPEG and try again.', 'error');
      return null;
    }
    if (f.size > MAX_BYTES) {
      this.setStatus(`This file is ${(f.size / 1024 / 1024).toFixed(1)} MB; the limit is 25 MB.`, 'error');
      return null;
    }
    this.setStatus('');
    return f;
  }

  private setLocation(loc: { lat: number; lng: number } | null): void {
    this.loc = loc;
    this.locText.textContent = loc ? `Using ${loc.lat.toFixed(5)}, ${loc.lng.toFixed(5)}` : '';
    this.locClear.hidden = !loc;
  }

  private useLocation(): void {
    this.locBtn.disabled = true;
    this.locText.textContent = 'Finding your location…';
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        this.locBtn.disabled = false;
        this.setLocation({ lat: pos.coords.latitude, lng: pos.coords.longitude });
      },
      (err) => {
        this.locBtn.disabled = false;
        this.setLocation(null);
        this.locText.textContent = `Location unavailable: ${err.message}`;
      },
      { enableHighAccuracy: true, timeout: 15000, maximumAge: 60000 },
    );
  }

  private async upload(): Promise<void> {
    if (this.busy) return;
    const file = this.checkFile();
    if (!file) {
      if (!this.file.files?.length) this.setStatus('Choose a photo first.', 'error');
      this.file.focus();
      return;
    }
    this.busy = true;
    this.submit.disabled = true;
    this.submit.textContent = 'Uploading…';
    this.setStatus('Uploading photo…');
    try {
      const result = await this.api.uploadPhoto({
        photo: file,
        lat: this.loc?.lat,
        lng: this.loc?.lng,
      });
      this.form.reset();
      this.setLocation(null);
      this.setStatus(
        result.claimed ? 'Photo added. You are the first surveyor of this plot!' : 'Photo added to this plot.',
        'ok',
      );
      this.events.onUploaded(result);
    } catch (err) {
      this.setStatus(message(err), 'error');
      if (err instanceof ApiError && err.status === 401) this.setSurveyor(null); // session ended
    } finally {
      this.busy = false;
      this.submit.disabled = false;
      this.submit.textContent = 'Upload photo';
    }
  }
}

function message(err: unknown): string {
  if (err instanceof ApiError) return err.status >= 500 ? `Server error: ${err.message}` : err.message;
  if (err instanceof TypeError) return 'Network error. Check your connection and try again.';
  return err instanceof Error ? err.message : String(err);
}

function el<K extends keyof HTMLElementTagNameMap>(tag: K, cls: string, text?: string): HTMLElementTagNameMap[K] {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  if (text !== undefined) e.textContent = text;
  return e;
}
