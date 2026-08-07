# 03 — Home page (Stage 1)

The home page is the **foundation** of the redesign. Every other page inherits patterns established here: how a hero is composed, how a section reveal works, how event kinds are laid out, how the catalog preview tiles are styled, and how the contact section is paired with the contact form. We are taking the time to get this page right because the visual language we lock in here will repeat across the rest of the site.

The original (`/home/vladwb/developer/salon_catalog/internal/templates/pages/index.templ`) is a long single-page that stacks nine sections. We keep the **same narrative arc** but compress, refine, and add polish.

## 1. Goal

> A prospective client (a bride, a mother planning a quinceañera, an event coordinator) lands on the home page and within 10 seconds feels: *"This is an upscale venue, it looks well-run, I trust them."* By 60 seconds they have either clicked "Solicitar cotización", or they have scrolled to the contact form and started filling it out.

The page must work in three modes:

- **Skim:** hero → quick social proof (logos / "Villa Chenacolo · desde 2014") → CTA.
- **Browse:** scroll through all six event kinds, the catalog preview, the gallery, then act.
- **Read:** each section is rich enough to read slowly.

## 2. Source of truth

- **Original Go template:** `internal/templates/pages/index.templ`
- **Original CSS tokens:** `web/style/styles.css`
- **Data already in DB:** event kinds (Bodas, Quinceañeras, Bautizos, Corporativos, Graduaciones, Fiestas Privadas), product categories, products, social links.
- **Static content:** the editorial copy (taglines, section descriptions) is hardcoded in the original templ file. We move it to `lib/copy/home.ts` so it's typed and central.

## 3. Section breakdown (top to bottom)

The home page is one `<article>` containing eight sections. Each section is a self-contained React component in `components/home/`.

| # | Section | Component | Background | Key content |
| --- | --- | --- | --- | --- |
| 1 | Hero | `<HomeHero />` | Video background | Wordmark, tagline, primary CTA |
| 2 | "Nuestra oferta" intro | `<OfferIntro />` | Cream (`bg-background`) | Eyebrow, title, lede, three feature cards |
| 3 | Bodas | `<EventSection kind="bodas" />` | Cream | Image + copy + checklist + WhatsApp link |
| 4 | Quinceañeras | `<EventSection kind="quinceaneras" />` | Deep cocoa (`bg-primary`) | Image (mirrored) + copy + WhatsApp link |
| 5 | Bautizos | `<EventSection kind="bautizos" />` | Cream | Same pattern |
| 6 | Corporativos | `<EventSection kind="corporativos" />` | Deep cocoa | Same pattern |
| 7 | Graduaciones | `<EventSection kind="graduaciones" />` | Cream | Same pattern |
| 8 | Fiestas privadas | `<EventSection kind="privadas" />` | Deep cocoa | Same pattern |
| 9 | Catalog preview | `<CatalogPreview />` | Tinted photo background | "Conoce nuestro catálogo" + 8 categories × 4 products each + "Ver todos" |
| 10 | Gallery preview | `<GalleryPreview />` | Cream | 6-image mosaic + "Ver galería completa" link |
| 11 | "¿Quiénes somos?" | `<AboutStrip />` | Deep cocoa with pattern | Pull-quote-style statement |
| 12 | Contact | `<ContactSection />` | Photo background with overlay | Form on the left, contact info card on the right |
| 13 | Closing CTA | `<ClosingCta />` | Cream | Final nudge to "Solicitar cotización" |

That's **13** visible sections (the original had 9; we split one into three for breathing room). They alternate background tones (cream / cocoa / cream / cocoa / cream) to give the page rhythm without using dividers.

## 4. Per-section design

### 4.1 Hero (`<HomeHero />`)

**Goal:** establish tone in 2 seconds.

- Full-viewport height (`min-h-[88vh] md:min-h-[92vh]`).
- **Video background:** `chenacolo_vid.webm` (with `.mp4` fallback), autoplay, muted, loop, `playsinline`, `object-cover`. The `<video>` is rendered as the *first* child of an absolutely-positioned wrapper with `inset-0 -z-10`.
- **Poster image:** `chenacolo_3.jpeg` shown until the video is ready. `<Image fill priority placeholder="blur" blurDataURL=…>`.
- **Overlay:** `bg-gradient-to-b from-primary/55 via-primary/40 to-primary/70` so the wordmark and CTA are always legible over any frame of the video.
- **Foreground content (centred):**
  - **Wordmark:** the migrated `logo_name.webp` / `.png`, `h-24 md:h-36 lg:h-44 w-auto` with `invert` (the source is dark; we want it light over the dark overlay) — but verify: if the source is already light, drop the invert. *Action item in implementation: open the file, choose correct asset.*
  - **Tagline:** `text-eyebrow` ("Sala de acontecimientos especiales") in `text-primary-foreground/80`, `mb-6`.
  - **Title:** `text-display-xl md:text-display-2xl font-display font-medium` reading *"Donde los momentos importantes se vuelven recuerdos."* (refined; can be tuned).
  - **CTA:** `<Button size="lg">` "Solicitar cotización" linking to `/solicitar-cotizacion`, `bg-accent text-accent-foreground hover:bg-accent/90`, with a tiny chevron-down inside the button as a "scroll cue". An additional ghost link "Ver la experiencia" links to `/#experiencia` (anchors to the gallery section).
