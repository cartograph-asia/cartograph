// Screenshots of the server-rendered pages for design review.
//
//   cd web && npm run screenshots -- round-2
//   cd web && npm run screenshots -- home,photo --out scratch
//
// Takes every page of a set in tools/screenshots/pages.json (or a comma list
// of page names) at 400 px and 1400 px wide, as full-page JPEGs of at most
// 300 KB (tall pages at a reduced device scale) into docs/design/<out>/ with
// the names <page>-<width>.jpg. It runs against cmd/webpreview on
// 127.0.0.1:8093 and starts it (go run ./cmd/webpreview) when nothing
// answers there. Pages given as full URLs (the /map page and the small
// maps, which need tiles and the API) are taken from there instead, as a
// rule the compose stack on 127.0.0.1:8080, which must be running. It drives the installed Google Chrome through
// playwright-core (a dev dependency in web/), so no browser is downloaded;
// set CHROME_PATH to use another Chromium.
import { spawn } from 'node:child_process';
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const root = resolve(here, '../..');
const { chromium } = createRequire(join(root, 'web/package.json'))('playwright-core');

const BASE = 'http://127.0.0.1:8093';
const WIDTHS = [400, 1400];
const MAX_BYTES = 300 * 1024;

function args() {
  const a = process.argv.slice(2);
  const opt = { set: 'round-1', out: null };
  for (let i = 0; i < a.length; i++) {
    if (a[i] === '--out') opt.out = a[++i];
    else opt.set = a[i];
  }
  opt.out ??= opt.set;
  return opt;
}

async function up() {
  try {
    return (await fetch(BASE + '/')).ok;
  } catch {
    return false;
  }
}

// startPreview runs cmd/webpreview unless one is already answering, and
// returns a function that stops what it started.
async function startPreview() {
  if (await up()) return () => {};
  // In its own process group, so that stopping it also stops the
  // webpreview binary that go run starts.
  const proc = spawn('go', ['run', './cmd/webpreview'], { cwd: root, stdio: 'ignore', detached: true });
  const kill = () => {
    try {
      process.kill(-proc.pid, 'SIGTERM');
    } catch {}
  };
  for (let i = 0; i < 120; i++) {
    if (await up()) return kill;
    await new Promise((r) => setTimeout(r, 500));
  }
  kill();
  throw new Error('cmd/webpreview did not start on ' + BASE);
}

// loadEverything scrolls the page so that lazy images load, then waits for
// images and fonts.
async function loadEverything(page) {
  await page.evaluate(async () => {
    for (let y = 0; y < document.body.scrollHeight; y += innerHeight / 2) {
      scrollTo(0, y);
      await new Promise((r) => setTimeout(r, 60));
    }
    scrollTo(0, 0);
    await Promise.all([...document.images].map((img) => (img.complete ? null : img.decode().catch(() => null))));
    await document.fonts.ready;
  });
  // A map keeps fetching tile ranges for a while and may never let the
  // network fall quiet; past a limit, give it a moment to draw instead.
  const map = await page.$('.maplibregl-canvas, [data-minimap]');
  await page.waitForLoadState('networkidle', { timeout: map ? 10000 : 30000 }).catch((err) => {
    if (!map) throw err;
  });
  if (map) await page.waitForTimeout(2500);
}

// Each shot is tried at full scale first; a tall page that does not fit
// MAX_BYTES at a decent quality is rendered at a smaller device scale (the
// layout stays at the viewport's CSS width, the image gets fewer pixels).
const SCALES = [1, 0.8, 0.65, 0.5];
const QUALITIES = [82, 74, 66, 58, 50];

// shoot loads url at width and saves a full-page JPEG of at most MAX_BYTES.
async function shoot(browser, url, width, file) {
  for (const scale of SCALES) {
    const context = await browser.newContext({ viewport: { width, height: 900 }, deviceScaleFactor: scale });
    const page = await context.newPage();
    try {
      await page.goto(url, { waitUntil: 'load' });
      await loadEverything(page);
      // A full-page shot stretches the viewport for the capture, and a map
      // caught in that resize is drawn at MapLibre's 400 × 300 fallback size.
      // So a page with a small map gets its full height first and a moment
      // to redraw.
      if (await page.$('[data-minimap]')) {
        const height = await page.evaluate(() => document.documentElement.scrollHeight);
        await page.setViewportSize({ width, height });
        await page.waitForTimeout(2000);
      }
      for (const q of QUALITIES) {
        const buf = await page.screenshot({ fullPage: true, type: 'jpeg', quality: q });
        if (buf.length <= MAX_BYTES) {
          writeFileSync(file, buf);
          return { scale, q, kb: Math.round(buf.length / 1024) };
        }
      }
    } finally {
      await context.close();
    }
  }
  throw new Error(`${url} at ${width}px does not fit ${MAX_BYTES} bytes`);
}

const opt = args();
const { pages, sets } = JSON.parse(readFileSync(join(here, 'pages.json'), 'utf8'));
const names = sets[opt.set] ?? opt.set.split(',');
for (const n of names) if (!pages[n]) throw new Error(`unknown page "${n}" (see pages.json)`);
const outDir = join(root, 'docs/design', opt.out);
mkdirSync(outDir, { recursive: true });

const stop = await startPreview();
const browser = await chromium.launch(
  process.env.CHROME_PATH ? { executablePath: process.env.CHROME_PATH } : { channel: 'chrome' },
);
try {
  for (const width of WIDTHS) {
    for (const n of names) {
      const file = join(outDir, `${n}-${width}.jpg`);
      const url = /^https?:\/\//.test(pages[n]) ? pages[n] : BASE + pages[n];
      const { scale, q, kb } = await shoot(browser, url, width, file);
      console.log(`${file.slice(root.length + 1)}  scale ${scale}  q${q}  ${kb} KB`);
    }
  }
} finally {
  await browser.close();
  stop();
}
