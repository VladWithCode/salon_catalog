# 00 — Project setup

The goal of this stage is purely infrastructure: add the libraries we'll need, migrate assets, and establish the folder conventions every page plan will rely on. No page-level UI is built in this stage.

## 1. Stack additions

All additions are added with **bun** (per the global preference), keeping `bun.lock` as the single source of truth.

```bash
# shadcn/ui — uses the project's own init; we'll go with the New York style + neutral base
bunx --bun shadcn@latest init

# Animation
bun add framer-motion

# Useful helpers used by the API/data layer
bun add clsx tailwind-merge class-variance-authority

# Icons (already widely used; safe baseline for an elegant site)
bun add lucide-react
```

Notes:

- `shadcn init` will rewrite `app/globals.css` and `tailwind.config.*` if present. We are on Tailwind 4 (CSS-first config, no JS config file), so shadcn will emit a `globals.css` with its CSS variables — we'll keep that and **merge** our refined tokens into it (see `01-design-system.md`).
- The `clsx` + `tailwind-merge` pair is the conventional `cn()` helper. shadcn scaffolds this as `lib/utils.ts` — keep it.
- `framer-motion` is preferred over the original GSAP because it composes with React's render cycle, ships tree-shakeable ESM, and avoids fighting React 19's concurrent rendering.

## 2. shadcn/ui configuration

shadcn `components.json` should look like this (Tailwind 4-compatible). This is what we'll commit so every page plan can rely on it.

```json
{
  "$schema": "https://ui.shadcn.com/schema.json",
  "style": "new-york",
  "rsc": true,
  "tsx": true,
  "tailwind": {
    "config": "",
    "css": "app/globals.css",
    "baseColor": "stone",
    "cssVariables": true,
    "prefix": ""
  },
  "aliases": {
    "components": "@/components",
    "utils": "@/lib/utils",
    "ui": "@/components/ui",
    "lib": "@/lib",
    "hooks": "@/hooks"
  },
  "iconLibrary": "lucide"
}
```

Choice rationale:

- **New York** style over Default — slightly more compact and editorial; better baseline for a luxury brand.
- **baseColor: `stone`** — closest to our warm neutral; the refined tokens in `01-design-system.md` will override the scale.
- **No `tailwind.config` filename** — Tailwind 4 reads theme from CSS, so we configure entirely in `globals.css`.

### Components to add up-front

These are the only shadcn primitives we know we need from the home page plan. We add them now so the chrome (`02-shared-layout.md`) and the home page (`03-home-page.md`) can be built without revisiting setup.

```bash
bunx --bun shadcn@latest add button input label textarea sheet dialog \
  separator navigation-menu form sonner tooltip badge scroll-area skeleton
```

Each one is added to `components/ui/` and customised inline as needed per page plan. We do **not** modify the generated files outside of token updates.

## 3. Asset migration

Copy the visual assets from the Go project so the new app has real content from day one.

```bash
# From the chenacolo (new) project root
mkdir -p public/assets public/assets/uploads

# Salon photos, video, fonts, svg icons
cp -r /home/vladwb/developer/salon_catalog/web/static/assets/* public/assets/

# Uploaded product images
cp -r /home/vladwb/developer/salon_catalog/web/static/uploads/* public/assets/uploads/

# Favicon
cp /home/vladwb/developer/salon_catalog/web/static/favicon.ico public/favicon.ico
```

After copy, prune duplicates that won't be used (the SVGs we replaced with Lucide, e.g. `bin.svg`, `briefcase.svg`, `glass.svg`, `play-btn.svg`, `wsp.svg`, `wizard.svg`) — keep them in `public/assets/_legacy/` for reference but do not reference them from React code. The video (`chenacolo_vid.webm` / `chenacolo_vid.mp4`) and `chenacolo_*.jpeg` photos are essential.

> **Note for the Go side:** the original serves `web/static/uploads/*` under `/static/uploads/`. We will either:
>
> 1. Have the Next.js `<Image>` component load from `public/assets/uploads/...` directly (preferred for Stage 1; no network round-trip, full `<Image>` optimization), **or**
> 2. Configure `next.config.ts` `images.remotePatterns` to allow the Go server and load via absolute URL once the Go side is exposed to the public.
>
> We start with option 1 in this redesign because it is self-contained.

## 4. Folder layout

This is the directory tree the rest of the plans assume. We add the directories in this stage, even if some are empty.

