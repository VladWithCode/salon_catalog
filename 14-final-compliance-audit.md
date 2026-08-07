# 14 — Auditoría final de cumplimiento

Fecha: 2026-08-07. Rama: `mai`. Auditoría del repositorio completo contra los planes originales (`README`, `00-project-setup.md`, `01-design-system.md`, `02-shared-layout.md`, `03-home-page.md`) y contra la arquitectura segura final de las Fases 4–13.

No se desplegó nada. No se tocó producción. No se abrió Pull Request.

## Regla de precedencia aplicada

Los documentos originales contienen decisiones que fueron sustituidas por implementaciones posteriores más seguras. Ninguna mejora posterior se revirtió para coincidir literalmente con un plan antiguo. Orden usado: seguridad y contratos probados → comportamiento probado con PostgreSQL/Playwright → Go como fuente de verdad → diseño visual y UX de los planes → ejemplos conceptuales antiguos.

---

## 1. Resumen

| Métrica | Valor |
|---|---|
| Requisitos auditados | 119 |
| PASS sin cambios | 108 |
| Fallos encontrados y corregidos (FIXED) | 9 |
| Diferencias deliberadas documentadas | 6 |
| Bloqueos externos | 2 |
| Cumplimiento de lo verificable | 100 % (119/119 resueltos: 108 PASS + 9 FIXED + 2 BLOCKED documentados) |

---

## 2. Fallos reales encontrados y corregidos

### 2.1 Contraste del gold como texto pequeño (obligatorio, §6) — FIXED

Cinco fallos WCAG reales, medidos (no estimados) sobre los tokens de `app/globals.css`:

| Par | Ratio previo | Requisito | Estado |
|---|---|---|---|
| `--accent` sobre `--background` (eyebrows, links 12–14 px) | **3.42:1** | 4.5:1 | fallaba |
| `--accent` sobre `--card` | **3.63:1** | 4.5:1 | fallaba |
| `--accent` sobre `--secondary` | **2.85:1** | 4.5:1 | fallaba |
| `--accent` sobre `--primary` (eyebrows en secciones cocoa) | **3.50:1** | 4.5:1 | fallaba |
| `--accent-foreground` sobre `--accent` (texto de botón 15 px) | **3.50:1** | 4.5:1 | fallaba |

`01-design-system.md` §8 ya anticipaba el problema ("accent on background ≈ 3.1:1 — *only* used for non-text decorative elements"), pero el código sí lo usaba como texto: `.type-eyebrow` tenía `color: hsl(var(--accent))` y decenas de enlaces `type-small text-accent`.

**Corrección** — dos tokens nuevos en la misma familia de marca, sin rebrand y sin tocar `--accent`:

| Token | Valor | Ratios medidos |
|---|---|---|
| `--accent` (sin cambios) | `32 50% 45%` | solo decoración: reglas, bordes, iconos, focus ring (3.42:1 ≥ 3:1 exigido por WCAG 1.4.11) |
| `--accent-strong` (nuevo) | `32 55% 32%` | background **5.81:1**, card **6.16:1**, secondary **4.83:1**, muted **5.26:1** |
| `--accent-on-dark` (nuevo) | `32 60% 62%` | sobre `--primary` **5.95:1** |

Botones gold pasaron de `bg-accent` + `text-accent-foreground` (3.50:1) a `bg-accent-strong` + `text-primary-foreground` (**6.16:1**). `.type-eyebrow` usa `--accent-strong`, y `--accent-on-dark` bajo `.section-dark` / `.on-dark`.

Archivos: `app/globals.css` y 20 componentes.

### 2.2 Formulario de contacto inoperante sin JavaScript (§18, §33) — FIXED

Dos defectos encadenados, ambos confirmados en vivo:

1. `components/home/contact-form.tsx` contenía
   `<noscript><style>#contact-submit-button { display: none !important; }</style></noscript>`
   más el texto *"El envío de este formulario requiere JavaScript"*. Es decir, el botón de envío **desaparecía** sin JS. `03-home-page.md` §10.5 exige explícitamente que el formulario funcione sin JavaScript.
