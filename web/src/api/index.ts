import { httpApi } from './http';
import type { Api } from './types';

export const MOCK = import.meta.env.VITE_MOCK === '1';

/** Resolves to the in-memory mock when VITE_MOCK=1; the mock is not bundled otherwise. */
export async function loadApi(): Promise<Api> {
  if (MOCK) {
    const { mockApi } = await import('./mock');
    return mockApi;
  }
  return httpApi;
}

export * from './types';