- **Entrance:** wordmark fades + rises 12px on mount (0.8s, `ease-elegant`). Tagline 0.1s after. Title 0.25s after. CTA 0.4s after. No scroll trigger.
- **Reduced motion:** all entrance transitions degrade to instant.
- **Performance:** the video is **preloaded as `metadata`**, not `auto`, to avoid eating bandwidth on slow connections. The poster is `priority` so first paint is fast.

### 4.2 Offer intro (`<OfferIntro />`)

**Goal:** explain *what* the venue is, then tease the three main service pillars.

- `py-section` vertical padding.
- **Eyebrow:** "NUESTRA OFERTA" in `text-eyebrow text-accent`, centred.
- **Title:** `text-display-lg font-display` "Un lugar que *eleva* cualquier celebración." (the italic word is in Playfair italic).
- **Lede:** one paragraph (`text-body-lg max-w-prose mx-auto text-center text-muted-foreground`).
- **Three feature cards** in a `grid md:grid-cols-3 gap-6`:
  1. **Servicios** — "¿Cómo podemos ayudarte?" → links to `/servicios`. Image `chenacolo-st-2.jpg`.
  2. **Catálogo** — "Mobiliario y piezas decorativas" → links to `/catalogo`. Image `chenacolo_18.jpeg`.
  3. **Experiencia** — "Recorre el salón" → links to `/experiencia`. Image `chenacolo-st-5.jpg`.

Each card: aspect ratio `[5/4]`, `rounded-xl overflow-hidden`, image `object-cover`, dark overlay `from-primary/40 to-primary/70`, text bottom-left, "Conoce más →" link bottom-right. Hover: image scales 1.04 over 500ms.

- **Motion:** `<RevealOnView>` wraps the whole section, `<StaggerGroup>` wraps the 3 cards with `staggerStep={0.08}`.

### 4.3 Event sections (`<EventSection />`, six instances)

**Goal:** explain what the venue offers for each event kind and route the visitor to WhatsApp for a personalised chat.

