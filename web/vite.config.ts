import { defineConfig } from 'vite';

// In dev, everything except the app itself goes to the compose stack's Caddy entry point.
const target = 'http://localhost:8080';
const proxy = Object.fromEntries(
  ['/api', '/healthz', '/tiles', '/img'].map((p) => [p, { target, changeOrigin: true }]),
);

export default defineConfig({
  // The map lives at /map/; every other path is a server-rendered page from the api.
  base: '/map/',
  server: { port: 5173, proxy },
  preview: { proxy },
  // maplibre-gl alone is ~1 MB minified; one chunk is fine for the prototype.
  build: { outDir: 'dist', target: 'es2022', chunkSizeWarningLimit: 1500 },
});
