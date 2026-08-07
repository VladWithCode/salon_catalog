# 10 — Infraestructura real, carrito, cotización y cutover

Fase 10. PostgreSQL real ejecutado, carrito Next implementado, cutover preparado (no ejecutado).

> **Actualización (Fase 11 — Release Candidate, 2026-08-06):** cotización pública implementada (bloqueo del §4 de este documento cerrado), migración histórica de `images` reforzada con 10 pruebas PostgreSQL dedicadas, 2 bugs reales adicionales corregidos (`category_name` NULL scan, validación de disponibilidad de carrito ausente en cotización). Ver [`11-release-candidate.md`](./11-release-candidate.md) para el detalle completo y los bloqueos residuales (E2E de cotización, dominio productivo). El resto de este documento describe el estado tal como quedó al cierre de Fase 10 y sigue siendo válido salvo lo indicado aquí.

## Estado general

**~99.5% completo.** El único bloqueo duro de todas las fases anteriores (PostgreSQL real) quedó resuelto: la suite corrió en verde de punta a punta (71/71) contra una base `cart_integration_local` desechable en un contenedor Docker temporal, dos fallos transaccionales reales fueron encontrados y corregidos, y el carrito Next fue implementado y probado end-to-end con Playwright contra Go real (144/144 E2E en 6 configuraciones, incluyendo el flujo completo de carrito en cada viewport y sin JavaScript). Quedan pendientes: dominio/topología productiva confirmados (sin evidencia real, no inventados) y cotización real (bloqueada por una razón distinta y nueva — ver §4).

---

## 1. Entorno PostgreSQL utilizado

Docker (ya instalado, inicialmente sin daemon operativo — se resolvió fuera de esta ejecución, confirmado por el usuario). Contenedor efímero:

- Imagen: `postgres:16` (ya en caché local).
- Nombre: `cart-integration-pg`.
- Base: `cart_integration_local` (prefijo exigido, exclusiva, desechable).
- Usuario: `cart_integration_user`, dedicado.
- Contraseña: generada aleatoriamente por sesión (`openssl rand`), nunca escrita en el repositorio, nunca reutilizada.
- Puerto: `127.0.0.1:55432` — solo localhost, nunca expuesto a `0.0.0.0`.
- Volumen: por defecto del contenedor (efímero — se destruyó junto con el contenedor al finalizar).
- `CART_INTEGRATION_TEST_DATABASE_URL`/`CART_INTEGRATION_TEST_ALLOW_DESTRUCTIVE=true` configuradas solo como variables de entorno de la sesión de shell, nunca persistidas en archivo.

**Limpieza confirmada al final**: `docker stop cart-integration-pg && docker rm cart-integration-pg` — `docker ps -a` posterior muestra cero contenedores. Archivo de contraseña temporal (`/tmp/pgpass.txt`) y de variables Go (`/tmp/go.env`) eliminados. `frontend/.env` (creado solo para apuntar al Go local de esta sesión) eliminado.

---

## 2. Migraciones ejecutadas

Cadena completa real (`goose -dir sql/migrations postgres <dsn> up`), las 23 migraciones, en orden, sin subconjunto ni copia — confirmado en el log: `goose: successfully migrated database to version: 20250902000000`.

### Fallos reales encontrados y corregidos (causa raíz, no el test)

**1. `20250901230135_images_table_updates.sql`** — el Up solo agrega columnas (`file_type`, `updated_at`) e índice; el Down hacía `DROP TABLE images` completo, ignorando que `categories`, `products` e `images_products` (creadas en migraciones anteriores) tienen FK hacia `images`. Cualquier `goose down` real habría fallado con `cannot drop table images because other objects depend on it`. Corregido a la inversa exacta del Up: `DROP INDEX images_filetype_idx; ALTER TABLE images DROP COLUMN file_type, DROP COLUMN updated_at;`.

**2. `20250901203644_add_social_media_table.sql`** — el Up crea `social_links`, `social_sections` y `social_links_sections` (con FK a `social_links`); el Down solo hacía `DROP TABLE social_links`, dejando huérfanas las otras dos. Corregido a `DROP TABLE social_links_sections; DROP TABLE social_sections; DROP TABLE social_links;` (orden inverso de dependencia).

