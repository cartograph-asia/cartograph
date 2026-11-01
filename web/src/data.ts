// Session cache for plot summaries and coverage, with batching and in-flight de-duplication.
import { cellToParent } from 'h3-js';
import { chunk, COVERAGE_BATCH, PLOTS_BATCH } from './grid';
import type { Api, CoverageCell, PlotSummary } from './api';

export class DataStore {
  /** cell -> summary, or null for a known blank plot. */
  private plots = new Map<string, PlotSummary | null>();
  private plotsInflight = new Map<string, Promise<void>>();
  /** `${res}:${cell}` -> coverage, or null when the API returned nothing for it (nothing surveyed). */
  private coverage = new Map<string, CoverageCell | null>();
  private coverageInflight = new Map<string, Promise<void>>();

  constructor(private api: Api) {}

  plot(cell: string): PlotSummary | null | undefined {
    return this.plots.get(cell);
  }

  coverageOf(res: 5 | 6, cell: string): CoverageCell | null | undefined {
    return this.coverage.get(`${res}:${cell}`);
  }

  /** Fetch every cell not yet cached or in flight (batches of PLOTS_BATCH), then wait for all of them. */
  async ensurePlots(cells: string[]): Promise<{ requests: number; requested: number }> {
    const missing = cells.filter((c) => !this.plots.has(c) && !this.plotsInflight.has(c));
    const batches = chunk(missing, PLOTS_BATCH);
    for (const batch of batches) {
      const p = this.api.getPlots(batch).then(
        (found) => {
          for (const c of batch) this.plots.set(c, null);
          for (const s of found) this.plots.set(s.cell, s);
        },
      ).finally(() => {
        for (const c of batch) this.plotsInflight.delete(c);
      });
      for (const c of batch) this.plotsInflight.set(c, p);
    }
    const waits = new Set<Promise<void>>();
    for (const c of cells) {
      const p = this.plotsInflight.get(c);
      if (p) waits.add(p);
    }
    await Promise.all(waits);
    return { requests: batches.length, requested: missing.length };
  }

  async ensureCoverage(res: 5 | 6, cells: string[]): Promise<void> {
    const key = (c: string) => `${res}:${c}`;
    const missing = cells.filter((c) => !this.coverage.has(key(c)) && !this.coverageInflight.has(key(c)));
    for (const batch of chunk(missing, COVERAGE_BATCH)) {
      const p = this.api.getCoverage(res, batch).then(
        (found) => {
          for (const c of batch) this.coverage.set(key(c), null);
          for (const cc of found) this.coverage.set(key(cc.cell), cc);
        },
      ).finally(() => {
        for (const c of batch) this.coverageInflight.delete(key(c));
      });
      for (const c of batch) this.coverageInflight.set(key(c), p);
    }
    const waits = new Set<Promise<void>>();
    for (const c of cells) {
      const p = this.coverageInflight.get(key(c));
      if (p) waits.add(p);
    }
    await Promise.all(waits);
  }

  /** Forget a plot (after an upload) and the coverage of its parents. */
  invalidate(cell: string): void {
    this.plots.delete(cell);
    for (const res of [5, 6] as const) this.coverage.delete(`${res}:${cellToParent(cell, res)}`);
  }
}
