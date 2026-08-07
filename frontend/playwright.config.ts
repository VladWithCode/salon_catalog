import { defineConfig, devices } from "@playwright/test";

// E2E/visual QA for the static, fully-migrated public pages (Fase 8/9).
// Deliberately does not start Go (no PostgreSQL available in this
// environment — see 08-final-readiness.md); webServer here starts Next
// only, against whichever GO_API_BASE_URL is already configured. Pages
// that need Go data (catalog, product detail) will show their real
// "backend unavailable" state when Go isn't reachable — that state is
// itself one of the things this suite verifies.
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: 0,
  reporter: [["list"]],
  use: {
    baseURL: "http://127.0.0.1:3100",
    trace: "retain-on-failure",
  },
  projects: [
    { name: "mobile-360", use: { viewport: { width: 360, height: 800 } } },
    { name: "tablet-768", use: { viewport: { width: 768, height: 1024 } } },
    { name: "small-desktop-1024", use: { viewport: { width: 1024, height: 768 } } },
    { name: "desktop-1280", use: { viewport: { width: 1280, height: 800 } } },
    { name: "wide-1440", use: { viewport: { width: 1440, height: 900 } } },
    {
      name: "chromium-js-disabled",
      use: { ...devices["Desktop Chrome"], javaScriptEnabled: false },
    },
    // Cross-engine coverage (03-home-page.md §10.7 asks for Chrome, Safari
    // and Firefox). Playwright's WebKit is the same engine Safari ships, but
    // it is not Safari itself — results are reported per engine, never as
    // "Safari verified".
    {
      name: "firefox",
      use: { ...devices["Desktop Firefox"], viewport: { width: 1280, height: 800 } },
      testMatch: /public-pages\.spec\.ts/,
    },
    {
      name: "webkit",
      use: { ...devices["Desktop Safari"], viewport: { width: 1280, height: 800 } },
      testMatch: /public-pages\.spec\.ts/,
    },
    // prefers-reduced-motion is a real media-query state, not just a code
    // path: this project drives the whole public-pages suite with it forced
    // on, so a reduced-motion regression (content hidden instead of merely
    // still) fails loudly.
    {
      name: "reduced-motion",
      use: {
        ...devices["Desktop Chrome"],
        viewport: { width: 1280, height: 800 },
        reducedMotion: "reduce",
      },
      testMatch: /public-pages\.spec\.ts/,
    },
  ],
  webServer: {
    command: "bunx next start -p 3100",
    url: "http://127.0.0.1:3100",
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
    // Fase 10: a real, disposable Go + PostgreSQL instance is available in
    // this run (see 10-production-cutover.md) at 127.0.0.1:8090. When it is
    // not running, GO_API_BASE_URL still needs to be syntactically
    // configured so getGoAPIBaseURL() doesn't throw before each fetcher's
    // own try/catch can turn a failed connection into its "unavailable"
    // result — that graceful-degradation path is exercised by
    // e2e/public-pages.spec.ts regardless of which port is live.
    env: { GO_API_BASE_URL: process.env.GO_API_BASE_URL ?? "http://127.0.0.1:8090" },
  },
});
