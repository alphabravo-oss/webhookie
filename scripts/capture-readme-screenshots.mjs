// 1920x1080 viewport at deviceScaleFactor 2 → 3840x2160 PNGs for the README.
import { mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const outDir = join(root, 'docs/screenshots');
mkdirSync(outDir, { recursive: true });

const shots = [
  { url: 'http://localhost:5173/', file: 'inbox.png', wait: 'text=Inbox' },
  {
    url: 'http://localhost:5173/slack/ch-slack-alerts',
    file: 'slack-alerts.png',
    wait: 'text=#alerts',
  },
];

const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({
  viewport: { width: 1920, height: 1080 },
  deviceScaleFactor: 2,
  colorScheme: 'dark',
});
const page = await context.newPage();

for (const shot of shots) {
  await page.goto(shot.url, { waitUntil: 'load' });
  await page.locator(shot.wait).first().waitFor({ state: 'visible', timeout: 15000 });
  await page.waitForTimeout(600);
  const dest = join(outDir, shot.file);
  await page.screenshot({ path: dest, type: 'png', animations: 'disabled' });
  console.log('wrote', dest);
}

await browser.close();