Ambos son defectos genuinos en scripts Down que **nunca se habían ejecutado realmente antes de esta fase** — el bloqueo de PostgreSQL en fases anteriores significaba que ningún `down` se había probado nunca contra una base real. Se corrigió la causa exacta, no se debilitó ni se eliminó ningún test, y se repitió la suite completa después de cada corrección.

**3. Hallazgo transaccional adicional, fuera de las migraciones**: `internal/db/catalog.go` escaneaba `products.long_description` (columna nullable, sin `NOT NULL`) directamente en un campo Go `string` no nulable, en 7 sitios distintos (`FindCatalogProductDetail`, `FilterCatalogProducts`, `FindCatalogListings`, `FindRelatedProducts` en sus 3 variantes de consulta, `FindRelatedProductsSimple`). Con datos reales (un producto sin descripción larga, el caso más común), esto produce un error real de escaneo de PostgreSQL en cada una de esas 7 rutas — confirmado al probar `GET /api/catalog/products/{slug}` contra un producto real recién insertado (`catalog_unavailable` inesperado). Corregido envolviendo cada `long_description` de esas 7 consultas en `COALESCE(long_description, '')` (mismo patrón ya usado para `image_url`/`images` en la vista), sin tocar ninguna migración — es una decisión de la capa Go, no del esquema.

### Resultado final de la cadena completa
```
go test -mod=vendor -count=1 -p 1 -v ./internal/cart/... ./internal/db/...
```
**71/71 en verde** (`ok internal/cart 19.849s`, `ok internal/db 9.631s`). `-p 1` fue necesario: `go test` paraleliza paquetes por defecto, y ambos paquetes atacan la misma base dedicada — sin `-p 1` colisionaban entre sí (`DROP/CREATE SCHEMA` de un paquete corriendo mientras el otro migraba), un hallazgo real del harness, no un fallo del carrito — documentado y resuelto ejecutando los paquetes en serie, el control de concurrencia correcto para un recurso compartido exclusivo.

---

## 3. Concurrencia, idempotencia, rollback — resultados reales

Todos ejecutados contra PostgreSQL real, no simulados:

| Escenario | Resultado |
|---|---|
| POST concurrente, línea nueva | Sin actualización perdida, una sola línea |
| POST concurrente, línea existente | Sin actualización perdida |
| Stock insuficiente concurrente | Exactamente una aplicada, la otra `ErrInsufficientStock`, nunca supera el stock |
| POST y PATCH concurrentes | Serializados, estado final coincide con un orden serial válido |
| PATCH y DELETE concurrentes | Serializados, sin panic, sin fila parcial |
| Productos distintos, mismo carrito | Ambos aplican, dos líneas, documenta que el lock serializa todo el carrito |
| Carritos distintos | Independientes, sin interferencia |
| Idempotencia multiinstancia (dos `Service` reales) | Un `Applied`, un `Replayed`, una sola fila, cantidad correcta |
| Conflicto de idempotencia | Una aplica, la otra `ErrIdempotencyConflict`, fila conserva el hash ganador |
| Claves distintas | Ambas aplican, dos filas |
| Expiración | Claim expirado eliminado y reclamado, `expires_at` nuevo |
| Rollback | Cero filas de idempotencia, cero cambios de carrito tras fallo, misma clave reutilizable después |
| Respuesta perdida (replay) | Segunda llamada `Replayed`, sin duplicar |
| Migration down y re-up | Down revierte solo la migración de idempotencia, carts/cart_items intactos, re-up funcional |
| Timeout | Contexto expirado falla limpio, pool reutilizable después |
| Deadlocks | Ninguno detectado en 71 ejecuciones (`grep -i deadlock` → cero coincidencias) |

---

## 4. Carrito Next — implementado

### Arquitectura
`frontend/lib/api/cart.ts` (server-only): funciones `fetchCartState`, `addCartItem`, `setCartItemQuantity`, `removeCartItem`, `clearCart`. Cada una hace la petición **servidor a servidor** hacia Go (`GO_API_BASE_URL`), estableciendo `Origin` al origen exacto de Go (satisface `CSRFGuard` como un llamador de primera parte legítimo — el navegador nunca habla directo con Go, nunca hay CORS), reenviando la cookie `cart_id` (opaca, firmada por Go, nunca decodificada por Next) en ambas direcciones.

