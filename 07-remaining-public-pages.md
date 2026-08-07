# 07 — Páginas públicas restantes

Fase 7. Migración de las páginas públicas Go que aún no tenían equivalente Next.

## Inventario de rutas (NewRouter)

**Públicas HTML (Go), ahora migradas a Next:**
- `GET /politica-privacidad` → `/politica-privacidad`
- `GET /terminos-servicio` → `/terminos-servicio`
- `GET /politica-cookies` → `/politica-cookies`
- `GET /reservaciones` → `/reservaciones` (informacional, ver bloqueo)
- `GET /experiencia` → `/experiencia`
- `GET /solicitar-cotizacion` → `/solicitar-cotizacion` (informacional, ver bloqueo)

**Públicas HTML, ya migradas antes de esta fase:** `/`, `/servicios`, `/catalogo`, `/catalogo/producto/{id}`.

**Permanecen exclusivamente en Go (no tocadas):**
- `POST /solicitar-cotizacion`, `PUT|DELETE /cotizacion/carrito/items/{id}` — mutación real de cotización, atada a sesión de carrito bloqueada.
- `POST /solicitar-contacto` — formulario de contacto Go standalone (distinto de `POST /api/contact-requests`, que Next ya usa desde Home).
- `GET /productos/{id}`, `GET /catalogo/producto/{id}` (Go) — rollback del detalle, compat QR.
- Todo `/carrito*`, `/wizard/*` público de mutación — bloqueado por PostgreSQL.
- `GET /iniciar-sesion`, `GET /cerrar-sesion` — autenticación admin, fuera de alcance ("páginas públicas").
- Todo `/panel/*`, `/api/products*` admin, `/solicitudes/*` — admin, no tocado.

**Ruta Go rota descubierta (no corregida, fuera de alcance):** `internal/templates/pages/reservations.templ` postea a `hx-post="/api/reservations"` — esa ruta **no está registrada** en ningún archivo de `internal/routes` (confirmado por grep). El formulario Go actual ya no funciona. No se replicó ese flujo roto en Next.

## Fuente de verdad
Todo el copy migrado proviene literalmente de los templates `.templ` correspondientes (`politica-privacidad.templ`, `terminos-servicio.templ`, `politica-cookies.templ`, `salon.templ`, `reservations.templ`, `quote_request.templ`) o de `frontend/lib/copy/contact.ts` ya aprobado. Nada inventado: sin precios nuevos, sin políticas nuevas (los importes de apartado/cancelación de Términos son texto ya publicado en Go, migrado sin alterar).

## Bloqueos documentados (no migración parcial oculta)

**Cotización real** (`POST /solicitar-cotizacion`): depende de `withProtectedCartSession` — lee/escribe el carrito de la sesión firmada. Reimplementarlo en Next exigiría exactamente la integración de carrito bloqueada por PostgreSQL. Next sirve una página **informativa** con copy real del header del formulario Go ("Solicitar Cotización" / mismo mensaje) y CTA a WhatsApp/teléfono ya aprobados — sin formulario, sin almacenamiento de datos en React, sin `cart_id`. Documentado en comentario del archivo.

**Reservaciones**: el formulario Go apunta a un endpoint inexistente (`/api/reservations`, hallazgo confirmado por grep, cero resultados). No se inventó un flujo — página informativa con el mismo hero real y CTA de contacto ya aprobados, sin formulario roto.

**Experiencia**: migrado casi completo (hero, video, sección de arquitectura, galería) — el popup de galería con lightbox y el overlay de video-tour (`popup.js` + GSAP) del Go original no se reprodujeron; se usa `<video controls>` nativo (sin autoplay) y un grid de imágenes con enlaces reales a tamaño completo, dentro del permiso explícito de omitir lightbight/zoom complejo si amplía demasiado el alcance.

## Archivos creados
```
frontend/components/legal/legal-page.tsx
frontend/app/(site)/politica-privacidad/page.tsx
frontend/app/(site)/terminos-servicio/page.tsx
frontend/app/(site)/politica-cookies/page.tsx
frontend/app/(site)/reservaciones/page.tsx
frontend/app/(site)/experiencia/page.tsx
frontend/app/(site)/solicitar-cotizacion/page.tsx
07-remaining-public-pages.md
```

## Sin modificaciones
`frontend/lib/copy/footer.ts` y `frontend/lib/copy/nav.ts` **ya apuntaban** a estas rutas exactas (`/politica-privacidad`, `/terminos-servicio`, `/politica-cookies`, `/experiencia`, `/solicitar-cotizacion`) — cero cambio necesario, los enlaces del footer/nav simplemente empiezan a resolver dentro de Next. `frontend/components/site/site-footer.tsx` sin tocar.

## Carrito
Sin integrar. Sin evidencia de suite PostgreSQL corrida en verde esta ejecución. Cero botón/contador/drawer/POST JSON/Idempotency-Key en React.

## Cutover
Rutas Next nuevas conviven con las mismas rutas Go (`/politica-privacidad`, etc. — Go las sigue sirviendo, no se tocaron ni se eliminaron). Rollback: apagar/no enlazar la ruta Next equivalente, Go sigue funcionando sin cambios. Pendiente: dominio público, topología de proxy inverso, y — solo para `/solicitar-cotizacion` — la integración real de carrito antes de que esta página deje de ser informativa.
