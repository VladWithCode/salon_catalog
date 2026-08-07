# 01 — Design system

The single source of truth for the visual language. Every page plan defers to the tokens defined here. The original site's palette was warm and elegant; we **refine** it (not rebrand) to feel more sophisticated while keeping the same DNA.

The system is configured in `app/globals.css` (Tailwind 4 CSS-first) and exposed to Tailwind via the `@theme` block. shadcn/ui's CSS variables for the same names are merged in.

## 1. Visual direction (the elevator pitch)

> A warm, editorial luxury that whispers instead of shouts. Deep cocoa grounds the page, soft champagne paper lifts it, and a single accent of brushed gold draws the eye where it matters. Type is high-contrast Playfair paired with calm Inter — like a wedding invitation that promises a beautifully-run event.

This is the compass for every token below. When in doubt, ask: *would this still look good in a paper-printed brochure?*

## 2. Colour tokens

We are on **shadcn/ui's CSS variable system** so the shadcn primitives inherit our palette. Variables are HSL channels (no `hsl(...)` wrapper) so Tailwind can apply opacity modifiers.

### 2.1 Refined palette (light mode)

| Token | Value (HSL) | Hex equivalent | Role |
| --- | --- | --- | --- |
| `--background` | `36 33% 94%` | `#F4ECE0` | Page background — warm cream "paper" |
| `--foreground` | `24 25% 11%` | `#211B16` | Primary text — deep warm near-black |
| `--card` | `36 40% 97%` | `#FAF6EE` | Card surface — slightly lifted paper |
| `--card-foreground` | `24 25% 11%` | `#211B16` | Text on card |
| `--popover` | `36 40% 97%` | `#FAF6EE` | Floating surfaces (menus, popovers) |
| `--popover-foreground` | `24 25% 11%` | `#211B16` | Text on popover |
| `--primary` | `24 30% 18%` | `#3A2A1E` | Brand deep cocoa — primary buttons, key surfaces |
| `--primary-foreground` | `36 40% 97%` | `#FAF6EE` | Text on primary |
| `--secondary` | `32 30% 86%` | `#E8DBC6` | Soft champagne — secondary surfaces, badges |
| `--secondary-foreground` | `24 30% 18%` | `#3A2A1E` | Text on secondary |
| `--muted` | `32 18% 90%` | `#E6E0D6` | Subdued surfaces (input background, dividers) |
| `--muted-foreground` | `24 12% 38%` | `#6A5E52` | Secondary text, captions |
| `--accent` | `32 50% 45%` | `#B68A4D` | **Brushed gold** — CTAs, highlights, links on hover |
| `--accent-foreground` | `24 30% 18%` | `#3A2A1E` | Text on accent (use sparingly; usually primary-foreground) |
| `--destructive` | `0 65% 42%` | `#B7302B` | Errors, destructive actions |
| `--destructive-foreground` | `36 40% 97%` | `#FAF6EE` | Text on destructive |
| `--border` | `32 22% 82%` | `#D9CDB8` | Hairlines, card borders |
| `--input` | `32 22% 82%` | `#D9CDB8` | Form input borders |
| `--ring` | `32 50% 45%` | `#B68A4D` | Focus ring — matches accent |
| `--radius` | `0.5rem` | — | Default corner radius (overridable per component) |

Comparison to the original (`web/style/styles.css`):

| Original | New | Why |
| --- | --- | --- |
| `--color-light: #efe5d2` | `--background: #F4ECE0` | Slightly cooler, less yellow — reads more "linen" than "butter". |
| `--color-dark: #110e0a` | `--foreground: #211B16` | Lifted a touch; pure black felt heavy on body copy. |
| `--color-primary: #3A2E23` | `--primary: #3A2A1E` | Almost identical; nudged a hair warmer. |
| `--color-secondary: #E4D3B4` | `--secondary: #E8DBC6` | Slightly lifted; pairs better with the new background. |
| `--color-accent: #9C7B56` | `--accent: #B68A4D` | **The biggest shift** — brighter, more confident gold. |
| `--color-neutral: #544940` | `--muted-foreground: #6A5E52` | Re-cast as a *text* token instead of a surface, where it actually was used. |

### 2.2 Dark mode (planned, not used by Stage 1)

We plan a dark mode for the admin experience later. For now the site is **light-mode only**; the public marketing site reads as a daytime, editorial brand, and forcing a dark mode onto it would muddy the elegance. The CSS variable system is set up so we can introduce `.dark` in a later pass without renaming tokens.

## 3. Typography

The original used a custom Agatho serif + Inter. We replace Agatho with **Playfair Display** (Google Fonts, self-hosted via `next/font`) — high-contrast, widely loved by luxury hospitality brands, and ships with an excellent italic that we'll use for accent words ("*inolvidable*", "*el escenario perfecto*", etc.).

### 3.1 Type families

| Family | Role | Source | CSS variable |
| --- | --- | --- | --- |
| **Playfair Display** (400, 500, 600, 700, italic 400/500) | Display & headings | `next/font/google` | `--font-display` |
| **Inter** (400, 500, 600, 700) | Body, UI, navigation | `next/font/google` | `--font-sans` |