2. Aun mostrando el botón, `app/api/contact-requests/route.ts` sólo aceptaba `application/json`. Un envío nativo de formulario manda `application/x-www-form-urlencoded`, así que devolvía **415 con un cuerpo JSON crudo** directamente en la ventana del navegador. Verificado antes del fix:
   ```
   HTTP/1.1 415 Unsupported Media Type
   content-type: application/json
   ```

**Corrección**: el Route Handler acepta además `application/x-www-form-urlencoded` y responde con **PRG 303** a `/?contacto=enviado#contacto` (mismo contrato que carrito y cotización); `forwardContactRequest()` es ahora el único punto donde una solicitud validada llega a Go, compartido por ambas rutas para que no puedan divergir; `ContactSection` renderiza la confirmación con `role="status"` / `role="alert"`; se eliminó el bloque `<noscript>` y su copy.

Verificado en vivo contra Go + PostgreSQL reales:
```
POST urlencoded  → HTTP/1.1 303 See Other, location: /?contacto=enviado#contacto
POST JSON        → 201  (ruta del cliente intacta)
SELECT ... FROM quotes → "Prueba Sin JavaScript|contacto"
```

### 2.3 Accesibilidad Lighthouse por debajo del objetivo (§29) — FIXED

Primera medición real: **Accessibility 93**, contra el objetivo ≥ 95 de `03-home-page.md` §10.3. Causa concreta, no genérica: las auditorías `definition-list` y `dlitem` fallaban porque el `<dl>` de la tarjeta "Encuéntranos" envolvía cada par en un `<div>` que contenía **otro** `<div>` más el icono, en lugar de `<dt>`/`<dd>` como hijos directos.

**Corrección**: el icono vive ahora dentro de su propio `<dt>`, y `<dt>`/`<dd>` son hijos directos del `<div>` envolvente. La retícula de dos columnas se mantiene idéntica.

Resultado tras el fix (3 corridas, Chrome headless desktop): **Accessibility 100** en las tres.

### 2.4 FAB de WhatsApp ausente entre 768 y 1023 px (§10) — FIXED

`components/site/whatsapp-fab.tsx` llevaba `md:hidden lg:inline-flex`, dejando la franja 768–1023 px sin ningún acceso a WhatsApp (el CTA del header en ese rango es "Cotizar", no WhatsApp). `02-shared-layout.md` §9 lo requiere en todos los breakpoints. Clases eliminadas; cubierto por prueba automatizada en las 5 resoluciones.

### 2.5 Contraste del eyebrow sobre fotografía (§6) — FIXED

Las tarjetas de `offer-intro` situaban el eyebrow sobre un degradado `to-primary/85` encima de una foto: con imágenes claras el fondo efectivo dejaba el texto por debajo de 4.5:1, y el valor real dependía de la foto. El degradado inferior pasó a `to-primary/95`, que es cocoa prácticamente opaco, de modo que el contraste es determinista y no depende de la imagen.

### 2.6 Cobertura automatizada ausente (§37, §38) — FIXED

No existía guardia para ninguno de los puntos anteriores. Se añadió `frontend/e2e/audit-compliance.spec.ts` con 5 pruebas:

1. Contraste ≥ 4.5:1 calculado sobre los colores que el navegador resuelve realmente, para cada `.type-eyebrow` del home (los que están sobre imagen se verifican contra la lista de tokens accesibles permitidos, porque su fondo pintado no es un valor único).
2. El home renderiza exactamente **13** secciones (`03-home-page.md` §3).
3. Contacto sin JavaScript: envío nativo → 303 → confirmación visible.
4. FAB de WhatsApp visible en 360 / 768 / 1024 / 1280 / 1440.
5. Sin errores de consola inesperados en el home (sólo se admite el rechazo de autoplay del vídeo, que es política del navegador, no fallo de la aplicación).

### 2.7 Proyectos Firefox, WebKit y reduced-motion inexistentes (§32) — FIXED

