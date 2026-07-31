# 02 — Shared layout (site chrome)

Every public page in the redesign wraps the same chrome: a top-of-page **site header** (with navigation), a **footer** with contact details and links, and a **page-level layout** that sets up common metadata and language. This file plans the chrome once so each page plan just composes with it.

The chrome is intentionally **light, restrained, and editorial**. The original used a translucent dark header that turned solid on scroll; we refine that into a header that *is* the page's identity — never a strip of UI floating on top of the content.

## 1. File tree

```
app/
├── layout.tsx               # Root: <html>, <body>, fonts, theme, Sonner toaster
├── (site)/
│   ├── layout.tsx           # Site chrome: <SiteHeader />, <main>, <SiteFooter />, <WhatsAppFab />
│   ├── page.tsx             # Home (Stage 1 — see 03-home-page.md)
│   └── …
components/
├── site/
│   ├── site-header.tsx
│   ├── site-footer.tsx
│   ├── nav-menu.tsx         # Desktop navigation menu
│   ├── mobile-nav.tsx       # Sheet-based mobile drawer
│   ├── logo.tsx             # The Villa Chenacolo wordmark
│   ├── contact-strip.tsx    # Optional slim contact bar used inside sections
│   ├── whatsapp-fab.tsx     # Floating WhatsApp button
│   ├── social-icons.tsx     # Facebook, Instagram, WhatsApp inline SVGs
│   └── footer-services.tsx  # Footer link columns (server component, takes data)
lib/
├── copy/
│   ├── nav.ts               # Typed nav links (label, href, optional description)
│   ├── footer.ts            # Footer column copy
│   └── contact.ts           # Address, phone, email, hours
└── api/
    └── socials.ts           # Fetches social links from Go backend
```

## 2. Root layout (`app/layout.tsx`)

The root layout only sets up document-level concerns. Page-specific layout is the `(site)` group layout.

Responsibilities:

- `<html lang="es">` (Spanish site)
- Body gets `font-sans antialiased bg-background text-foreground` classes
- Loads both Google fonts via `next/font/google` and exposes them as CSS variables
- Renders the Sonner toaster for the few client-side notifications we have
- Sets default `metadata` (title, description, theme color) — pages override per-page

Outline (no real code yet — illustrative):

```tsx
import { Inter, Playfair_Display } from "next/font/google";
import { Toaster } from "sonner";
import "./globals.css";

const inter = Inter({ /* … */ variable: "--font-sans" });
const playfair = Playfair_Display({ /* … */ variable: "--font-display" });

export const metadata = {
  metadataBase: new URL("https://villachenacolo.com"),
  title: { default: "Villa Chenacolo", template: "%s · Villa Chenacolo" },
  description: "Salón de eventos de alto nivel en Durango. Bodas, quinceañeras, bautizos, eventos corporativos y más.",
  openGraph: { /* … */ },
  icons: { icon: "/favicon.ico" },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="es" className={`${inter.variable} ${playfair.variable}`}>
      <body className="min-h-screen bg-background text-foreground font-sans antialiased">
        {children}
        <Toaster richColors position="top-center" />
      </body>
    </html>
  );
}
```

## 3. Site layout (`app/(site)/layout.tsx`)

A simple server component that wraps the chrome around `{children}`. It does the social-links fetch once and passes to footer (RSC pattern — no client waterfall).

```tsx
import { SiteHeader } from "@/components/site/site-header";
import { SiteFooter } from "@/components/site/site-footer";
import { WhatsAppFab } from "@/components/site/whatsapp-fab";
import { getSocialLinks } from "@/lib/api/socials";

export default async function SiteLayout({ children }: { children: React.ReactNode }) {
  const socials = await getSocialLinks();
  return (
    <>
      <SiteHeader />
      <main id="content" className="pt-16 md:pt-20">
        {children}
      </main>
      <SiteFooter socials={socials} />
      <WhatsAppFab />
    </>
  );
}
```

The `<main>` top-padding offsets the fixed header height (defined in the header itself).

## 4. Site header (`components/site/site-header.tsx`)

The original used a transparent header that turned solid on scroll. We refine that to a **calm, paper-toned header with a hairline gold underline on scroll**. The header is sticky on scroll, but the visual change is subtle.

### 4.1 Visual

- **Default (top of page):** transparent background, foreground text colour, no shadow.
- **On scroll (>16px):** `bg-background/85 backdrop-blur-md`, soft `shadow-soft`, hairline gold underline (`after:bg-accent/40`, 1px, sliding in from the centre in 300ms).
- **Height:** `h-16` mobile, `h-20` desktop.
- **Position:** `sticky top-0 z-50`.

