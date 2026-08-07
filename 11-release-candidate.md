# Fase 11 — Release Candidate

> **Actualización (Fase 12 — Final Verification, 2026-08-06):** E2E real de cotización ejecutado (180/180 Playwright, 6 proyectos), doble envío confirmado con evidencia directa (riesgo residual, sin resolver), `CartItem.Available` cubierto con prueba PostgreSQL dedicada, `-p 1` confirmado obligatorio también para `internal/routes`. Ver [`12-final-verification.md`](./12-final-verification.md) para el detalle. El resto de este documento sigue vigente.

Fecha: 2026-08-06. Continuación directa de `10-production-cutover.md` (Fase 10, aprobada). No se desplegó ni modificó producción en ningún punto de esta fase.

## 1. Protección contra escrituras paralelas

Hashes base registrados al inicio de la fase para todos los archivos listados en la instrucción (carrito Next, Server Actions, CSRF, cookie/sesión, rutas Go de carrito/cotización/quotes admin, ambas migraciones citadas, `package.json`, `bun.lock`, `next.config.ts`, docs finales). **Ningún archivo fue modificado por un proceso externo durante la fase** — sin conflictos que reportar.

## 2. Migración histórica — `20250901230135_images_table_updates.sql`

- **Up: sin cambios.** Idéntico a Fase 10.
- **Down: solo el fix de Fase 10** (`DROP TABLE images` → `DROP INDEX images_filetype_idx; ALTER TABLE images DROP COLUMN file_type, DROP COLUMN updated_at;`).
- Versión/nombre del archivo: sin cambios.
- Down revierte únicamente lo agregado por este Up (columnas `file_type`, `updated_at`, índice `images_filetype_idx`) — nunca la tabla `images` completa (creada por `20250703195451_add_images_table.sql`, migración distinta y no tocada).
- Dependientes: `images_products` (FK a `images.id`) confirmado sobreviviente.

**10 pruebas PostgreSQL reales agregadas** — `internal/db/images_migration_postgres_test.go::TestPostgresImagesTableUpdatesMigrationDownAndReup`, un solo test secuencial que cubre los 10 puntos exigidos: (1) up completo, (2) estado de `images` antes del down (fila de prueba insertada + columnas/índice presentes), (3) down de exactamente esta migración (`down-to 20250901203644`, la migración inmediatamente anterior), (4) `images` sigue existiendo, (5) `images_products` sigue existiendo, (6) la fila y el link previos sobreviven, (7) `file_type`/`updated_at`/índice confirmados ausentes tras el down, (8) re-up, (9) fila original + nueva inserción tras el re-up sobreviven con el default correcto (`image/jpeg`), (10) goose termina en la versión correcta (`20250902000000`).

Resultado real: **PASS**.

## 3. Suite PostgreSQL

Contenedor desechable reutilizado de Fase 10 con credenciales nuevas de sesión: `cart-integration-pg`, `127.0.0.1:55432`, `cart_integration_local`, solo localhost, sin datos reales.

```
go test -mod=vendor -p 1 -count=1 -v ./internal/cart/... ./internal/db/...
```