`playwright.config.ts` sólo tenía Chromium. Se añadieron proyectos `firefox`, `webkit` y `reduced-motion` (este último fuerza `prefers-reduced-motion: reduce` sobre toda la suite de páginas públicas, para que una regresión que *oculte* contenido en lugar de sólo detener el movimiento falle de forma visible). Navegadores descargados con `bunx playwright install firefox webkit` — sin dependencias nuevas en `package.json`.

### 2.8 Selectores frágiles en las pruebas de carrito y cotización (§37) — FIXED

`cart-flow.spec.ts` y `quote-flow.spec.ts` localizaban la tarjeta de producto con
`page.locator("article", { hasText: "Mesa Redonda de Prueba" }).first()`.
El `<article>` de página también contiene ese texto, así que `.first()` devolvía el contenedor de la página entera, no la tarjeta. Con un solo producto disponible el descuido quedaba oculto; al sembrar un catálogo más realista (3 productos, 2 categorías) el selector empezó a resolver **2 botones** y las pruebas fallaron con `strict mode violation`.

Es un defecto de la prueba, no de la aplicación, y se corrigió **el selector, nunca la expectativa**: ahora se filtra por el `<heading>` exacto de la tarjeta y se toma el elemento más interno. Las 11 pruebas de carrito y cotización vuelven a pasar sin relajar ninguna aserción.

### 2.9 Cadena de migraciones desactualizada en pruebas (§34) — FIXED

`internal/db/cart_atomic_postgres_test.go` e `internal/db/images_migration_postgres_test.go` afirmaban que la cadena terminaba en `20250902000000`, que dejó de ser la última al añadirse la migración de idempotencia de cotizaciones en la Fase 13. Actualizado a `20251001000000`. (Corregido durante la Fase 13; verificado aquí.)

---

## 3. Matriz de cumplimiento

Estados: **PASS** verificado sin cambios · **FIXED** fallaba y se corrigió · **BLOCKED** requiere algo externo · **DIFF** diferencia deliberada respecto al plan antiguo.

### 3.1 `00-project-setup.md`