### 4.2 Structure

```
<header>
  <Container>
    <Logo />                # Left-aligned wordmark
    <NavMenu />             # Center on desktop, hidden on mobile
    <div>                   # Right side
      <PhoneLink />         # desktop only: small phone number
      <QuoteCTA />          # desktop only: "Solicitar cotización" outline button
      <MobileMenuButton />  # mobile only
    </div>
  </Container>
  <MobileNav />             # sheet, mounted on demand
</header>
```

### 4.3 Components

#### `<Logo />` (`components/site/logo.tsx`)

Renders the existing wordmark using `next/image`. The image is the existing `public/assets/logo_name.webp` (already in the migrated assets).

- On a light surface, render the **dark** version: `logo_name.webp` (no inversion needed; the source is already dark-toned).
- On a dark surface (e.g. footer, dark hero overlays), use `logo_name_white.png` and apply `invert-0` (the source is already white-toned).
- We pick the variant via a `variant: "light" | "dark"` prop. Default `light` for the header.

Sizes: `h-8 md:h-10 w-auto`.

#### `<NavMenu />` (`components/site/nav-menu.tsx`)

Uses shadcn `navigation-menu` primitive, customised to be **minimalist** (no bulky bordered panel).

Links come from `lib/copy/nav.ts`:

```ts
export const navLinks = [
  { label: "Inicio",          href: "/" },
  { label: "Servicios",       href: "/servicios" },
  { label: "Catálogo",        href: "/catalogo" },
  { label: "Experiencia",     href: "/experiencia" },
  { label: "Contacto",        href: "/#contacto" },        // home anchor
  { label: "Cotizar",         href: "/solicitar-cotizacion", emphasis: true },
] as const;
```

- The first 4 are editorial nav. "Cotizar" is a **ghost button** (outline only) on the right, distinct from the nav items but inside the header so it's always reachable.
- Active state: a 2px gold underline animates in below the current item (framer-motion `layoutId` on a single moving `<motion.span>`).
- The desktop nav is `hidden md:flex`.

#### `<MobileMenuButton />` + `<MobileNav />` (`components/site/mobile-nav.tsx`)

- A 44×44 icon button (Lucide `Menu` / `X`) for accessibility.
- Opens a shadcn `Sheet` from the right (`side="right"`), `w-[min(20rem,90vw)]`.
- Sheet contents: full-height flex column with the logo at the top, the nav links as large rows (`text-lg py-4`), the address & phone at the bottom, and a "Cotizar" filled button at the very bottom.
- Sheet transitions: 300ms slide + fade, respects reduced motion.
- Focus is trapped while open; `Esc` closes; click on backdrop closes; restores focus to the trigger button on close.

### 4.4 Header behaviour

- **Scroll detection** with a `useEffect` that toggles a `data-scrolled` attribute. CSS handles the visual change.
- **Sticky** on all pages, but on the home page the first viewport's hero is meant to feel *over* the header — the transparent state handles that. On other pages the header starts in the scrolled state (we add `data-scrolled` initially based on the route).
- **Anchor links** like `/#contacto` work because the home page renders an `<section id="contacto">`. On other pages the browser will navigate to `/#contacto` and the home page's anchor handling kicks in (we use `next/link` for client-side nav).

## 5. Site footer (`components/site/site-footer.tsx`)

A server component that takes the social links data and renders four columns + a legal strip.

### 5.1 Visual

- `bg-primary text-primary-foreground` (the deep cocoa) with a thin gold hairline at the top (`h-px bg-accent/30`).
- Generous vertical padding: `py-20 md:py-24`.
- Maximum content width `max-w-7xl mx-auto px-6 md:px-10`.

### 5.2 Structure

```
<footer>
  <GoldHairline />                                    # decorative
  <div class="grid grid-cols-1 md:grid-cols-12 gap-12">
    <BrandColumn />        # col-span-4: logo, tagline, socials
    <ServicesColumn />     # col-span-2: links to event sections on home
    <ExploreColumn />      # col-span-2: Inicio, Catálogo, Galería, Contacto
    <ContactColumn />      # col-span-4: address, phone, email, hours
  </div>
  <LegalStrip />           # border-t, copyright + privacy/terms/cookies links
</footer>
```

Copy in `lib/copy/footer.ts` and `lib/copy/contact.ts` (typed, easy to update):

