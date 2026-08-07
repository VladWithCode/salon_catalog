import { plugin } from "bun";

// "server-only" is designed to throw unless imported inside Next's own
// server bundler, which swaps it for a no-op at build time. bun test runs
// files directly under Bun's runtime, not Next's bundler, so this preload
// stubs the same no-op Next would produce — it does not change what ships
// in the actual Next build (next.config.ts and the webpack/turbopack layer
// are untouched).
plugin({
  name: "stub-server-only-for-tests",
  setup(build) {
    build.module("server-only", () => ({
      contents: "export {};",
      loader: "js",
    }));
  },
});