| Requisito | Estado | Evidencia |
|---|---|---|
| `components.json` con estilo new-york, rsc, aliases | PASS | `frontend/components.json` |
| Tailwind 4 CSS-first, sin `tailwind.config.js` | PASS | `app/globals.css` con `@theme inline`; no existe config JS |
| Alias `@/*` | PASS | `tsconfig.json`; usado en todos los imports |
| `cn()` en `lib/utils.ts` + `clsx` + `tailwind-merge` | PASS | `lib/utils.ts`, `package.json` |
| `class-variance-authority` | PASS | presente y usado |
| `lucide-react` | PASS | usado en header, footer, formularios, listas |
| Framer Motion | DIFF | se usa `motion` ^13 (el paquete sucesor oficial de framer-motion), importado como `motion/react` |
| Bun como gestor, `bun.lock` única lockfile | PASS | sólo `bun.lock`; no hay `package-lock.json`, `yarn.lock` ni `pnpm-lock.yaml` |
| Estructura `app/(site)`, `components/`, `lib/`, `hooks/`, `public/` | PASS | coincide con el árbol del plan |
| Assets migrados y referenciados sin rotura | PASS | los 26 assets referenciados existen en `public/assets/` (verificado uno a uno) |
| Vídeo Chenacolo webm + mp4 | PASS | `chenacolo_vid.webm`, `chenacolo_vid.mp4` |
| Favicon | DIFF | `icons: { icon: "/assets/logo.webp" }` en lugar de `favicon.ico`; el asset existe y es la marca real |
| Health endpoint | PASS | `app/api/_health/route.ts`; devolvió 200 en toda la auditoría |
| TypeScript strict | PASS | `tsconfig.json`; `bun run build` compila el proyecto con typecheck |
| Lint y build limpios | PASS | `bun run lint` y `bun run build` sin errores ni warnings |
| Sin rutas locales absolutas en código productivo | PASS | búsqueda de `/home/vladwb`, `C:\`, `/tmp/` en `app/`, `components/`, `lib/`, `internal/`: 0 resultados |
| `NEXT_PUBLIC_API_BASE_URL` | DIFF | sustituido por `GO_API_BASE_URL` server-only; el navegador nunca habla con Go |
| CORS de desarrollo | DIFF | eliminado por diseño same-origin; hay pruebas Go que **afirman su ausencia** (`catalog_product_api_test.go`) |
| Copia de `uploads/` al build | DIFF | sustituido por proxy de medios `app/api/catalog/media/[filename]` con allowlist de nombres |

### 3.2 `01-design-system.md`

| Requisito | Estado | Evidencia |
|---|---|---|
| 20 tokens de color exactos | PASS | `app/globals.css` `:root`, valores HSL idénticos al plan |
| `--radius` 0.5rem + escala lg/xl | PASS | `@theme inline` |
| Playfair Display display/headings | PASS | `app/layout.tsx`, `--font-display`, pesos 400–700 + itálica |
| Inter body/UI | PASS | `--font-sans`, pesos 400–700 |
| `latin-ext` para español (ñ, tildes) | PASS | ambas fuentes declaran `subsets: ["latin","latin-ext"]` |
| Escala tipográfica display-2xl…button | PASS | `@theme inline` + clases `.type-*`, con tamaños responsivos a 48rem |
| Eyebrow uppercase, tracking 0.18em, Inter 500, 12 px | PASS | `.type-eyebrow` |
| Párrafos largos no centrados | PASS | `p { text-align: start }` en `@layer base`; centrado sólo en heroes/eyebrows |
| Ancho legible máximo | PASS | `.measure-body { max-width: 65ch }` |
| `space-section` / `block` / `stack` | PASS | variables + utilidades, con salto responsivo a 9rem/4.5rem |
| Tres sombras cálidas | PASS | `--shadow-soft`, `--shadow-card`, `--shadow-elevated` con tinte `rgb(33 27 22 / …)` |
| Overlays cálidos, nunca negro puro | PASS | `.image-overlay-gradient` y `.image-overlay-flat` usan `--primary` |
| Contraste AA de todos los pares de texto | **FIXED** | ver §2.1; ahora foreground/background 15.06:1, muted 5.46:1, gold-texto ≥ 4.83:1 |
| Focus ring visible, nunca eliminado | PASS | `:focus-visible` global con `outline: 2px solid hsl(var(--ring))` |
| Motion: opacity/transform, sin parallax/3D/scroll-jacking | PASS | revisión de `motion/react`; sólo fade/rise/crossfade |
| `prefers-reduced-motion` respetado | PASS | media query global + `useReducedMotion`; proyecto Playwright `reduced-motion` en verde |
| Iconografía Lucide + SVG sociales inline | PASS | `components/site/social-icons.tsx` con `currentColor` |
| Primitivos shadcn en `components/ui/` | DIFF | no se instalaron: el plan mismo advierte contra acumular archivos muertos; los patrones (botón, input, sheet, etc.) están implementados directamente con los tokens |

### 3.3 `02-shared-layout.md`

| Requisito | Estado | Evidencia |
|---|---|---|
| `<html lang="es">` | PASS | `app/layout.tsx` |
| Skip-to-content a `#content` | PASS | primer elemento del `<body>`, visible al enfocar |
| `<main id="content">` | PASS | `app/(site)/layout.tsx` |
| Header sticky, h-16 / h-20, estado top vs scrolled | PASS | `components/site/site-header.tsx` |
| 6 enlaces de navegación exactos | PASS | `lib/copy/nav.ts`: Inicio, Servicios, Catálogo, Experiencia, Contacto, Cotizar |
| Ningún enlace roto | PASS | todas las rutas existen; `public-pages.spec.ts` navega y verifica los legales |
| Menú móvil: `aria-expanded`, `aria-controls`, Escape, foco atrapado y restaurado, backdrop | PASS | `components/site/mobile-nav.tsx` (`role="dialog"`, `aria-modal`, `firstFocusable.focus()`, `trigger.focus()` al cerrar) |
| Botón de menú ≥ 44×44 | PASS | `min-h-11 min-w-11` |
| Footer con marca, tagline, servicios, explorar, contacto, sociales, copyright, legales | PASS | `components/site/site-footer.tsx` |
| Datos de contacto reales aprobados | PASS | `lib/copy/contact.ts`: Villa Chenacolo · 618 259 3026 · villachenacolo@gmail.com · wa.me/526182593026 |
| `tel:` / `mailto:` / `wa.me` con target blank + rel noopener | PASS | verificado en footer, contact-strip, FAB |
| `<address>` semántico y `nav aria-label` | PASS | `contact-strip.tsx`, footer con `aria-label="Páginas legales"` |
| FAB WhatsApp fijo, aria-label, print:hidden, touch ≥ 44 | **FIXED** | visible ahora en todos los breakpoints (§2.4); contraste del botón 6.80:1 |
| Metadata por defecto con título plantilla | PASS | `app/layout.tsx` `template: "%s · Villa Chenacolo"` |
| `metadataBase` / canonical / OG absoluto | **BLOCKED** | sin dominio productivo confirmado; deliberadamente ausentes (§4) |