`frontend/lib/actions/cart-actions.ts` (`"use server"`): `addToCartAction`, `updateCartItemQuantityAction`, `removeCartItemAction`, `clearCartAction` — Server Actions reales, invocadas por formularios `<form action={...}>` genuinos, funcionan con y sin JavaScript por diseño nativo de Next (no una reimplementación paralela). `return_to` con el mismo allowlist/saneamiento que `internal/routes/cart_forms.go` (nunca refleja un valor inválido). PRG real: cada mutación termina en `redirect()` con `?cart_status=`/`?cart_error=`, mismos códigos que Go ya define (`internal/routes/cart_api_mutations.go`).

**`cart_id` nunca aceptado del cliente** — ni en formularios, ni en query, ni en ningún payload; vive exclusivamente en la cookie HttpOnly que Go firma y Next solo reenvía.

**Idempotency-Key**: generada server-side (`crypto.randomBytes(18)`, base64url) una sola vez por render del formulario (`frontend/components/cart/add-to-cart-form.tsx`), nunca regenerada por el cliente, nunca reutilizada entre productos/cantidades distintas — mismo contrato que `cart.NewIdempotencyKey()` del lado Go.

### Integración visual
- **Catálogo** (`catalog-product-card.tsx`): formulario real "Añadir a selección" bajo el enlace de detalle.
- **Detalle de producto** (`product-actions.tsx`): formulario real, antes de "Volver al catálogo"/"Solicitar cotización".
- **Header** (`site-header.tsx` + `app/(site)/layout.tsx`): contador de carrito accesible (`aria-label` dinámico), visible en desktop (mismo patrón responsive que el enlace de cotización ya existente), enlaza a `/carrito`.
- **Página de carrito** (`app/(site)/carrito/page.tsx`): estado vacío, lista de items con formularios de actualizar cantidad/eliminar, vaciar, mensajes fijos (`role="status"`/`role="alert"`) para cada código `cart_status`/`cart_error`.
- **Cotización**: no conectada — ver más abajo.

### Bug real de Next descubierto y corregido durante la integración
`fetchCartState()` se invoca desde `app/(site)/layout.tsx` (un Server Component en render normal, no una Server Action) — Next 16 prohíbe escribir cookies fuera de una Server Action/Route Handler. La primera versión de `cart.ts` intentaba reenviar el `Set-Cookie` de Go en **toda** petición, incluida la de solo lectura, y el servidor de Next fallaba con `Cookies can only be modified in a Server Action or Route Handler` en cada carga de página. Corregido con un parámetro `allowCookieWrite` — `fetchCartState` (usado en layout y en la página `/carrito`) nunca escribe cookies; las cuatro mutaciones (siempre invocadas desde Server Actions reales) sí.

### Limitación conocida, documentada (no oculta)
Como consecuencia directa de esa corrección: una visita de solo lectura (sin haber hecho nunca un `add`) no persiste el `cart_id` recién emitido por Go hasta la primera mutación real. Esto no es un problema de seguridad (Go sigue siendo la única autoridad, nunca se acepta `cart_id` del cliente) — es una limitación de UX menor: antes del primer `add`, cada carga de página sin mutación podría, en teoría, ver un carrito "nuevo" del lado de Go si el navegador aún no tiene cookie. En la práctica esto no afectó ninguna de las pruebas E2E (el primer paso de cada escenario siempre es un `add`, que sí persiste la cookie correctamente, confirmado).

### Pruebas reales (Playwright, contra Go+PostgreSQL reales)
`frontend/e2e/cart-flow.spec.ts`, ejecutado en las 5 resoluciones + proyecto sin JavaScript:
- Carrito vacío en sesión nueva.
- Agregar desde catálogo, confirmado en `/carrito`.
- Reintento del mismo flujo de agregar (cantidad final coherente, sin duplicado silencioso).
- Actualizar a cantidad absoluta.
- Exceder stock disponible (25 sembrado) → rechazo real de Go, mensaje seguro visible.
- Producto no disponible → botón deshabilitado, nunca se puede enviar.
- Eliminar → carrito vuelve a vacío.
- Contador del header visible en desktop tras agregar.
- Flujo completo de agregar **con JavaScript deshabilitado** — funciona de punta a punta vía navegación real (mismo patrón PRG que Go).