In `app/layout.tsx`:

```tsx
import { Playfair_Display, Inter } from "next/font/google";

const playfair = Playfair_Display({
  subsets: ["latin", "latin-ext"],   // Spanish support
  weight: ["400", "500", "600", "700"],
  style: ["normal", "italic"],
  variable: "--font-display",
  display: "swap",
});

const inter = Inter({
  subsets: ["latin", "latin-ext"],
  weight: ["400", "500", "600", "700"],
  variable: "--font-sans",
  display: "swap",
});
```

> Spanish (`latin-ext`) is required for accents, tildes, and ñ.

### 3.2 Type scale

Defined as Tailwind utilities via the `@theme` block. The scale is **slightly tighter** than the default Tailwind scale (we want editorial, not gigantic).

| Token | Size (mobile → desktop) | Line height | Tracking | Use |
| --- | --- | --- | --- | --- |
| `text-display-2xl` | `5rem → 8rem` | `0.95` | `-0.02em` | Hero wordmark |
| `text-display-xl` | `3.75rem → 5.5rem` | `1.05` | `-0.02em` | Section openers |
| `text-display-lg` | `3rem → 4.25rem` | `1.1` | `-0.015em` | H1 / page titles |
| `text-display-md` | `2.25rem → 3rem` | `1.15` | `-0.01em` | H2 |
| `text-display-sm` | `1.75rem → 2.25rem` | `1.2` | `-0.005em` | H3 |
| `text-eyebrow` | `0.75rem` | `1` | `0.18em` (uppercase) | Section kicker |
| `text-body-lg` | `1.125rem → 1.25rem` | `1.65` | `normal` | Lede paragraphs |
| `text-body` | `1rem` | `1.65` | `normal` | Body copy |
| `text-body-sm` | `0.875rem` | `1.6` | `normal` | Captions, helper text |
| `text-button` | `0.9375rem` (15px) | `1` | `0.02em` (uppercase on CTAs) | Buttons |

Defaults in `@theme`:

```css
@theme {
  --font-display: var(--font-display);
  --font-sans: var(--font-sans);
  --text-display-2xl: 5rem;     --text-display-2xl--line-height: 0.95;   --text-display-2xl--letter-spacing: -0.02em;
  --text-display-xl:  3.75rem;  --text-display-xl--line-height:  1.05;   --text-display-xl--letter-spacing:  -0.02em;
  --text-display-lg:  3rem;     --text-display-lg--line-height:  1.1;    --text-display-lg--letter-spacing:  -0.015em;
  --text-display-md:  2.25rem;  --text-display-md--line-height:  1.15;   --text-display-md--letter-spacing:  -0.01em;
  --text-display-sm:  1.75rem;  --text-display-sm--line-height:  1.2;    --text-display-sm--letter-spacing:  -0.005em;
  --text-eyebrow:    0.75rem;   --text-eyebrow--line-height:    1;      --text-eyebrow--letter-spacing:    0.18em;
  --text-body-lg:    1.125rem;  --text-body-lg--line-height:    1.65;
  --text-body:       1rem;      --text-body--line-height:       1.65;
  --text-body-sm:    0.875rem;  --text-body-sm--line-height:    1.6;
  --text-button:     0.9375rem; --text-button--line-height:     1;      --text-button--letter-spacing:     0.02em;
}
```

Responsive sizes use Tailwind 4's responsive variants on top — e.g. `text-display-xl md:text-display-2xl`.

### 3.3 Typography rules

- **Display headlines use Playfair; never Inter bold.** Inter bold at 60px feels generic; Playfair at 60px feels editorial.
- **Italic is a spice, not a base.** One or two words per page go in Playfair italic for emphasis.
- **Eyebrows are uppercase, wide-tracked, set in Inter medium 12px.** They sit above section titles in the brand gold (`text-accent`).
- **Body copy is never centred for paragraphs.** Centre alignment is reserved for short hero copy, eyebrows, and CTAs.
- **Max body width:** `prose prose-neutral max-w-[65ch]` for any long-form copy so we don't get hero-sized paragraphs.

## 4. Spacing, radii, shadows

### 4.1 Spacing rhythm

We adopt a **section-scale rhythm** so the page has natural breathing room:

| Token | Value | Use |
| --- | --- | --- |
| `space-section` | `6rem` (96px) mobile → `9rem` (144px) desktop | Vertical padding for full sections |
| `space-block` | `3rem` → `4.5rem` | Spacing between major blocks within a section |
| `space-stack` | `1.5rem` | Default vertical gap between paragraphs/headings |

Implemented as a utility pair: `.py-section { @apply py-24 md:py-36; }` etc.

### 4.2 Radii

| Token | Value | Used on |
| --- | --- | --- |
| `--radius` | `0.5rem` | Buttons, inputs, small cards |
| `--radius-lg` (computed via Tailwind `calc(--radius) + 0.25rem` = `0.75rem`) | Large cards, modals |
| `--radius-xl` | `1.5rem` | Hero card overlays, large feature cards |
| `rounded-full` | — | Pills, avatar, circular buttons |