`-p 1` es obligatorio: ambos paquetes destruyen y recrean `public` de la misma base dedicada (`DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); en paralelo colisionan (confirmado en Fase 10, no repetido en Fase 11 porque el hallazgo ya está documentado y el comando ya lo aplica). No se ocultó vía retries.

**Resultado: 72/72 PASS** (71 heredadas de Fase 10 + la nueva prueba de la migración de `images`). **Cero deadlocks** (`grep -i deadlock` sin coincidencias).

`go test -mod=vendor -p 1 -count=1 ./internal/cart/... ./internal/db/... ./internal/routes/...` (incluyendo las nuevas pruebas de cotización) también corrido íntegro al cierre: **todo verde**, sin regresiones.

## 4-5. CSRF

Auditoría de `internal/security/csrf.go`: Origin se lee de `Header.Values("Origin")` (rechaza múltiples valores o listas separadas por coma); si Origin está ausente, cae a `Referer` (nunca al revés — un Origin inválido nunca hace fallback); ambos se normalizan a `scheme+hostname+port` y se comparan contra `CSRF_TRUSTED_ORIGINS`. `X-Forwarded-Host`, `X-Forwarded-Proto`, `Forwarded`, `Host` de la request: **ignorados por completo**, nunca leídos por el guard. `Protect()` rechaza antes de invocar `next` — cero UUID, cero cookie, cero acceso a DB en una request rechazada (confirmado por `TestCSRFGuardRunsBeforeCartSessionAndLoaders`, que cuenta generaciones de UUID = 0 tras un rechazo).

Cadena real Next→Go: Navegador solo habla con Next; el Server Action de Next hace el fetch server-to-server con `Origin` fijado al origen propio de Go (`goOrigin()` en `frontend/lib/api/cart.ts` / `frontend/lib/api/quote.ts`) — legítimo porque el navegador nunca llega a Go directamente (sin CORS, sin superficie de ataque cross-site en ese header). Detrás de un reverse proxy productivo: Go debe confiar en headers `Forwarded`/`X-Forwarded-*` únicamente si el proxy los reescribe de forma confiable — el guard actual no los lee en absoluto, así que esto es indiferente al guard mismo; la topología recomendada (sección 8) es la que importa aquí.

**15/15 escenarios unitarios confirmados** (pre-existentes de fase anterior, re-verificados en vivo): same-origin aceptado, Origin externo, Referer externo, ausencia combinada, contradicción Origin/Referer, Host manipulado, `X-Forwarded-Host` ignorado, `X-Forwarded-Proto` ignorado, `Forwarded` ignorado, GET/HEAD/OPTIONS sin CSRF, POST/PUT/PATCH/DELETE protegidos, múltiples headers rechazados, cero leak de detalles internos, cero cookie/UUID/DB antes de rechazar.

Nuevo: `TestQuoteRequestJSONRejectsInvalidCSRFOrigin` confirma el mismo guard aplicado a `/solicitar-cotizacion`.

## 6. Cookie y sesión

`internal/session/cart.go` — 12 pruebas pre-existentes re-verificadas en vivo: nombre exacto `cart_id`, HttpOnly, `SameSite=Lax`, `Path=/`, `MaxAge`, `Secure` (según `CART_COOKIE_SECURE`), sin `Domain` (host-only), firma HMAC-SHA256 validada, cookie inválida/duplicada reemplazada, primer add crea cookie, replay conserva identidad. `frontend/lib/api/cart.ts` reenvía el valor sin decodificar/re-firmar nunca; `frontend/lib/api/quote.ts` sigue exactamente el mismo patrón para el flujo de cotización.

## 7-11. Cotización pública — implementada (Opción A)

**Hallazgo de auditoría inicial:** `HandleQuoteRequestSubmission` ya cargaba el carrito por cookie firmada (nunca por `cart_id` de cliente) y ya estaba protegido por CSRF + sesión de carrito. Faltaba: contrato JSON, PRG-303, y validación real del contenido del carrito antes de crear la cotización.

**Cambios en `internal/routes/contact_requests.go`:**
- `wantsQuoteRequestJSON` negocia contenido: solo `Content-Type: application/json` + `Accept: application/json` (exactamente lo que hace el Server Action de Next) activa el contrato JSON — cualquier envío real de navegador (con o sin JS) sigue el flujo HTMX/formulario existente, sin cambios de comportamiento.
- `handleQuoteRequestJSON`: body limitado a 8 KB (`http.MaxBytesReader`), `json.Decoder.DisallowUnknownFields()` (rechaza campos desconocidos), validación de nombre/teléfono/email/fecha, `validateQuoteCart` antes de crear la cotización, nunca expone SQL/pgx/tablas/UUID interno/stack.
- `validateQuoteCart`: recarga el carrito desde PostgreSQL por la cookie firmada; rechaza carrito vacío, producto no disponible, cantidad fuera de stock actual.
- **Hallazgo real durante las pruebas contra PostgreSQL:** `cart_items.product_id` tiene `ON DELETE CASCADE` hacia `products` (`sql/migrations/20250726033344_add_carts_table.sql`). Borrar un producto borra también su fila en `cart_items` — **"producto eliminado" es indistinguible, a este nivel, de un carrito vacío/reducido**; no hay tabla de auditoría. Documentado en el código (`validateQuoteCart`) y aquí; **no se inventó una tabla ni una migración nueva** para fingir distinguirlo — se corrigió el diseño de la prueba para reflejar el comportamiento real del esquema.
- Flujo HTMX existente: ahora también corre `validateQuoteCart` antes de guardar (antes no validaba disponibilidad/stock, solo adjuntaba `cart.ID` si no vacío) — bug de negocio cerrado sin romper la respuesta de fragmento HTMX.
- **PRG 303 real:** si la request no trae `HX-Request: true` (envío de formulario real, con o sin JS), éxito redirige `303 See Other` a `/solicitar-cotizacion?enviado=1` en vez de renderizar el fragmento — evita reenvío al refrescar.

**Bug real encontrado y corregido en `internal/db/cart.go`:** `LoadItems` escaneaba `cp.category_name` (nullable, `LEFT JOIN categories`) directamente en un `string` no anulable — mismo patrón de bug que el de `long_description` en Fase 10. Reproducido en vivo contra PostgreSQL real (`cannot scan NULL into *string`) al probar un producto sin categoría. Corregido con `COALESCE(cp.category_name, '')`. Se agregó también `CartItem.Available` (antes ausente del struct, necesario para que Go valide disponibilidad sin una segunda consulta).

**Doble envío:** no existe protección persistente contra un doble POST real (no hay `Idempotency-Key` en este contrato ni restricción única en `quotes`). Añadir una tendría que ser una migración nueva (tabla o columna de deduplicación) — **no se implementó silenciosamente**; se documenta como limitación residual real, no evadible sin cambio de esquema aprobado en otra fase.

**Next — `/solicitar-cotizacion` es ahora un flujo real:**
- `frontend/lib/api/quote.ts` (nuevo, server-only): `submitQuoteRequest`, mismo diseño same-origin/no-CORS que `cart.ts`.
- `frontend/lib/actions/quote-actions.ts` (nuevo, Server Action): `submitQuoteRequestAction`, PRG vía `redirect()`.
- `frontend/app/(site)/solicitar-cotizacion/page.tsx`: Server Component real — resumen del carrito (vía `fetchCartState()`), estado vacío, estado backend-unavailable, formulario progresivo con labels/`required`/`autocomplete` correctos (`name`, `tel`, `email`), confirmación accesible (`role="status"`), enlace de vuelta a `/carrito`. Funciona sin JavaScript (Server Action = submit nativo, mismo patrón ya probado en el carrito). No hay CORS, no se acepta `cart_id`, no se duplica validación de negocio (Go sigue siendo la única fuente de verdad — Next solo reenvía campos de texto).
- **Campo `event_type` omitido del formulario Next:** `db.FindAllEventKinds()` no tiene endpoint público (mismo hallazgo de Fase 10) — el campo sigue existiendo en el contrato Go (opcional) pero Next no ofrece un selector real todavía; documentado, no inventado.

**`/api/quotes` administrativo: sin cambios, sigue protegido por `auth.ValidateAuth`** (`internal/routes/quotes.go`).

## Pruebas de cotización (reales, contra PostgreSQL)

`internal/routes/quote_request_postgres_test.go` — **11 pruebas nuevas, 11/11 PASS en vivo**:

1. Carrito vacío → `cart_empty`.
2. Carrito con producto válido + datos válidos → `200 {"status":"success"}`, cotización real verificada en `quotes`.
3. Producto unavailable → `product_unavailable`.
4. Producto eliminado (cascada real) → `cart_empty` (ver hallazgo arriba).
5. Cantidad inválida (excede stock real) → `invalid_quantity`.
6. Campo requerido ausente (`name` vacío) → `invalid_request`.
7. Email inválido → `invalid_request`.
8. Teléfono inválido (< 10 dígitos) → `invalid_request`.
9. Campo desconocido (`total_price`) → `invalid_request` (`DisallowUnknownFields`).
10. CSRF inválido (Origin ajeno) → `403`.
11. Cookie inválida/manipulada → tratada como carrito nuevo vacío → `cart_empty` (nunca reutiliza el carrito de otra identidad).

**No cubiertas por pruebas de Go en esta fase** (documentado, no fabricado):
- Body demasiado grande / Content-Type incorrecto: mismo mecanismo (`MaxBytesReader`, `mime.ParseMediaType`) ya cubierto por pruebas unitarias del cart API (`internal/routes/cart_api_mutations_test.go`); no se duplicó una prueba idéntica para este endpoint por límite de tiempo de la fase.
- Error de DB inyectado: requeriría fault-injection sobre la conexión PostgreSQL real, fuera de alcance del contenedor desechable de esta fase.
- Doble envío: ver limitación documentada arriba — dos requests reales distintas hoy crean dos cotizaciones.
- `/api/quotes` sin auth: cubierto por `TestQuotesMutationsRequireAuth` (`internal/routes/categories_auth_test.go`), no repetido aquí.
- Solicitud pública sin auth admin: la prueba 2 de arriba ya lo demuestra (ninguna cabecera de autenticación admin usada).
- Sin JavaScript: territorio Playwright, no cubierto en esta fase por límite de tiempo — **residual, ver cierre**.

## E2E de cotización

**No ejecutado en esta fase.** Requiere Go real + PostgreSQL sembrado corriendo simultáneamente a Playwright (mismo patrón que `cart-flow.spec.ts` de Fase 10); dado el tiempo disponible en esta fase se priorizó cerrar la validación real a nivel Go+PostgreSQL (arriba, 11/11 PASS) sobre añadir el navegador. **Bloqueo residual no evadible dentro del tiempo de esta fase** — no fabricado, no simulado.

## 12. Reservaciones

Re-verificado: sin modelo, sin tabla, sin handler, sin `/api/reservations` registrado (`grep` confirma). `frontend/app/(site)/reservaciones/page.tsx` sigue puramente informativa — sin formulario, sin `action` roto, sin promesa de confirmación inmediata, sin copy que implique una función inexistente. Sin cambios necesarios.

## 13. Dominio y topología

**Sin evidencia nueva.** Repetida la búsqueda (deployment files, reverse proxy, `Dockerfile`, `docker-compose`, `Caddyfile`, `nginx`, DNS documentado): ningún archivo de infraestructura de despliegue en el repositorio, igual que en Fase 10. El dominio del QR no se usa como evidencia suficiente (regla explícita). **No se crea `metadataBase` ni `canonical`.**

Topología recomendada (sin cambios respecto a Fase 10, reafirmada):
- Next dueño de páginas públicas, un origen público único.
- Go accesible solo internamente (no expuesto directo a Internet).
- APIs públicas de Go expuestas bajo el mismo origen público vía reverse proxy (mismo host que Next).
- Sin CORS en ningún punto — todo tráfico navegador↔Go pasa por Next server-to-server.
- Cookies (`cart_id`) emitidas para el host público, sin `Domain`.
- `Forwarded`/`X-Forwarded-*` aceptados por el proxy solamente si el proxy los reescribe; Go mismo no los usa para CSRF (ver sección 4-5).

## 14. Configuración de producción (plantilla, sin valores reales)

| Variable | Propósito |
|---|---|
| `GO_API_BASE_URL` | Origen interno donde Next contacta a Go server-to-server. |
| `CART_COOKIE_SECRET` | ≥32 bytes, HMAC de `cart_id`. Rotación: invalida todas las sesiones activas — planear ventana. |
| `CART_COOKIE_SECURE` | `true` en producción (HTTPS real). |
| `CSRF_TRUSTED_ORIGINS` | Debe incluir el origen público exacto que sirve Next (y por tanto el que Go recibe como Origin en las llamadas server-to-server). |
| `DATABASE_URL` | Postgres productivo — nunca el contenedor desechable. |
| URL pública del sitio | Usada para `canonical`/`metadataBase` — **pendiente hasta confirmar dominio real** (sección 13). |
| Trusted proxy | El reverse proxy productivo debe ser el único que setea `X-Forwarded-*`; Go no depende de ellos para CSRF pero sí para logging/IP real si se agrega en el futuro. |
| Health checks | Go: endpoint liviano de DB ping. Next: `/api/_health` ya existe. |
| Timeouts | Next→Go: 5s (ya usado en `cartRequest`/`submitQuoteRequest`). |
| Body limits | Cart API 8 KB, quote-request JSON 8 KB. |
| Logging | Nunca loguear `cart_id`, `Idempotency-Key`, ni el secreto. |
| Migraciones | `goose -dir sql/migrations postgres "$DATABASE_URL" up`, antes de iniciar Go. |
| Orden de arranque | Migraciones → Go → health check Go → Next → health check Next → reverse proxy al frente. |

## 15. QA

**Frontend:**
- `bun install --frozen-lockfile`, `bun run lint`, `bun run test` (28/28), `bun run build`: **todos verdes**, ejecutados en vivo esta fase.
- `bun run test:e2e`: **no ejecutado esta fase** (requiere Go+Postgres+Next simultáneos; Fase 10 ya dejó 144/144 verdes para carrito/páginas públicas — no repetido por no haber cambios ahí; el nuevo flujo de cotización queda sin E2E, ver bloqueo arriba).

**Go:**
- `go build -mod=vendor ./...`: limpio.
- `go vet -mod=vendor ./...`: limpio.
- `go test -mod=vendor -p 1 -count=1 . ./cmd/... ./internal/...`: **todo verde**, 0 fallos.
- Race detector: no ejecutado (requiere CGO + compilador C, no confirmado disponible; no se instaló uno solo para esto, según instrucción explícita).

**Playwright multi-resolución/JS-on-off para cotización/CSRF/cookies/doble-clic/etc.: no ejecutado esta fase** — mismo bloqueo de tiempo que el E2E de cotización.

## 16. Cutover (plan, no ejecutado)

Sin cambios respecto al plan ya documentado en `10-production-cutover.md` salvo agregar el paso de cotización pública (ya no bloqueado):

1. Backup de PostgreSQL productivo.
2. Cargar variables (sección 14).
3. `goose up` contra `DATABASE_URL` productivo.
4. Iniciar Go.
5. Health check Go (DB ping).
6. Iniciar Next.
7. Health check Next (`/api/_health`).
8. Configurar reverse proxy (mismo origen público para Next y las rutas API de Go).
9. Confirmar cookies `cart_id` se emiten para el host público, `Secure=true`.
10. Smoke tests: home, catálogo, detalle de producto.
11. Carrito: agregar, actualizar, eliminar (smoke, no toda la matriz).
12. Cotización: enviar una solicitud real de prueba, confirmar en panel admin.
13. QR: confirmar que el destino sigue siendo válido.
14. Admin: login, panel de solicitudes.
15. Monitoreo: confirmar logs / métricas activas.
16. Criterios de rollback: cualquier health check fallido, error rate elevado en los primeros smoke tests, o CSRF/cookie mal configurados (sesiones rotas).
17. Comando de rollback: apuntar el reverse proxy de vuelta al stack Go+templ anterior (rutas Go nunca se eliminaron — siguen sirviendo el sitio completo si Next se retira).
18. Verificación posterior: repetir smoke tests contra el stack de rollback.

## 17-19. Cierre

**Contenedor y credenciales:** limpiados al final de esta fase (ver comando de cierre ejecutado tras este documento) — `cart-integration-pg` detenido y eliminado, `/tmp/pgpass.txt` eliminado, confirmado que no queda PostgreSQL escuchando en `127.0.0.1:55432`.

**No se declara 100%:** dominio real y cutover productivo siguen sin confirmar (secciones 13, 16). Bloqueos residuales explícitos: E2E de cotización con Playwright no ejecutado, matriz completa de Playwright (5 resoluciones × JS on/off, doble-clic, reduced-motion, etc.) no re-ejecutada esta fase, protección persistente contra doble envío de cotización no implementada (requiere migración futura), `event_type` sin selector real en Next (sin endpoint público de tipos de evento), dominio de producción no confirmado.

**Lo que sí se cerró con evidencia real y verificable esta fase:** migración histórica de `images` (10/10 checks, PostgreSQL real), 72/72 suite PostgreSQL sin regresión, CSRF y cookie/sesión reconfirmados con sus 15+12 pruebas, cotización pública implementada de punta a punta (Go + Next) con validación real de carrito, 2 bugs reales de PostgreSQL corregidos (`category_name` NULL, validación de disponibilidad ausente), 11/11 pruebas nuevas de cotización contra PostgreSQL real, QA de Go y de Bun (lint/test/build) verde, reservaciones re-confirmada informativa, contenedor y secretos limpiados.
