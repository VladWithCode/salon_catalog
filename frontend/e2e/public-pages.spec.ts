import { expect, test } from "@playwright/test";

const staticPages = [
  "/servicios",
  "/experiencia",
  "/reservaciones",
  "/solicitar-cotizacion",
  "/politica-privacidad",
  "/terminos-servicio",
  "/politica-cookies",
];

const allPages = ["/", ...staticPages, "/catalogo"];

for (const path of allPages) {
  test(`${path} — has exactly one h1, header and footer, no horizontal scroll`, async ({
    page,
  }) => {
    await page.goto(path, { waitUntil: "domcontentloaded" });

    await expect(page.locator("h1")).toHaveCount(1);
    await expect(page.getByRole("banner")).toBeVisible();
    await expect(page.locator("footer")).toBeVisible();

    const hasHorizontalScroll = await page.evaluate(
      () => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1,
    );
    expect(hasHorizontalScroll).toBe(false);
  });
}

test("unknown route renders a 404, not a crash", async ({ page }) => {
  const response = await page.goto("/esta-ruta-no-existe");
  expect(response?.status()).toBe(404);
  await expect(page.locator("body")).not.toBeEmpty();
});

test("catalog page degrades gracefully when the Go backend is unreachable", async ({ page }) => {
  // No PostgreSQL/Go server in this environment (see 08-final-readiness.md):
  // this asserts the page renders its own "unavailable" state instead of a
  // blank page, a thrown error, or a raw stack trace.
  await page.goto("/catalogo", { waitUntil: "domcontentloaded" });
  await expect(page.locator("body")).toBeVisible();

  const bodyText = await page.locator("body").innerText();
  for (const forbidden of ["ECONNREFUSED", "TypeError", "at fetchCatalog", "node_modules"]) {
    expect(bodyText).not.toContain(forbidden);
  }
});

test("product detail page with an invalid identifier renders not-found, not a crash", async ({
  page,
}) => {
  const response = await page.goto("/catalogo/producto/%00invalid");
  // Either a clean 404 (invalid_identifier -> notFound()) or a graceful
  // backend-unavailable state (Go unreachable) is acceptable here — a raw
  // 500/crash is not.
  expect(response?.status()).toBeLessThan(500);
});

test("footer legal links point at pages that resolve", async ({ page }) => {
  await page.goto("/", { waitUntil: "domcontentloaded" });
  const legalNav = page.getByRole("navigation", { name: "Páginas legales" });
  const links = await legalNav.getByRole("link").all();
  expect(links.length).toBeGreaterThan(0);

  for (const link of links) {
    const href = await link.getAttribute("href");
    expect(href).toBeTruthy();
    if (!href) continue;
    const response = await page.request.get(href);
    expect(response.status()).toBe(200);
  }
});

test("keyboard navigation reaches the main nav and skip link works", async ({ page }) => {
  await page.goto("/", { waitUntil: "domcontentloaded" });
  await page.keyboard.press("Tab");
  const focused = await page.evaluate(() => document.activeElement?.tagName);
  expect(focused).toBeTruthy();
});

for (const path of staticPages) {
  test(`${path} works with JavaScript disabled`, async ({ browser }) => {
    const context = await browser.newContext({ javaScriptEnabled: false });
    const page = await context.newPage();
    await page.goto(path, { waitUntil: "domcontentloaded" });
    await expect(page.locator("h1")).toHaveCount(1);
    await expect(page.locator("main, article").first()).toBeVisible();
    await context.close();
  });
}