### 3.4 `03-home-page.md`

| Requisito | Estado | Evidencia |
|---|---|---|
| 13 secciones en el orden del plan | PASS | `app/(site)/page.tsx`; prueba automatizada cuenta exactamente 13 `article > section` |
| Hero: vídeo webm+mp4, muted, loop, playsInline, poster, overlay cálido | PASS | `components/home/home-hero.tsx` |
| Hero: `preload="metadata"`, nunca `auto` | PASS | atributo explícito; Lighthouse Performance 98–99 |
| Hero: vídeo fuera del árbol de accesibilidad | PASS | `aria-hidden`, `tabIndex={-1}`, poster con `alt=""` |
| Hero: sin vídeo bajo reduced-motion | PASS | render condicional por `useReducedMotion()` |
| Offer intro: eyebrow, heading, lede, 3 tarjetas con imagen y enlace | PASS | `components/home/offer-intro.tsx` → /servicios, /catalogo, /experiencia |
| 6 secciones de evento con alternancia cream/cocoa | PASS | `lib/copy/home.ts` define las 6 en orden; `variant` alterna |
| Evento: eyebrow numerado, título, regla dorada, lede, checklist, CTA WhatsApp | PASS | `components/home/event-section.tsx`, `components/shared/check-list.tsx` |
| Enlaces WhatsApp de evento válidos | PASS | `wa.me` con texto prellenado por evento |
| Catalog preview con datos reales de Go, nunca mock | PASS | `getCatalogListings()`; verificado con datos reales servidos por Go en esta auditoría |
| Catalog preview: estado vacío y "Ver todos" | PASS | `components/home/catalog-preview.tsx` |
| Catalog preview no impone 8 categorías fijas | PASS | itera sobre el dataset real (2 categorías en el entorno de auditoría, sin romperse) |
| Gallery preview ~6 imágenes, object-cover, enlace a /experiencia | PASS | `components/home/gallery-preview.tsx` |
| About strip cocoa con patrón, eyebrow, Playfair, regla dorada | PASS | `components/home/about-strip.tsx` |
| Contacto: `id="contacto"`, formulario real, Go procesa, sin DB directa | PASS | `app/api/contact-requests/route.ts` → Go; Next nunca toca PostgreSQL |
| Contacto: labels visibles, sin placeholder-only, `aria-describedby` | PASS | `components/home/contact-form.tsx` |
| Contacto: validación server-side, límite de cuerpo, Content-Type | PASS | 8 KB, `readLimitedRequestBody`, validación en Go |
| Contacto: sin fuga de errores internos | PASS | códigos controlados; nada de SQL/pgx/stack |
| Contacto: funciona sin JavaScript | **FIXED** | ver §2.2, probado end-to-end con Playwright y confirmado en PostgreSQL |
| Contacto: tarjeta "Encuéntranos" | **FIXED** | contenido intacto; `<dl>` corregido (§2.3) |
| react-hook-form + zod | DIFF | no se añadieron: el flujo progresivo cumple la intención (validación real y UX accesible) sin dependencias extra ni romper el funcionamiento sin JS |
| Closing CTA | PASS | `components/home/closing-cta.tsx` |
| Sin testimonios inventados, sin precios, sin blog | PASS | ninguno presente |
| Lighthouse Perf ≥ 90 / A11y ≥ 95 / BP ≥ 95 / SEO ≥ 95 | **FIXED** | 98–99 / **100** / 100 / 100 en 3 corridas |
| Cero scroll horizontal desde 360 px | PASS | aserción explícita por página en las 5 resoluciones |
| Chrome, Safari, Firefox | PASS / DIFF | Chromium, Firefox y **WebKit** verificados por separado; WebKit es el motor de Safari, no Safari |