- **Layout pattern:** 2-column grid (`grid lg:grid-cols-12 gap-12 items-center`). Image takes 5/12, copy takes 7/12. On odd-indexed sections, the image is on the right (CSS order utilities).
- **Image:** `<Image>` with `aspect-[4/3]`, `rounded-xl`, subtle `shadow-card`. We pick the photo from the migrated assets per event (the original's choices are good; we may refine a couple).
- **Copy column:**
  - **Eyebrow:** event number, e.g. "01" in `text-eyebrow text-accent`.
  - **Title:** event name in `text-display-md font-display`.
  - **Gold rule:** a 2px line, 3rem wide, in `bg-accent` — the same flourish the original used but cleaner.
  - **Lede:** one paragraph (`text-body-lg text-muted-foreground`).
  - **Checklist:** 5–7 items as a vertical list, each with a Lucide `Check` icon (`h-4 w-4 text-accent`) and the item text. `space-y-3`.
  - **Footnote (optional):** some sections have a "🛠️ El mobiliario se renta por separado." style note in `text-body-sm text-muted-foreground`.
  - **CTA:** an outline button "Me interesa saber más" linking to a pre-filled WhatsApp URL (the same `wa.me/...` deep links the original used, kept identical so the existing flow isn't broken).
- **Background:** alternates between `bg-background` and `bg-primary text-primary-foreground`. Cocoa sections invert the eyebrow colour, the gold rule stays gold (it's a true brand colour), and the checklist icons stay gold.
- **Motion:** `<RevealOnView>` per section; image scales 1.02 on hover; CTA underline animates in on hover.

The six event kinds data lives in `lib/copy/home.ts`:

```ts
export const eventKinds = [
  { id: "bodas",          num: "01", title: "Bodas",                 image: "chenacolo_11.jpeg",  dark: false, whatsapp: "...", lede: "…", items: [...] },
  { id: "quinceaneras",   num: "02", title: "Quinceañeras & Celebraciones", image: "chenacolo-st-7.jpg", dark: true,  whatsapp: "...", lede: "…", items: [...] },
  { id: "bautizos",       num: "03", title: "Bautizos y Comuniones", image: "chenacolo-st-9.jpg", dark: false, whatsapp: "...", lede: "…", items: [...] },
  { id: "corporativos",   num: "04", title: "Eventos corporativos",  image: "chenacolo-st-4.png", dark: true,  whatsapp: "...", lede: "…", items: [...] },
  { id: "graduaciones",   num: "05", title: "Graduaciones",          image: "chenacolo-st-5.jpg", dark: false, whatsapp: "...", lede: "…", items: [...] },
  { id: "privadas",       num: "06", title: "Fiestas privadas",      image: "chenacolo-st-3.jpg", dark: true,  whatsapp: "...", lede: "…", items: [...] },
] as const;
```

This makes the section trivially a `eventKinds.map(...)` over a single `<EventSection>`.

### 4.4 Catalog preview (`<CatalogPreview />`)

**Goal:** show the breadth of the catalog and route the visitor to `/catalogo`.

- **Background:** a softly tinted photo (`chenacolo_24.jpeg`) with a warm gradient overlay — `bg-gradient-to-tl from-primary/65 via-primary/45 to-accent/30`. Text is always `text-primary-foreground`.
- **Heading:** eyebrow "CATÁLOGO" + `text-display-lg` "Cada detalle, *pensado* para tu evento." + a short lede.
- **Content:** for **each** of the 8 categories, render a horizontal row with the category name and 4 product cards + a "Ver todos" card. The original was a flat list — we refine to a **scrolling carousel** per category on mobile and a 5-up grid on `lg+`.
- **Carousel:** shadcn `carousel` (Embla) — drag/swipe on touch, snap-to-card, optional autoplay disabled by default, arrows + dots hidden unless on hover (desktop). `prefers-reduced-motion`: arrows only, no auto-scroll, no snap animation.
- **Product card:** `aspect-square`, `rounded-lg`, `overflow-hidden`, image `object-cover`, name overlay at the bottom (`bg-card/95 backdrop-blur-sm`, `p-3`, name in `text-body-sm font-semibold line-clamp-2`). Hover: image scale 1.05, overlay becomes a "Ver producto" pill.
- **"Ver todos" card:** same aspect, but with a `bg-card/95` interior, a Lucide `ArrowRight` icon, and the category name + "Ver todos".
- **Data:** fetched from `GET /api/catalog/listings` (see `00-project-setup.md`). The component is **async** and renders the listings directly. Empty state: a single full-width card "Ver catálogo" linking to `/catalogo`.
- **Motion:** `<RevealOnView>` on the heading; cards inside each row fade up with stagger.

### 4.5 Gallery preview (`<GalleryPreview />`)

**Goal:** make the visitor want to see the rest of the gallery.

- **Background:** `bg-background` (cream).
- **Heading:** eyebrow "GALERÍA" + `text-display-lg` "Queremos presumirte."
- **Mosaic:** 6 images in an `aspect-[3/2] grid grid-cols-6 grid-rows-2 gap-3` with a deliberate layout (a tall image on the left, a wide one on the right, four squares in the middle) — we hand-pick the photos from the migrated assets (`chenacolo_25`, `30`, `31`, `23`, `18`, `11`).
- **Hover:** image scale 1.06, 400ms. A small `+` overlay in the centre fades in.
- **CTA:** "Ver la galería completa →" link to `/experiencia`, `text-accent`, sits at the bottom right.
- **Motion:** `<RevealOnView>` on the heading; each mosaic tile uses `<StaggerItem>`.

### 4.6 About strip (`<AboutStrip />`)

**Goal:** a single moment of "this is who we are" between the gallery and the contact form.

- **Background:** `bg-primary text-primary-foreground` with a soft texture pattern (the migrated `bg_pattern.png` or a CSS-generated subtle noise — we keep it under 8% opacity so it adds warmth without competing with the text).
- **Content:** centred, `max-w-3xl`:
  - Eyebrow "QUIÉNES SOMOS" in `text-eyebrow text-accent`.
  - Two pull-quote paragraphs in `text-display-md font-display`, *italics on the second one*: *"Cada detalle arquitectónico y cada servicio están pensados para que tu evento fluya a la perfección."* (the second sentence is the italicized flourish).
  - A small gold rule (1px, 4rem wide) below.
- **Motion:** `<RevealOnView>` with a slow 0.7s fade for a more "ceremonial" feel.

### 4.7 Contact section (`<ContactSection />`)

**Goal:** the conversion point. The form must be calm, confident, and short.

- **Layout:** a photo background (a wide salon shot, `chenacolo_4.jpeg`) with a warm overlay. On top: a 2-column layout. The form on the left, a "Encuéntranos" info card on the right.
- **The form** is a small client component (`<ContactForm />`) that:
  - Posts to `POST /api/contact-requests` (added to the Go backend in this stage; see Data contract).
  - Uses shadcn `Form` + `react-hook-form` + `zod` for validation. We add `react-hook-form`, `zod`, `@hookform/resolvers` in this stage.
  - Fields: `name` (required), `phone` (required, mexican phone validation), `eventDate` (optional), `message` (optional, `Textarea`).
  - Submit button: `bg-accent text-accent-foreground`, full width on mobile, inline on `sm+`. Loading state with a Lucide `Loader2` spin.
  - Success: replace the form body with a `text-body-lg` success message + a Lucide `CheckCircle2` icon in `text-accent`. Error: inline alert.
  - Toaster: a Sonner toast confirms the success or shows the server error.
- **The info card** on the right is a card with: title "Encuéntranos", paragraph, then a 4-row definition list (Address, Email, Facebook, Phone) with Lucide icons on the left. Below: a "O solicita una cotización →" link to `/solicitar-cotizacion`.
- **Section id:** `id="contacto"` so the header's `/#contacto` and the hero's scroll-cue can anchor to it.
- **Motion:** `<RevealOnView>` on the section; form fields stagger-fade in.

### 4.8 Closing CTA (`<ClosingCta />`)

**Goal:** one last nudge for the visitor who scrolled to the bottom.

- `py-section` with `bg-background` and a centred card: eyebrow "ESTÁS LISTO" + `text-display-md` "Hablemos de tu evento." + a `<Button size="lg">` "Solicitar cotización" to `/solicitar-cotizacion`. Optionally a small "o escríbenos por WhatsApp" link below in `text-accent`.
- This duplicates the contact CTA but for the bottom-of-page reader. If analytics later show it's redundant, we remove it.

## 5. Components inventory

All shadcn primitives we need (mostly already in `00-project-setup.md`):

- `Button`, `Input`, `Textarea`, `Label`, `Form` (rhf), `Sheet` (mobile menu), `Carousel` (Embla), `ScrollArea` (mobile menu), `Separator`, `Skeleton` (loading), `Sonner` toaster.

Custom components in `components/home/`:

- `home-hero.tsx`
- `offer-intro.tsx`
- `event-section.tsx` (takes `EventKind` prop, used 6×)
- `catalog-preview.tsx`
- `gallery-preview.tsx`
- `about-strip.tsx`
- `contact-section.tsx`
- `contact-form.tsx` (client component)
- `closing-cta.tsx`
- `section-heading.tsx` (shared primitive: eyebrow + title + optional lede + optional rule)
- `product-card.tsx` (used in catalog preview; later reused in catalog page)
- `image-tile.tsx` (shared primitive: image with hover scale + optional overlay)

Shared: in `components/shared/`:

- `section-heading.tsx` (used by multiple home sections; later reused on every page)
- `image-tile.tsx` (used by home + experience + catalog)
- `check-list.tsx` (used by every event section; renders a list with a check icon)

## 6. Data contract

### What the home page needs

| Data | Source | Where it's fetched |
| --- | --- | --- |
| `eventKinds` (6) | Hardcoded in `lib/copy/home.ts` | (no fetch) |
| `catalogListings`: `Record<category, Product[]>` (max 4 products per category) | `GET /api/catalog/listings` (new Go endpoint) | RSC at `app/(site)/page.tsx` |
| `socialLinks` | `GET /api/socials` (new Go endpoint) | RSC at `(site)/layout.tsx` (already done in `02-shared-layout.md`) |
| Contact form submission | `POST /api/contact-requests` (new Go endpoint) | Client fetch from `ContactForm` |

### New Go endpoints to add in this stage

These are **small, additive** changes to the Go project. The handler bodies are 5–10 lines each; they wrap existing DB calls.

```go
// In internal/routes/routes.go
router.HandleFunc("GET /api/catalog/listings", withCORS(jsonHandler(getCatalogListings)))
router.HandleFunc("GET /api/socials",          withCORS(jsonHandler(getSocialLinks)))
router.HandleFunc("POST /api/contact-requests", withCORS(jsonHandler(postContactRequest)))
```

`getCatalogListings` reuses `db.FindCatalogListings()`, capped to 4 products per category.
`getSocialLinks` reuses `db.GetSocialLinks()`.
`postContactRequest` validates the body (name, phone, optional eventDate, optional message) and inserts into `contact_requests` via the existing `db` package.

`withCORS` is a tiny middleware:

```go
func withCORS(h http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusNoContent); return
        }
        h(w, r)
    }
}
```

In production the same-origin reverse proxy makes this unnecessary; we keep the dev-only CORS for now.

## 7. Page composition (sketch)

```tsx
// app/(site)/page.tsx
import { HomeHero } from "@/components/home/home-hero";
import { OfferIntro } from "@/components/home/offer-intro";
import { EventSection } from "@/components/home/event-section";
import { CatalogPreview } from "@/components/home/catalog-preview";
import { GalleryPreview } from "@/components/home/gallery-preview";
import { AboutStrip } from "@/components/home/about-strip";
import { ContactSection } from "@/components/home/contact-section";
import { ClosingCta } from "@/components/home/closing-cta";
import { eventKinds, offerIntroCopy, homeCopy } from "@/lib/copy/home";
import { getCatalogListings } from "@/lib/api/catalog";

export const metadata = {
  title: "Villa Chenacolo · Salón de eventos de alto nivel en Durango",
  description: "…",
};

export default async function HomePage() {
  const listings = await getCatalogListings();   // 4 products per category
  return (
    <article>
      <HomeHero />
      <OfferIntro {...offerIntroCopy} />
      {eventKinds.map((k) => (
        <EventSection key={k.id} kind={k} />
      ))}
      <CatalogPreview listings={listings} />
      <GalleryPreview />
      <AboutStrip {...homeCopy.about} />
      <ContactSection />
      <ClosingCta />
    </article>
  );
}
```

## 8. Accessibility & responsive notes

- **Reduced motion:** `<RevealOnView>` and `<StaggerGroup>` collapse to instant when `prefers-reduced-motion: reduce`.
- **Video:** `aria-hidden="true"` and `tabIndex={-1}` so it doesn't appear in the accessibility tree. The poster image carries the visual content.
- **Form:** every field has a visible label (no placeholder-only labels). Error messages are linked via `aria-describedby`.
- **Carousel:** keyboard-navigable (left/right arrows when focused), `aria-roledescription="carrusel"`, slides labelled.
- **Breakpoints:**
  - `< sm`: hero copy `text-display-xl` → `text-display-lg`, event sections stack (image on top), catalog preview is per-category carousel, gallery mosaic becomes 2 columns × 3 rows.
  - `sm – md`: hero `text-display-2xl`, event sections still stacked.
  - `md+`: 2-column event sections, catalog preview becomes a 5-up grid per category.
  - `lg+`: max content width capped at `max-w-7xl mx-auto`.
- **Container query usage:** none required for Stage 1; the responsive utilities above cover it. We add container queries later if a section benefits.

## 9. Out of scope (for this page)

- A live calendar / availability check (that lives on `/reservaciones`).
- Multi-step quote request form with cart (that lives on `/solicitar-cotizacion`).
- Pricing tables (we don't have them in the original; not invented here).
- Customer testimonials — original doesn't have them; we'd need real quotes. **Stage 1 does not fabricate testimonials.** If we add them later it's a separate section in a separate plan.
- A blog / news section — not in the original.
- A live chat widget — replaced by the WhatsApp FAB (`02-shared-layout.md`).

## 10. Definition of "Stage 1 done"

1. All 13 sections render, with real images, real catalog data from the Go backend, and the contact form actually submits to the Go backend and shows a success state.
2. The page is responsive at every breakpoint listed in §8 with no horizontal scroll on any viewport ≥ 360px wide.
3. Lighthouse desktop: Performance ≥ 90, Accessibility ≥ 95, Best Practices ≥ 95, SEO ≥ 95.
4. `prefers-reduced-motion` is respected end-to-end.
5. The page works with JavaScript disabled for **static** sections (hero, sections 2–6, gallery, about, closing) and the contact form gracefully shows a "JavaScript required" message OR we provide a server-action fallback (we'll likely go with server actions, which means no-JS still works for the form).
6. `bun run lint && bun run build` pass with zero errors and zero warnings.
7. Manual QA on a real Chrome, Safari, and Firefox: hero video plays, scroll reveals trigger, catalog carousel swipes, contact form submits, mobile menu opens, footer links navigate.

When all of the above are true, Stage 1 is done. We then move to `04-services-page.md`.