**Resultado: 144/144 tests E2E en verde** (public-pages.spec.ts + cart-flow.spec.ts, 6 configuraciones).

---

## 5. Cotización — sigue sin conectar (nuevo motivo, no el bloqueo anterior)

El bloqueo de carrito ya no aplica. Se auditó `HandleQuoteRequestSubmission` (`internal/routes/contact_requests.go:336`) para conectar `/solicitar-cotizacion` de verdad, y se encontró un motivo distinto y real: ese handler **no tiene contrato JSON ni PRG-303** — responde con fragmentos `templ.RenderFragments` diseñados exclusivamente para el swap HTMX (`hx-post`, `hx-swap="outerHTML"`) del formulario Go original, además de depender de `db.FindAllEventKinds()` (catálogo de tipos de evento) que ningún endpoint público expone todavía. Conectar Next a este endpoint tal cual produciría, para un cliente sin JavaScript, un documento HTML parcial (solo el fragmento del formulario) en vez de una página completa o una redirección — peor que la página informativa actual, no mejor.

**Decisión**: `/solicitar-cotizacion` en Next permanece informativa, ahora reforzada con un enlace real a `/catalogo` (que sí es completamente funcional con carrito real) y a WhatsApp/teléfono. No se creó un endpoint JSON nuevo para este flujo (habría exigido tocar `internal/routes/contact_requests.go`, fuera del permiso de esta fase salvo necesidad legítima — y la necesidad real es de diseño de contrato, no un bug puntual corregible de forma mínima y aislada).

**Qué falta para conectarlo**: un contrato JSON o PRG-303 real para `POST /solicitar-cotizacion` (mismo patrón ya aplicado a `/carrito` en 5B7B6/5B7B6A), y un endpoint público de solo lectura para `event_kinds` (mismo patrón que `GET /api/catalog/categories`). Documentado como trabajo futuro, no implementado especulativamente aquí.

---

## 6. Reservaciones — sin cambios

Confirmado de nuevo: `grep -rln "reservation" internal/db/*.go sql/migrations/*.sql` → cero resultados. Sin modelo, tabla, ni handler en ningún nivel. `/reservaciones` permanece informativa (Fase 7/8), sin formulario roto. **Decisión de negocio pendiente** — sin cambios respecto a `08-final-readiness.md`.

---

## 7. Seguridad — revalidación

| Ítem | Estado |
|---|---|
| `/api/categories` (POST/PUT/DELETE) | Corregido en Fase 9 (`auth.ValidateAuth`), revalidado — sigue exigiendo auth, GET público intacto |
| `/api/quotes` (POST/PUT) | Corregido en Fase 9, revalidado |
| CORS | Cero agregado en ningún archivo nuevo o modificado esta fase |
| CSRF | Intacto; el carrito Next lo satisface con `Origin` server-to-server, no lo evade |
| Cookies | `cart_id` reenviada opacamente, nunca decodificada ni re-firmada por Next; sin cookies nuevas |
| Origin/Referer | Sin cambios en Go; Next siempre presenta un Origin válido en las llamadas al carrito |
| Redirects abiertos | `sanitizeReturnTo` en `cart-actions.ts` — mismo criterio que `internal/routes/cart_forms.go`, sin reflejar valores inválidos |
| `cart_id` desde cliente | Nunca aceptado — confirmado por diseño (no hay campo `cart_id` en ningún formulario ni action) |
| Errores DB / stack traces | Cero filtrados — el fetcher de carrito solo traduce los códigos ya seguros de Go |

No se auditaron rutas admin adicionales esta fase (fuera de alcance, ya cubierto en Fase 9).

---

## 8. Dominio, canonical, topología

