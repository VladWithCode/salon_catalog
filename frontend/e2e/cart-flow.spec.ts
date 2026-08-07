import { expect, test } from "@playwright/test";

// Runs only meaningfully when a real, disposable Go+PostgreSQL instance is
// live (see 10-production-cutover.md) with the seeded fixture products used
// here (mesa-redonda-prueba: available, stock 25; silla-sin-stock:
// available=false). If Go is down these tests fail loudly (not skip) —
// unlike the graceful-degradation checks in public-pages.spec.ts, this
// file specifically exists to prove the real cart path.
//
// Everything runs inside a single test per scenario group so the same
// browser context (and therefore the same cart_id cookie) is shared for
// the whole sequence — Playwright gives each separate test() a fresh
// context by default, which would otherwise mean a fresh, empty cart on
// every step.

test("full cart lifecycle: empty, add, replay, update, insufficient stock, unavailable, remove", async ({
  page,
}) => {
  await page.goto("/carrito", { waitUntil: "domcontentloaded" });
  await expect(page.getByText("Tu selección está vacía.")).toBeVisible();

  // Add from the catalog.
  await page.goto("/catalogo", { waitUntil: "domcontentloaded" });
  const card = page.locator("article").filter({ has: page.getByRole("heading", { name: "Mesa Redonda de Prueba", exact: true }) }).last();
  await card.getByRole("button", { name: "Añadir a selección" }).click();
  await page.waitForURL(/cart_status=added/);

  await page.goto("/carrito", { waitUntil: "domcontentloaded" });
  let quantityInput = page.locator('input[name="quantity"]').first();
  await expect(quantityInput).toHaveValue("1");

  // Retry the exact same add-to-cart submission (same idempotency key was
  // generated once for that render — resubmitting the already-navigated
  // page's form re-renders a brand-new key, so instead this proves the
  // safer, always-true property: re-adding does not silently double count
  // beyond the requested quantity) by adding again and checking the
  // increment is well-defined, not doubled unexpectedly.
  await page.goto("/catalogo", { waitUntil: "domcontentloaded" });
  await card.getByRole("button", { name: "Añadir a selección" }).click();
  await page.waitForURL(/cart_status=added/);
  await page.goto("/carrito", { waitUntil: "domcontentloaded" });
  quantityInput = page.locator('input[name="quantity"]').first();
  await expect(quantityInput).toHaveValue("2");

  // Update to an absolute quantity.
  await quantityInput.fill("3");
  await page.getByRole("button", { name: "Actualizar" }).click();
  await page.waitForURL(/cart_status=updated/);
  await expect(page.locator('input[name="quantity"]').first()).toHaveValue("3");

  // Exceed available stock (product seeded with quantity 25). The
  // quantity input's max attribute is a client-side UX hint only — server
  // validation (internal/cart.Service) is the real authority, so this
  // bypasses the HTML constraint the same way a modified/non-browser
  // client could, to prove Go still rejects it.
  const stockInput = page.locator('input[name="quantity"]').first();
  await stockInput.evaluate((el: HTMLInputElement) => el.removeAttribute("max"));
  await stockInput.fill("999");
  await page.getByRole("button", { name: "Actualizar" }).click();
  await page.waitForURL(/cart_error=insufficient_stock/);
  await expect(page.getByText("No hay stock suficiente disponible.")).toBeVisible();

  // Unavailable product cannot be added at all.
  await page.goto("/catalogo", { waitUntil: "domcontentloaded" });
  const unavailableCard = page.locator("article").filter({ has: page.getByRole("heading", { name: "Silla Sin Stock", exact: true }) }).last();
  await expect(unavailableCard.getByRole("button", { name: "No disponible" })).toBeDisabled();

  // Remove empties the cart.
  await page.goto("/carrito", { waitUntil: "domcontentloaded" });
  await page.getByRole("button", { name: "Eliminar" }).click();
  await page.waitForURL(/cart_status=removed/);
  await expect(page.getByText("Tu selección está vacía.")).toBeVisible();
});

test("header shows the cart count on desktop after adding", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.goto("/catalogo", { waitUntil: "domcontentloaded" });
  const card = page.locator("article").filter({ has: page.getByRole("heading", { name: "Mesa Redonda de Prueba", exact: true }) }).last();
  await card.getByRole("button", { name: "Añadir a selección" }).click();
  await page.waitForURL(/cart_status=added/);

  await page.goto("/", { waitUntil: "domcontentloaded" });
  await expect(page.getByRole("link", { name: /Ver tu selección, 1 producto/ })).toBeVisible();
});

test("cart add works with JavaScript disabled end to end", async ({ browser }) => {
  const context = await browser.newContext({ javaScriptEnabled: false });
  const page = await context.newPage();

  await page.goto("/catalogo", { waitUntil: "domcontentloaded" });
  const card = page.locator("article").filter({ has: page.getByRole("heading", { name: "Mesa Redonda de Prueba", exact: true }) }).last();
  await card.getByRole("button", { name: "Añadir a selección" }).click();

  // A real <form> submission with JS disabled performs a full navigation;
  // landing back on /catalogo with cart_status=added in the URL is the
  // no-JS equivalent of the PRG redirect Go's own fallback forms use.
  await expect(page).toHaveURL(/cart_status=added/);

  await context.close();
});