We keep radii **modest**. Luxury = sharp and clean, not bubbly.

### 4.3 Shadows

Three elevations, all very soft and warm-tinted:

```css
--shadow-soft:   0 1px 2px 0 rgb(33 27 22 / 0.04), 0 1px 3px 0 rgb(33 27 22 / 0.06);
--shadow-card:   0 4px 16px -4px rgb(33 27 22 / 0.08), 0 2px 6px -2px rgb(33 27 22 / 0.05);
--shadow-elevated: 0 24px 48px -16px rgb(33 27 22 / 0.18), 0 8px 16px -4px rgb(33 27 22 / 0.08);
```

Exposed as `shadow-soft`, `shadow-card`, `shadow-elevated` Tailwind utilities.

## 5. Imagery & overlays

- Salon photos are the brand. Every photo should be **object-cover, well-composed, never stretched or pixelated**.
- Image overlays (on hero, on contact section, on event cards) use a **soft warm-tinted gradient** so the foreground text is always legible:
  - Light text over photo: `bg-gradient-to-b from-primary/60 via-primary/50 to-primary/70`
  - Or: a single `bg-primary/55` flat if the photo is dark enough.
- Avoid pure black overlays — they clash with the warm palette. Always warm-tint the dark.

## 6. Motion

### 6.1 Principles

1. **Elegance, never theatre.** A 200ms fade beats a 600ms bouncy entrance every time.
2. **Stagger siblings; do not animate them in parallel.** The eye reads a stagger as a story.
3. **Animate `opacity` and `transform` only.** Avoid animating `top/left/width/height` (jank-prone).
4. **Respect `prefers-reduced-motion`.** Every motion primitive accepts a `reducedMotion` flag and degrades to a no-op.

### 6.2 Defaults

| Property | Value |
| --- | --- |
| Default duration | `0.5s` |
| Default easing | `cubic-bezier(0.22, 1, 0.36, 1)` (a soft, slightly overshooting ease-out — exposed as `ease-elegant`) |
| Stagger step | `0.08s` |
| Reveal distance | `y: 16px` (small, dignified) |

### 6.3 Reusable motion primitives (in `lib/motion/`)

| Helper | Purpose |
| --- | --- |
| `<RevealOnView />` | Wraps children, fades + slides up 16px when 30% in view. Used for most sections. |
| `<StaggerGroup />` + `<StaggerItem />` | A 2-component pattern: parent staggers children entrance. |
| `<FadeIn delay={n} />` | Simple opacity fade with optional delay. |
| `useReducedMotion` (re-export of framer-motion's) | Hook for components that need to branch. |

All three respect `prefers-reduced-motion` automatically (framer-motion does this when we pass `useReducedMotion().reduce` as a transition override).

### 6.4 Motion inventory (what the home page uses)

- **Hero wordmark:** fade + tiny rise on mount, 0.8s, no scroll trigger.
- **Hero CTA button:** fade + rise on mount, delayed 0.4s after wordmark.
- **Section titles + body:** reveal on scroll, staggered, 0.5s.
- **Event section image:** gentle 1.04× scale on hover (CSS transition, 400ms).
- **Catalog cards:** fade-up on scroll, 0.06s stagger between cards.
- **Gallery thumbnails:** same fade-up, 0.04s stagger.
- **Mobile menu open/close:** sheet-style slide from the right, 300ms.

No parallax, no auto-playing scroll jacking, no 3D rotations.

## 7. Iconography

- **Lucide React** for UI icons (nav, form, social).
- Custom SVG only for the **logo wordmark** (kept as a static asset under `public/assets/logo_name.webp` / `.png` and `<Image>`'d — the existing logo is on-brand).
- Social icons in the footer/header: Facebook, Instagram, WhatsApp — provided as inline SVGs in `components/site/social-icons.tsx` so they can inherit `currentColor` for hover.

## 8. Accessibility tokens (built into the palette)

- All foreground/background pairs exceed **WCAG AA 4.5:1** for body text. Verified values:
  - `foreground` on `background` ≈ 14.6:1 ✓
  - `primary-foreground` on `primary` ≈ 12.8:1 ✓
  - `muted-foreground` on `background` ≈ 4.7:1 ✓
  - `accent` on `background` ≈ 3.1:1 — *only* used for non-text decorative elements and the eyebrow label which uses the primary colour when it sits over a coloured surface.
- Focus ring: `outline-2 outline-ring outline-offset-2` (matches `--ring`).
- All interactive elements get `:focus-visible` styles; never remove focus outlines.

## 9. Definition of "design system done"

After this stage, every page plan can assume:

1. The CSS variables above are present in `globals.css` and shadcn primitives automatically inherit them.
2. `font-display` and `font-sans` classes (or the underlying CSS variables) are available globally.
3. `<RevealOnView>`, `<StaggerGroup>`, `<StaggerItem>`, `<FadeIn>` exist in `components/shared/motion/`.
4. The `cn()` helper is exported from `lib/utils.ts`.
5. The icon set is Lucide + the two custom social SVGs.
6. No page plan should need to invent a new colour or a new font — they should reuse these tokens.