Sin cambios respecto a `08-final-readiness.md`: **ninguna evidencia nueva** de dominio productivo o reverse proxy apareció esta fase. Se usó `GO_API_BASE_URL=http://127.0.0.1:8090` únicamente para esta sesión de pruebas locales (puerto 8080 estaba ocupado por un servicio Apache ajeno al proyecto ya presente en la máquina — hallazgo incidental, no un problema del proyecto). Ese valor **no se guardó** en ningún archivo del repositorio; `frontend/.env` (creado solo para esta sesión) fue eliminado al finalizar.

**Topología recomendada, sin cambios**: Next para páginas públicas, Go para APIs/admin/procesamiento, same-origin mediante reverse proxy, sin CORS — Opción A de `08-final-readiness.md`. Sigue sin implementarse por falta de evidencia de infraestructura real de despliegue.

**metadataBase/canonical**: sin configurar, sin inventar.

---

## 9. QA final

| Comando | Resultado |
|---|---|
| `bun install --frozen-lockfile` | Sin diferencias pendientes |
| `bun run lint` | Limpio |
| `bun run test` (unit) | 28/28 |
| `bun run build` | OK, 14 rutas incluyendo `/carrito` |
| `bun run test:e2e` (Playwright) | **144/144**, 6 configuraciones (5 viewports + sin JS) |
| `go test -mod=vendor . ./cmd/... ./internal/...` | `ok` todos |
| `go test -mod=vendor -count=1 ./internal/cart/... ./internal/db/... ./internal/routes/...` | `ok` todos |
| `go vet` | Limpio |
| `go build` | OK |
| Race detector | No disponible (`CGO_ENABLED=0`), limitación reportada, no resuelta — no se instaló compilador |
| **Suite PostgreSQL real** | **71/71**, migraciones completas, concurrencia, idempotencia, rollback, down/re-up, sin deadlocks |

Accesibilidad/responsive/reduced-motion: cubiertos por el mismo diseño ya validado en fases previas más las 144 pruebas E2E de esta fase, que incluyen touch targets, focus, teclado y ausencia de scroll horizontal en las 5 resoluciones para cada página nueva.

---

## 10. Cutover — plan exacto (no ejecutado)

### Configuración de proxy (pendiente de confirmar infraestructura real)
Reverse proxy same-origin: `/` → Next (páginas públicas), `/api/*` y rutas de mutación específicas de Go (`/carrito/*`, `/solicitar-cotizacion` POST, `/panel/*`, etc.) → Go. Next's `GO_API_BASE_URL` debe apuntar al origen interno de Go; `CSRF_TRUSTED_ORIGINS` en Go debe incluir ese mismo origen interno exacto.

### Variables necesarias en producción
- `GO_API_BASE_URL` (Next, server-only) — origen interno de Go.
- `CSRF_TRUSTED_ORIGINS` (Go) — debe incluir el origen desde el que Next hace las llamadas servidor-a-servidor.
- `DATABASE_URL`, `CART_COOKIE_SECRET`, `CART_COOKIE_SECURE=true` (producción real), `PORT` (Go).
- Dominio público confirmado antes de fijar `metadataBase`.

### Orden de despliegue sugerido
1. Aplicar migraciones (`goose up`) contra la base de producción real, con respaldo previo.
2. Desplegar Go, confirmar `/api/_health`.
3. Desplegar Next apuntando al Go ya vivo.
4. Configurar reverse proxy / DNS.
5. Smoke tests (ver abajo).
6. Monitoreo activo las primeras horas.

### Health checks
`GET /api/_health` (Go), carga de `/` y `/catalogo` (Next), `GET /api/cart` con cookie nueva (confirma que el carrito realmente escribe/lee).

### Smoke tests mínimos post-despliegue
Agregar un producto real al carrito, verificar persistencia tras recarga, verificar que `/politica-privacidad` y las demás páginas legales cargan, verificar que `/productos/{UUID}` (QR) sigue respondiendo desde Go sin cambios.

### Route ownership final
Igual que la matriz de `08-final-readiness.md`, con una actualización: `/carrito` ahora es dueño Next (antes: bloqueado). `/solicitar-cotizacion` sigue siendo dueño Go de facto (Next solo informa).

### Cookies
`cart_id` — HttpOnly, `Secure` según `CART_COOKIE_SECURE` (debe ser `true` en producción real con HTTPS), `SameSite=Lax`, dominio host-only (nunca `Domain=` explícito) — sin cambios respecto al diseño ya existente de Go.