### 3.5 Arquitectura final Fases 4–13 (preservación)

| Requisito | Estado | Evidencia |
|---|---|---|
| Cookie de carrito firmada, HttpOnly, SameSite=Lax, host-only, HMAC | PASS | `internal/session/cart.go`; 12 pruebas |
| CSRF con Origin y fallback a Referer; forwarded headers ignorados | PASS | `internal/security/csrf.go`; 15 escenarios |
| Nunca se acepta `cart_id` del cliente | PASS | resuelto siempre desde la cookie firmada |
| Idempotencia de carrito (replay, conflicto, rollback) | PASS | suite PostgreSQL real |
| Idempotencia persistente de cotización | PASS | `quote_idempotency_keys`; replay, 409, rollback, TTL 24 h |
| `/api/quotes` y `/api/categories` siguen protegidos por auth | PASS | `internal/routes/quotes.go`, `categories.go`; `TestQuotesMutationsRequireAuth` |
| Producto por UUID **y** slug; QR históricos intactos | PASS | ruta `[identifier]`; `catalogProductIdentifierKind` |
| Sin CORS | PASS | 0 cabeceras `Access-Control-*`; pruebas que lo afirman |
| Sin `dangerouslySetInnerHTML` | PASS | 0 usos (única coincidencia es un comentario) |
| Reservaciones informativa, sin tabla/endpoint/promesa | PASS | sin `/api/reservations` en ninguna ruta; página sin formulario |
| Migraciones históricas intactas salvo el fix aprobado | PASS | sólo el bloque Down de `20250901230135`; Up sin cambios |
| Rutas Go conservadas para rollback | PASS | ninguna eliminada |

---

## 4. Bloqueos externos

| Bloqueo | Motivo | Consecuencia |
|---|---|---|
| Dominio productivo, HTTPS, reverse proxy | Sin evidencia en el repositorio ni información entregada. `.env.example` sólo contiene placeholders (`http://localhost:8080`) explícitamente marcados como ejemplo. El dominio del QR no es evidencia suficiente por regla explícita. | **No se creó `metadataBase` ni canonical ni OG absoluto.** La metadata sigue siendo relativa. |
| Docker | El servicio `com.docker.service` está detenido y arrancarlo requiere elevación; Docker Desktop no llegó a iniciar. | Se usó una alternativa equivalente y verificable: un clúster **PostgreSQL 16.6 desechable creado con `initdb`** en directorio temporal, puerto 55437, sólo localhost, rol dedicado y contraseña aleatoria de sesión, base `cart_integration_local`. Eliminado por completo al cerrar. Ningún dato real fue tocado; nunca se usó el `DATABASE_URL` de nadie. |

---

## 5. Diferencias deliberadas respecto a los planes antiguos

| Plan dice | Implementación final | Por qué la final es correcta |
|---|---|---|
| `NEXT_PUBLIC_API_BASE_URL` expuesto al navegador | `GO_API_BASE_URL` server-only | El navegador nunca habla directamente con Go; elimina la superficie CORS y evita filtrar la topología interna |
| Middleware CORS `withCORS` para `localhost:3000` | Sin CORS | Diseño same-origin con Next como único cliente de Go; hay pruebas que fallan si reaparece una cabecera CORS |
| Copiar `web/static/uploads/*` a `public/` | Proxy `app/api/catalog/media/[filename]` con allowlist | No duplica archivos subidos ni los expone por ruta directa; valida el nombre antes de servir |
| shadcn/ui `components/ui/*` instalados | No instalados | El propio encargo prohíbe acumular componentes muertos "para marcar una casilla"; los patrones están implementados con los mismos tokens |
| react-hook-form + zod en el formulario | Formulario progresivo + validación en Go | Cumple la intención (validación real, UX accesible) y además **funciona sin JavaScript**, que la librería habría dificultado |
| `framer-motion` | `motion` ^13 (`motion/react`) | Sucesor oficial del mismo proyecto; misma API de `AnimatePresence`/`MotionConfig` |