```ts
// lib/copy/contact.ts
export const contact = {
  brand: "Villa Chenacolo",
  tagline: "Creamos momentos inolvidables con servicios de planificación de eventos excepcionales y de alto nivel.",
  address: "Entronque, 34234 Ignacio López Rayón, Dgo.",
  phone: "618 259 3026",
  phoneHref: "tel:+526182593026",
  email: "villachenacolo@gmail.com",
  hours: [
    { day: "Lun – Vie", hours: "9:00 – 18:00" },
    { day: "Sáb – Dom", hours: "9:00 – 18:00" },
    { day: "Dom",       hours: "Con cita" },
  ],
  whatsapp: "https://wa.me/526182593026?text=Hola!%20Me%20gustar%C3%ADa%20m%C3%A1s%20informaci%C3%B3n%20sobre%20su%20sal%C3%B3n.",
} as const;
```

### 5.3 Brand column

- Logo (light variant).
- Tagline (`text-body-sm text-primary-foreground/70`).
- Social icons (Facebook, Instagram, WhatsApp) — `h-5 w-5`, hover: `text-accent`. Links target `_blank` with `rel="noopener"`.

### 5.4 Services column

Links to each event-kind section on the home page (`/#seccion-bodas`, etc.). On non-home pages we render the same hash links (browsers will navigate to home and scroll). Future refinement: server-render the section anchors on the home page with proper ids (covered in `03-home-page.md`).

### 5.5 Contact column

Address as a single line, phone and email as `tel:` / `mailto:` links, hours as a small two-column list (`day | hours`).

### 5.6 Legal strip

`border-t border-primary-foreground/15 mt-16 pt-8` containing:

- Copyright `© {year} Villa Chenacolo. Todos los derechos reservados.`
- Inline links: `Política de Privacidad` · `Términos de Servicio` · `Política de Cookies` — each `text-primary-foreground/60 hover:text-accent text-body-sm`.

## 6. WhatsApp floating action button (`components/site/whatsapp-fab.tsx`)

A fixed bottom-right pill (not a circle — a pill feels more editorial than a chat bubble) that opens WhatsApp in a new tab.

- `fixed bottom-6 right-6 z-40` (below the sheet, above page content).
- Hidden on `print`.
- `bg-[#25D366] text-white` (WhatsApp brand) but kept subtle: small (`h-12 px-5`), `rounded-full`, `shadow-elevated`, label "WhatsApp" + small icon, hidden label on mobile (icon only), full pill on `sm+`.
- Hover: gentle scale `1.03`, 200ms.
- Renders a `<a href={contact.whatsapp} target="_blank" rel="noopener" aria-label="Chatea con nosotros por WhatsApp">`.

## 7. Metadata defaults

`app/layout.tsx` defines defaults; pages override. Each page plan includes the exact `metadata` export it should set (title, description, OG image, canonical).

Default OG image: a curated hero photo (`chenacolo_4.jpeg` or `chenacolo_31.jpeg`) at 1200×630 — we'll generate it once with a small compositing script if needed. Stage 1 uses the existing photo as-is.

## 8. Accessibility

- Skip-to-content link: a visually-hidden `<a href="#content" class="sr-only focus:not-sr-only">` rendered first in `<body>`.
- Header is a `<header>` with a `<nav aria-label="Principal">` inside.
- Footer `<nav aria-label="Servicios">` and `<address>` element for contact info (semantic, exposes the address to screen readers and tools).
- The mobile menu button has `aria-expanded` and `aria-controls`.
- All icon-only buttons have `aria-label`.
- Focus order: header → main → footer. Skip link jumps directly to `<main id="content">`.

## 9. Responsive behaviour

| Breakpoint | Header | Footer | WhatsApp FAB |
| --- | --- | --- | --- |
| `< sm` (mobile) | Hamburger only, no nav rows, no phone | 1 column | Icon only, smaller |
| `sm – md` | Hamburger | 2 columns | Pill with label |
| `md+` | Full nav + phone + CTA | 4 columns | Pill with label |
| `lg+` | Same as md, slightly more padding | 4 columns with more breathing room | Same |

## 10. Definition of "shared layout done"

After this stage, every page plan can assume:

1. The header is rendered, sticky, with all 6 nav items + a "Cotizar" CTA.
2. The footer renders 4 columns with brand, services, explore, contact, and the legal strip.
3. The WhatsApp FAB is always available, bottom-right.
4. Social icons inherit `currentColor` and link to the right URLs.
5. Default metadata is set; pages override what they need.
6. The Spanish locale is active (`<html lang="es">`).
7. The page can be navigated with keyboard only and with a screen reader.
8. `prefers-reduced-motion` is respected by every motion primitive in the chrome.