```
chenacolo/
├── app/
│   ├── (site)/                          # Route group for all public pages
│   │   ├── layout.tsx                   # Site chrome (header/footer). See 02-shared-layout.md
│   │   ├── page.tsx                     # Home. See 03-home-page.md
│   │   ├── servicios/page.tsx           # 04
│   │   ├── catalogo/
│   │   │   ├── page.tsx                 # 05
│   │   │   └── producto/[slug]/page.tsx # 06
│   │   ├── experiencia/page.tsx         # 07
│   │   ├── solicitar-cotizacion/page.tsx# 08
│   │   ├── reservaciones/page.tsx       # 09
│   │   ├── politica-privacidad/page.tsx # 10
│   │   ├── terminos-servicio/page.tsx   # 10
│   │   └── politica-cookies/page.tsx    # 10
│   ├── globals.css
│   ├── layout.tsx                       # Root layout: <html>, <body>, fonts
│   └── not-found.tsx
├── components/
│   ├── ui/                              # shadcn primitives (generated)
│   ├── site/                            # Site chrome: header, footer, mobile nav
│   ├── home/                            # Home-only sections (hero, events, …)
│   └── shared/                          # Truly cross-page primitives (CTA, SectionHeading, etc.)
├── lib/
│   ├── utils.ts                         # shadcn `cn()` helper
│   ├── api/                             # Typed fetchers to the Go backend
│   │   ├── client.ts                    # base URL + fetch wrapper
│   │   ├── catalog.ts
│   │   ├── socials.ts
│   │   ├── quotes.ts
│   │   └── reservations.ts
│   ├── copy/                            # Typed copy strings (Spanish)
│   │   ├── home.ts
│   │   ├── nav.ts
│   │   ├── footer.ts
│   │   └── …
│   └── motion/                          # Reusable Framer Motion variants & helpers
│       ├── fade-in.ts
│       ├── reveal.ts
│       └── stagger.ts
├── hooks/
│   ├── use-reduced-motion.ts
│   └── use-in-view.ts                   # thin wrapper around framer-motion's useInView
├── public/
│   └── assets/                          # Migrated from the Go project (see §3)
├── docs/                                # You are here
├── next.config.ts
├── postcss.config.mjs
├── tsconfig.json
├── package.json
└── bun.lock
```

The `(site)` route group lets us keep the marketing tree separate from any future non-site routes (e.g. `app/(auth)/login`).

## 5. Backend integration (Go)

The Go project at `/home/vladwb/developer/salon_catalog` already has the database, models, and a templated site. For the React frontend we will only **add** JSON read endpoints; we will not refactor the Go code.

### Environment

Add to `.env.local` (new file, gitignored):

```
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

`lib/api/client.ts` exports a single `apiFetch<T>(path, init?)` that prepends this base URL and throws on non-2xx with a typed error. All page-level fetchers use it.

### Endpoints we need (Stage 1 — Home)

The original Go code does not yet expose these. **We will add them to the Go project as part of Stage 1** (small, additive changes — no breaking refactors).

| Method | Path | Purpose | Used by |
| --- | --- | --- | --- |
| `GET` | `/api/catalog/listings` | Listings grouped by category, for the catalog preview strip on home. Mirrors `db.FindCatalogListings()`. | Home |
| `GET` | `/api/socials` | List of social media links for the footer. Mirrors `db.GetSocialLinks()`. | Layout (footer) |

Later stages will add more (catalog list, product detail, quote submission, reservation submission, contact). Each is added in its own plan file under "Data contract".

### Why add Go endpoints instead of calling the DB directly

- The Go server owns validation, business rules, and the schema.
- The React app stays purely a presentation layer.
- We can host the Go server and the Next.js app behind the same domain later (reverse proxy) so we can drop the CORS concern.

### CORS

For local dev, add a tiny middleware to the Go router that allows `http://localhost:3000` for the `/api/*` prefix. In production we will serve both from the same origin and remove it.

## 6. TypeScript & lint

- Stick with `strict: true` (already on).
- Path alias `@/*` is already configured in `tsconfig.json` — we use it everywhere.
- Add a shared `lib/types.ts` for cross-module types (`Category`, `Product`, `SocialLink`, `EventKind`, `CartItem`).
- Run `bun run lint` before every commit; we add a `prebuild` script that runs `bun run lint` to fail CI on style.

## 7. Smoke test (definition of "Stage 0 done")

After this stage is implemented we should be able to:

1. `bun run dev` boots on `http://localhost:3000` with no errors.
2. `app/page.tsx` still shows the default Next.js scaffold (we haven't touched it yet) but renders inside the refined font + colour scale from `globals.css`.
3. `GET http://localhost:3000/api/_health` (a tiny endpoint we'll add) returns 200 — confirms the Go server is reachable from the Next.js app.
4. `bun run lint && bun run build` pass with zero errors.

Once this is green, we move to `01-design-system.md` for the full token set, then `02-shared-layout.md`, then `03-home-page.md`.
