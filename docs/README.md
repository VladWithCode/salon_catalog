# Chenacolo — Design & Implementation Plans

This directory holds the per-page and per-system plans for the **Chenacolo** website redesign: a high-end event-salon marketing site (in Spanish) rebuilt as a **Next.js 16 (App Router) + React 19 + Tailwind CSS 4 + shadcn/ui** app, with **Framer Motion** for animation and the existing **Go backend** as the data source.

The redesign will be done **page by page** so that the design system can be validated against real layouts before being rolled out across the site.

## Scope (this redesign)

| In scope | Out of scope (for now) |
| --- | --- |
| Public marketing pages: **Home**, **Servicios**, **Experiencia** (gallery), **Catálogo** (browse + product), **Solicitar Cotización**, **Contacto**, **Reservaciones**, **Política de Privacidad / Términos / Cookies** | Admin / dashboard (`/panel/*`) — stays in Go for now |
| Shared layout chrome (header, footer, base layout) | The Go backend itself (we only add minimal read-only JSON endpoints as needed) |
| Spanish only (i18n-ready copy structure, but no English UI) | |
| Visual identity refinement (palette + typography polish) | Full rebrand |

## Document index

| File | Purpose |
| --- | --- |
| `00-project-setup.md` | Stack additions, environment config, asset migration, API contract with the Go backend, folder layout. |
| `01-design-system.md` | Design tokens: colors, typography, spacing, radii, shadows, motion. The single source of truth that all pages reference. |
| `02-shared-layout.md` | Root `layout.tsx`, `<SiteHeader />`, `<SiteFooter />`, navigation, mobile menu, contact-strip component, social links, metadata defaults. |
| `03-home-page.md` | **Stage 1** — the home page (`app/page.tsx`) broken into sections. |
| `04-services-page.md` | *Stage 2* (to be written) |
| `05-catalog-page.md` | *Stage 3* — catalog browse with categories, search, pagination, product cards. |
| `06-product-page.md` | *Stage 4* — product detail. |
| `07-experience-page.md` | *Stage 5* — gallery + salon tour. |
| `08-quote-request-page.md` | *Stage 6* — multi-section form with cart items. |
| `09-reservations-page.md` | *Stage 7* — date availability + simple form. |
| `10-static-legal-pages.md` | *Stage 8* — privacy, terms, cookies (mostly markdown). |

## How the plans are structured

Each page plan follows the same template so they're easy to compare and review:

1. **Goal** — what this page is for, who visits it, what they should feel/do.
2. **Source of truth** — link to the original Go template for reference, and the data it consumes.
3. **Sections / structure** — vertical breakdown of the page top-to-bottom, with the content intent of each section.
4. **Design decisions** — what we keep, refine, or replace versus the original, and why.
5. **Components & dependencies** — which shadcn primitives, custom components, and motion patterns it uses.
6. **Data contract** — exactly what the page fetches and from where.
7. **Accessibility & responsive notes** — breakpoints, focus order, reduced-motion behavior.
8. **Out of scope** — what this page explicitly does not include (cart, auth, etc.) so we don't scope-creep.

## Key decisions already locked in

These were decided up-front and are not re-litigated per page:

- **Backend strategy:** keep the Go backend; the new Next.js app calls its JSON endpoints. No reimplementation of business logic.
- **First page:** Home (foundation page; everything else inherits from it).
- **Visual direction:** refine the existing warm brown / cream / gold family — modernise the scale, deepen the dark, lift the gold — without rebranding.
- **Stack:** shadcn/ui primitives + Framer Motion. Tailwind 4 theme tokens drive both.
- **Assets:** copy `web/static/assets/*` from the Go project into `public/assets/` so the new app has real content.
- **Language:** Spanish only; copy lives in a typed `lib/copy/` module so we can add i18n later.

See `01-design-system.md` for the full token set and `02-shared-layout.md` for the chrome before reading any page plan.
