/**
 * Visual QA script: opens the home page in system Chrome, scrolls through
 * it slowly so every IntersectionObserver-driven animation fires, then
 * captures section screenshots and a final full-page shot.
 *
 * Usage: bun run scripts/qa-screenshots.ts [url]
 */
import { chromium } from "playwright-core";

const url = process.argv[2] ?? "http://localhost:3000";
const outDir = "/tmp/qa";

const browser = await chromium.launch({
  channel: "chrome",
  headless: true,
  args: ["--no-sandbox", "--disable-gpu"],
});
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });

await page.goto(url, { waitUntil: "networkidle" });

// Scroll slowly to the bottom to trigger every useInView / IntersectionObserver.
await page.evaluate(async () => {
  const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
  const step = 400;
  for (let y = 0; y <= document.body.scrollHeight; y += step) {
    window.scrollTo(0, y);
    await sleep(60);
  }
  window.scrollTo(0, 0);
  await sleep(400);
});

// Let animations settle.
await page.waitForTimeout(1200);

const sections = [
  "seccion-hero",
  "seccion-oferta",
  "seccion-bodas",
  "seccion-quinceaneras",
  "seccion-bautizos",
  "seccion-corporativos",
  "seccion-graduaciones",
  "seccion-privadas",
  "seccion-catalogo",
  "seccion-galeria",
  "seccion-nosotros",
  "contacto",
];

for (const id of sections) {
  const el = page.locator(`#${id}`);
  if ((await el.count()) === 0) {
    console.log(`MISSING #${id}`);
    continue;
  }
  await el.scrollIntoViewIfNeeded();
  await page.waitForTimeout(700);
  await el.screenshot({ path: `${outDir}/section-${id}.png` });
  console.log(`captured #${id}`);
}

// Footer + full page.
await page.locator("footer").scrollIntoViewIfNeeded();
await page.waitForTimeout(700);
await page.locator("footer").screenshot({ path: `${outDir}/footer.png` });

await page.screenshot({ path: `${outDir}/full.png`, fullPage: true });
console.log("captured full page");

await browser.close();
console.log("DONE");