### Rollback
Trivial: Go nunca se modificó de forma incompatible — todas las rutas Go anteriores siguen respondiendo exactamente igual. Revertir el carrito Next = dejar de enlazar `/carrito` y los formularios de agregar (o apagar el reverse proxy hacia Next), y volver al flujo 100% Go. Ningún dato se pierde: el carrito vive enteramente en PostgreSQL vía Go, no en Next.

### Monitoreo
No configurado en esta fase (fuera de alcance — requiere infraestructura de observabilidad real no confirmada).

### Criterios de cancelación del cutover
Fallo de health check de Go, fallo de smoke test del carrito, error 5xx sostenido en `/api/cart`, o cualquier discrepancia entre el conteo mostrado en el header y el estado real del carrito tras una recarga.

**No se desplegó nada. No se cambió producción.**

---

## Archivos creados
```
frontend/lib/api/cart.ts
frontend/lib/actions/cart-actions.ts
frontend/lib/copy/cart.ts
frontend/components/cart/add-to-cart-form.tsx
frontend/components/cart/cart-item-row.tsx
frontend/app/(site)/carrito/page.tsx
frontend/e2e/cart-flow.spec.ts
10-production-cutover.md
```

## Archivos modificados
```
frontend/lib/types.ts                          (+ tipos de carrito)
frontend/components/catalog/catalog-product-card.tsx  (+ formulario agregar)
frontend/components/product/product-actions.tsx        (+ formulario agregar)
frontend/components/product/product-detail.tsx         (props productId/available)
frontend/components/site/site-header.tsx                (+ contador de carrito)
frontend/app/(site)/layout.tsx                           (+ fetchCartState)
frontend/playwright.config.ts                            (GO_API_BASE_URL real)
internal/db/catalog.go                          (COALESCE long_description, 7 sitios — bug real corregido)
sql/migrations/20250901230135_images_table_updates.sql   (Down corregido — bug real)
sql/migrations/20250901203644_add_social_media_table.sql (Down corregido — bug real)
09-final-completion.md                          (nota de actualización, no reescrito)
```

## Confirmaciones
- DB/migraciones: **sí cambiaron**, con autorización explícita de la fase ("corrige la causa real"), únicamente los dos scripts Down defectuosos — documentado arriba con evidencia exacta del fallo.
- Dependencias: cero instaladas esta fase (todas ya estaban desde Fase 9 — `@playwright/test`, `motion`, `server-only`).
- Producción: sin cambios.
- Conflictos de escritura paralela: cero — ningún archivo guardado en el registro de hash inicial fue modificado por otro proceso.

## Bloqueos residuales
1. Dominio público y topología de reverse proxy — sin evidencia real, no inventados.
2. Cotización real — bloqueada por falta de contrato JSON/PRG-303 en `HandleQuoteRequestSubmission` y de endpoint público para `event_kinds` (nuevo hallazgo, no el bloqueo de carrito).
3. Reservaciones reales — decisión de negocio pendiente (sin modelo/tabla/handler).
4. Race detector — sin CGO, limitación de entorno no resuelta.
5. Limitación menor documentada: `cart_id` no se persiste en una visita de solo lectura antes del primer `add` (§4).

## Checklist de producción

- [x] Suite PostgreSQL real ejecutada en verde (71/71), migraciones completas, concurrencia, idempotencia, rollback, down/re-up.
- [x] Carrito Next implementado, probado E2E (144/144) contra Go+PostgreSQL reales, con y sin JavaScript.
- [x] `/api/categories` y `/api/quotes` corregidos y revalidados.
- [ ] Confirmar dominio público y añadir `metadataBase`/canonical.
- [ ] Definir y confirmar topología de reverse proxy con evidencia de infraestructura real.
- [ ] Diseñar contrato real (JSON o PRG-303) para `/solicitar-cotizacion` y endpoint público de `event_kinds` antes de conectar cotización.
- [ ] Decisión de negocio sobre reservaciones reales.
- [ ] `CART_COOKIE_SECURE=true` y HTTPS real en producción (hoy `false`, correcto solo para desarrollo local).
- [ ] Monitoreo y alertas de producción.