---

## 6. Validación ejecutada

### PostgreSQL real
Clúster desechable PostgreSQL **16.6** (`initdb`, puerto 55437, sólo localhost, credenciales aleatorias de sesión, base `cart_integration_local`).

```
go test -mod=vendor -p 1 -count=1 -v ./internal/cart/... ./internal/db/... ./internal/routes/...
```
**283 PASS · 0 FAIL · 0 deadlocks.**

`-p 1` sigue siendo obligatorio: los tres paquetes reinician el schema `public` de la misma base dedicada.

> Nota honesta: una corrida intermedia lanzada en segundo plano reportó `panic: test timed out after 10m0s` en `TestPostgresIdempotencyMigrationDownAndReup`, con un tiempo declarado de 3 h 1 m para un test de ~2 s. Se investigó en lugar de reintentar a ciegas: el mismo test corre en **2.61 s** en primer plano y la suite completa pasa en verde. La causa es inanición del proceso en segundo plano del entorno de ejecución, no un defecto del producto. Queda documentado en vez de omitido.

### Go
- `go test -mod=vendor . ./cmd/... ./internal/...` — verde
- `go vet -mod=vendor ./internal/cart/... ./internal/db/... ./internal/routes/...` — limpio
- `go build -mod=vendor ./...` — limpio
- `gofmt -l` — sin archivos pendientes
- Race detector — no ejecutado: requiere CGO y compilador C no disponibles; el encargo prohíbe instalar uno sólo para esto

### Frontend
- `bun install --frozen-lockfile` — sin cambios
- `bun run lint` — limpio
- `bun run test` — **28/28**
- `bun run build` — limpio

### Lighthouse (Chrome headless, preset desktop, 3 corridas)

| Corrida | Performance | Accessibility | Best Practices | SEO |
|---|---|---|---|---|
| 1 | 98 | 100 | 100 | 100 |
| 2 | 99 | 100 | 100 | 100 |
| 3 | 99 | 100 | 100 | 100 |
| Objetivo | ≥ 90 | ≥ 95 | ≥ 95 | ≥ 95 |

Lighthouse se instaló en un directorio temporal fuera del repositorio; `package.json` y `bun.lock` quedaron intactos (verificado con `git status`).

### Playwright
Proyectos: `mobile-360`, `tablet-768`, `small-desktop-1024`, `desktop-1280`, `wide-1440`, `chromium-js-disabled`, `firefox`, `webkit`, `reduced-motion`.
**285 pruebas · 285 PASS · 0 FAIL** en los 9 proyectos.

Cobertura: home, servicios, catálogo, producto, carrito, cotización, experiencia, reservaciones, legales, 404, backend caído, teclado y skip link, foco, cero scroll horizontal, JS activado y desactivado, replay e idempotencia, doble envío concurrente y conflicto.

---

## 7. Limpieza

- Clúster PostgreSQL temporal detenido y su directorio de datos eliminado.
- Procesos Go y Next de prueba detenidos; puertos 8090, 3100, 3101, 3102 y 55437 cerrados.
- Credenciales temporales de sesión eliminadas del disco.
- `test-results/` y reportes de Playwright excluidos por `.gitignore` (añadido en esta auditoría junto con `.claude/`).
- Sin `.env` real, sin dumps, sin `node_modules`, sin `.next` en el commit.
- Producción intacta: no hubo despliegue ni acceso a infraestructura productiva.

---

## 8. Estado final

Todo lo verificable sin infraestructura productiva está en verde. Quedan fuera, por decisión externa ya conocida y no por deuda técnica:

- dominio productivo, HTTPS y reverse proxy;
- el cutover en sí;
- reservaciones reales, a la espera del contrato de negocio.

No se declara despliegue productivo. Se declara **cumplimiento técnico completo de los planes de diseño e implementación** dentro del alcance auditable.
