import { expect, test } from "@playwright/test";

// Fase 14 compliance regressions. Each test here exists because the audit
// found a real defect (or a requirement with no automated guard at all) —
// they are not duplicates of public-pages.spec.ts.

/**
 * WCAG contrast, computed from the colours the browser actually resolves,
 * not from the token values in globals.css. The audit found the brand gold
 * (--accent, 3.42:1 on cream) being used as small text on light surfaces,
 * which fails 1.4.3. --accent-strong / --accent-on-dark replaced it there;
 * this locks that in so a future edit cannot silently reintroduce a
 * sub-4.5:1 eyebrow or link.
 */
function relativeLuminance([r, g, b]: number[]): number {
  const channel = (value: number) => {
    const v = value / 255;
    return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
  };
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b);
}

function contrastRatio(a: number[], b: number[]): number {
  const la = relativeLuminance(a);
  const lb = relativeLuminance(b);
  const [hi, lo] = la > lb ? [la, lb] : [lb, la];
  return (hi + 0.05) / (lo + 0.05);
}

function parseRGB(value: string): number[] | null {
  const match = value.match(/rgba?\(([^)]+)\)/);
  if (!match) return null;
  const parts = match[1].split(",").map((part) => Number.parseFloat(part.trim()));
  if (parts.length < 3 || parts.some((part) => Number.isNaN(part))) return null;
  // Fully transparent elements carry no visible text.
  if (parts.length >= 4 && parts[3] === 0) return null;
  return [parts[0], parts[1], parts[2]];
}

test("eyebrows and small gold text meet WCAG AA 4.5:1 on every home surface", async ({
  page,
}) => {
  await page.goto("/", { waitUntil: "domcontentloaded" });

  const samples = await page.evaluate(() => {
    // Walks up for the nearest ancestor that actually paints an opaque
    // surface. An ancestor carrying a gradient or photo is reported
    // separately: its rendered colour varies per pixel, so a single
    // computed value would be a fiction. Those cases are handled in the
    // markup instead (the offer-intro scrim is near-opaque cocoa) and are
    // asserted here only to the extent that they never fall back to the
    // undarkened brand gold.
    function resolve(element: Element): { background: string; overImagery: boolean } {
      let current: Element | null = element;
      let cameFrom: Element | null = null;
      let overImagery = false;
      while (current) {
        const style = getComputedStyle(current);
        if (style.backgroundImage && style.backgroundImage !== "none") {
          overImagery = true;
        }
        // The photo and its gradient scrim are absolutely-positioned
        // *siblings* of the copy, not ancestors of it (see ImageTile and
        // the home hero), so an ancestor-only walk would never see them.
        for (const child of Array.from(current.children)) {
          if (child === cameFrom) continue;
          const childStyle = getComputedStyle(child);
          if (childStyle.position !== "absolute" && childStyle.position !== "fixed") continue;
          if (child.tagName === "IMG" || childStyle.backgroundImage !== "none") {
            overImagery = true;
          }
        }
        const parts = style.backgroundColor.match(/rgba?\(([^)]+)\)/);
        if (parts) {
          const values = parts[1].split(",").map((p) => Number.parseFloat(p.trim()));
          const alpha = values.length >= 4 ? values[3] : 1;
          if (alpha >= 0.85) {
            return { background: style.backgroundColor, overImagery };
          }
        }
        cameFrom = current;
        current = current.parentElement;
      }
      return {
        background: getComputedStyle(document.body).backgroundColor,
        overImagery,
      };
    }

    return Array.from(document.querySelectorAll<HTMLElement>(".type-eyebrow")).map(
      (element) => {
        const resolved = resolve(element);
        return {
          text: (element.textContent ?? "").trim().slice(0, 40),
          color: getComputedStyle(element).color,
          background: resolved.background,
          overImagery: resolved.overImagery,
        };
      },
    );
  });

  expect(samples.length).toBeGreaterThan(0);

  // The exact rgb() of the two accessible gold tokens. Any eyebrow that is
  // not one of these — most importantly the raw 3.42:1 --accent — fails
  // regardless of which surface it sits on.
  const allowedGold = ["rgb(126, 85, 37)", "rgb(216, 162, 100)"];
  let measured = 0;

  for (const sample of samples) {
    const foreground = parseRGB(sample.color);
    const background = parseRGB(sample.background);
    if (!foreground) continue;

    if (sample.overImagery) {
      expect(
        allowedGold.includes(sample.color) || sample.color === "rgb(250, 246, 238)",
        `eyebrow "${sample.text}" over imagery uses ${sample.color}, not an accessible token`,
      ).toBe(true);
      continue;
    }

    if (!background) continue;
    measured += 1;
    const ratio = contrastRatio(foreground, background);
    expect(
      ratio,
      `eyebrow "${sample.text}" — ${sample.color} on ${sample.background}`,
    ).toBeGreaterThanOrEqual(4.5);
  }

  // Guard against the walk silently classifying everything as "over
  // imagery" and asserting nothing numeric.
  expect(measured).toBeGreaterThan(0);
});

test("home renders all thirteen narrative sections", async ({ page }) => {
  await page.goto("/", { waitUntil: "domcontentloaded" });
  // 03-home-page.md §3: hero, offer intro, six event sections, catalog
  // preview, gallery preview, about strip, contact, closing CTA.
  const sections = page.locator("article > section");
  await expect(sections).toHaveCount(13);
});

test("contact form submits and confirms with JavaScript disabled", async ({ browser }) => {
  const context = await browser.newContext({ javaScriptEnabled: false });
  const page = await context.newPage();

  await page.goto("/#contacto", { waitUntil: "domcontentloaded" });

  // A native form submission (no JS) sends urlencoded, not JSON. Before the
  // Fase 14 fix this returned a bare "415 unsupported_media_type" JSON body
  // straight into the browser window.
  await page.locator("#contact-name").fill("Prueba Sin JavaScript");
  await page.locator("#contact-phone").fill("6182593026");
  await page.locator("#contact-submit-button").click();

  await expect(page).toHaveURL(/contacto=enviado/);
  await expect(page.getByRole("status")).toBeVisible();

  await context.close();
});

test("WhatsApp FAB is reachable at every breakpoint", async ({ page }) => {
  // Previously carried md:hidden lg:inline-flex, leaving 768–1023px with no
  // WhatsApp affordance at all.
  for (const width of [360, 768, 1024, 1280, 1440]) {
    await page.setViewportSize({ width, height: 800 });
    await page.goto("/", { waitUntil: "domcontentloaded" });
    await expect(
      page.getByRole("link", { name: "Chatea con nosotros por WhatsApp" }),
      `WhatsApp FAB missing at ${width}px`,
    ).toBeVisible();
  }
});

test("no unexpected console errors on the home page", async ({ page }) => {
  const errors: string[] = [];
  page.on("console", (message) => {
    if (message.type() === "error") errors.push(message.text());
  });
  page.on("pageerror", (error) => errors.push(String(error)));

  await page.goto("/", { waitUntil: "load" });
  await page.waitForTimeout(1500);

  // Autoplay rejection is a browser policy decision about the muted hero
  // video, not an application fault, and it is the only message this page is
  // permitted to produce.
  const unexpected = errors.filter(
    (message) => !/play\(\) (failed|request)|NotAllowedError|autoplay/i.test(message),
  );
  expect(unexpected, `unexpected console errors: ${unexpected.join(" | ")}`).toHaveLength(0);
});
