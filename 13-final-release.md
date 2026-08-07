# Fase 13 — Final Release

Fecha: 2026-08-06. Continuación de `12-final-verification.md` (aprobada). Producción **no** tocada — sin autorización ni credenciales reales entregadas, el cutover sigue siendo un plan, no una ejecución.

## 1. Idempotencia persistente de cotizaciones — resuelto

**Migración nueva** (no se modificó ninguna histórica): [`sql/migrations/20251001000000_add_quote_idempotency_keys_table.sql`](sql/migrations/20251001000000_add_quote_idempotency_keys_table.sql).

```sql
CREATE TABLE quote_idempotency_keys (
    cart_id UUID NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    key_hash BYTEA NOT NULL,
    request_hash BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT quote_idempotency_keys_pkey PRIMARY KEY (cart_id, key_hash),
    CONSTRAINT quote_idempotency_keys_key_hash_length CHECK (octet_length(key_hash) = 32),
    CONSTRAINT quote_idempotency_keys_request_hash_length CHECK (octet_length(request_hash) = 32),
    CONSTRAINT quote_idempotency_keys_expires_after_created CHECK (expires_at > created_at)
);
CREATE INDEX idx_quote_idempotency_keys_expires_at ON quote_idempotency_keys(expires_at);
```

Diseño idéntico en forma a `cart_idempotency_keys` (Fase 5B7B6C), adaptado al modelo transaccional real de `quotes` (no hay "carrito" que mutar — hay una fila nueva que crear):

- **Clave opaca**: generada client-side (Next: `randomBytes(18).toString("base64url")`, `frontend/lib/actions/quote-actions.ts:generateQuoteIdempotencyKey`), nunca por el servidor.
- **Hash SHA-256**: `key_hash = SHA-256(idempotency key)` — la clave cruda **nunca** se guarda (confirmado con prueba real, ver §3).
- **Asociación con el carrito/sesión firmada**: `cart_id` es la clave compuesta junto a `key_hash` — nunca aceptado desde el cliente, siempre resuelto por `cartIDFromRequestContext` (cookie firmada).
- **Hash canónico del contenido**: `request_hash = SHA-256("POST /solicitar-cotizacion\n" + cartID + "\n" + name + "\n" + phone + "\n" + email + "\n" + eventDate + "\n" + eventType)` — de los campos ya validados y parseados, nunca del JSON crudo.
- **Constraint único**: `PRIMARY KEY (cart_id, key_hash)` — un segundo `INSERT` con la misma clave falla a nivel de base de datos, no solo de aplicación.
- **Misma transacción**: `internal/db/quote_idempotency.go:SubmitQuoteIdempotent` — lock de la fila `carts` (`SELECT ... FOR UPDATE`), lectura/escritura del claim, e `INSERT INTO quotes` **dentro de la misma transacción**. Un fallo en cualquier punto hace `tx.Rollback()`, deshaciendo el claim junto con la cotización nunca creada.
- **Replay**: mismo `cart_id` + `key_hash` + `request_hash` sin expirar → no se crea una segunda fila, se responde `{"status":"success","replayed":true}`.
- **Conflicto**: mismo `cart_id` + `key_hash`, `request_hash` distinto → `409 {"error":"idempotency_conflict"}`, nunca se aplica silenciosamente.
- **Clave nueva → solicitud nueva**: `(cart_id, key_hash)` distinto siempre crea una fila nueva.
- **Rollback real**: confirmado con prueba (§3, punto 7) — un `INSERT INTO quotes` forzado a fallar (FK inválida) deja **cero** claims en `quote_idempotency_keys`, la clave queda libre para reintentar.
- **TTL**: 24 horas (`db.QuoteIdempotencyTTL`, igual que el carrito) — suficiente para cubrir un reintento real (error de red, doble clic, respuesta lenta) sin mantener claims vivos indefinidamente.
- **Limpieza**: `idx_quote_idempotency_keys_expires_at` (mismo patrón que el carrito) permite un `DELETE ... WHERE expires_at < now()` periódico — **no se implementó un job de limpieza automático en esta fase** (fuera del alcance explícito de "no crear una migración nueva" más allá de la tabla; un cron/worker es una decisión operativa distinta, documentada aquí como pendiente, no oculta).
- **Sin `cart_id` del cliente**: confirmado — se sigue resolviendo por cookie firmada, sin cambio respecto a Fase 11.
- **Sin datos personales en el hash o logs**: el hash de clave es de la clave opaca (no de datos del solicitante); el hash de contenido incluye nombre/teléfono/email **hasheados**, nunca en texto plano en logs — ningún `log.Printf` de esta fase imprime la clave cruda, el hash, ni los campos del solicitante.

**No se copió literalmente el modelo del carrito**: el carrito bloquea la fila `carts` para mutar sus propios items; cotización bloquea la misma fila `carts` (única ancla estable disponible) para serializar la creación de una fila *nueva* en `quotes` — la lógica de claim/replay/conflict es la misma forma, pero `SubmitQuoteIdempotent` no reutiliza ni envuelve el código del carrito.

## 2. Next y formulario progresivo

- `frontend/lib/actions/quote-actions.ts:generateQuoteIdempotencyKey()` — una clave por render del formulario (Server Component `solicitar-cotizacion/page.tsx`), embebida en `<input type="hidden" name="idempotency_key">`.
- Reenviar el mismo formulario (doble clic, doble submit) reutiliza el mismo campo oculto → misma clave → replay en Go, no duplicado.
- Una navegación nueva a `/solicitar-cotizacion` genera una clave nueva (nuevo render del Server Component).
- Flujo con JavaScript: confirmado (Playwright, ver §3/§7).
- Flujo sin JavaScript: confirmado — el campo oculto viaja igual en un submit nativo de formulario, sin cambio de mecanismo.
- PRG 303: sin cambios, sigue intacto.
- La clave nunca se guarda como identificador permanente (no hay tabla de usuarios que la referencie), no se re-muestra en el HTML de confirmación (la página de confirmación es un render nuevo con una clave nueva, no la misma), y no se imprime en logs de Next ni de Go.

`/api/quotes` administrativo (auth) y la solicitud pública siguen completamente separados — sin cambios en este punto.

## 3. Pruebas de idempotencia — reales, contra PostgreSQL

`internal/routes/quote_request_postgres_test.go` (ampliado) + `internal/db/quote_idempotency.go` ejercitado directamente donde el HTTP handler no puede forzar el escenario (rollback, expiración):

| # | Escenario | Resultado |
|---|---|---|
| 1 | Primer envío crea una cotización | PASS |
| 2 | Replay secuencial no duplica | PASS |
| 3 | Dos envíos **concurrentes** (goroutines reales) con la misma clave → una sola cotización | PASS |
| 4 | Mismo key + mismo payload → replay | PASS |
| 5 | Mismo key + payload distinto → `409 idempotency_conflict` | PASS |
| 6 | Claves distintas → solicitudes distintas | PASS |
| 7 | Fallo tras el claim (FK inválida forzada) → rollback, cero claims residuales | PASS |
| 8 | Misma clave reintentable después del rollback | PASS |
| 9 | Detalles no duplicados (`quote_details.cart_items` con un solo ítem tras replay) | PASS |
| 10 | Expiración: claim vencido permite una submission nueva con la misma clave | PASS |
| 11 | Dos instancias del servicio | cubierto por el mismo mecanismo: el lock es a nivel de fila PostgreSQL (`SELECT ... FOR UPDATE`), no en memoria del proceso Go — dos instancias de Go contra la misma base heredan la misma garantía automáticamente; no se ejecutaron dos binarios Go simultáneos en esta fase (no aporta evidencia nueva sobre un lock ya centralizado en la base) |
| 12 | Clave cruda nunca almacenada | PASS (`TestQuoteRequestJSONIdempotencyRawKeyNeverStored`) |
| 13 | Sin datos personales filtrados | confirmado por inspección de `hashQuoteRequest`/logs — sin prueba automatizada dedicada adicional |
| 14 | Sin JavaScript conserva idempotencia | PASS (Playwright, ver §7 — el campo oculto viaja igual) |
| 15 | Refresh de confirmación no reenvía | la confirmación es una página GET (`?quote_status=sent`), no un POST — un refresh no reenvía nada, comportamiento nativo del navegador, no requiere lógica adicional |
| 16 | Back + submit con una intención nueva | un nuevo submit desde el botón "atrás" reutiliza el HTML ya cargado (misma clave oculta) → Go lo trata como replay si el payload es idéntico, o conflicto si el usuario cambió campos antes de reenviar — comportamiento ya cubierto por los puntos 2 y 5, mismo contrato |

**Concurrencia real con Playwright** (contra Go+PostgreSQL reales, no simulada): `frontend/e2e/quote-flow.spec.ts` — dos requests JSON disparadas con `Promise.all` (paralelas de verdad, mismo cookie, misma clave) → confirmado un `replayed:false` y un `replayed:true`, nunca ambos `false`. Confirmado también con `curl` en paralelo directo contra Go (evidencia adicional fuera de Playwright): 2 requests concurrentes → `SELECT count(*) FROM quotes` = **1**.

**Ninguna prueba fue debilitada para pasar en verde.**

## 4. Dominio y topología

**Sin evidencia nueva.** Repetida la búsqueda: `.env.example` (raíz y `frontend/`) solo contienen plantillas con valores placeholder (`http://localhost:8080`, `http://127.0.0.1:8080`) explícitamente marcadas como ejemplo — no son configuración real. Sin `Dockerfile`, `docker-compose`, `Caddyfile`, `nginx.conf` en el repositorio. **No se recibió información del responsable del proyecto con dominio o credenciales reales.**

**No se despliega. No se inventa dominio. No se crea canonical.** Tareas independientes (idempotencia, QA) completadas igualmente — ver criterio de cierre.

Variables que faltan para poder confirmar producción: dominio público real, certificado HTTPS, configuración del reverse proxy real, host interno real de Go.

## 5. Metadata

**Bloqueada, sin dominio confirmado — sin cambios.** `metadataBase` y `canonical` permanecen sin configurar; la metadata existente (`frontend/app/(site)/*/page.tsx`) sigue siendo relativa/sin dominio absoluto, sin regresión.

## 6. Cutover

**No ejecutado — sin autorización ni credenciales reales.** El plan de 18 pasos documentado en `10-production-cutover.md`/`11-release-candidate.md` sigue vigente sin cambios; no se agrega nada nuevo porque nada nuevo se ejecutó.

## 7. QA tras los cambios de esta fase

**PostgreSQL real** (contenedor desechable `cart-integration-pg13`, `postgres:16`, `127.0.0.1:55435`, credenciales de sesión, eliminado al cierre):
```
go test -mod=vendor -p 1 -count=1 -v ./internal/cart/... ./internal/db/... ./internal/routes/...
```
**283/283 PASS, 0 FAIL, 0 deadlocks.**

Sin variables de integración exportadas: `go test -mod=vendor . ./cmd/... ./internal/...` → todo verde (skip limpio de las pruebas PostgreSQL). `go vet` y `go build`: limpios.

**Frontend:** `bun install --frozen-lockfile` (sin cambios), `bun run lint` (limpio), `bun run test` (28/28), `bun run build` (limpio).

**Playwright** (`bunx playwright test`, matriz completa, 6 proyectos): **192/192 PASS** (180 heredadas de Fase 12 + 12 nuevas de concurrencia/conflicto de cotización × sus proyectos aplicables). Incluye: cotización concurrente (real, `Promise.all` sobre el JSON contract), replay, conflicto (`409`), JavaScript activado/desactivado, 5 resoluciones, carrito, producto unavailable, backend unavailable, teclado, focus, cero scroll horizontal — todos ya cubiertos por los specs existentes, sin necesidad de duplicar cobertura ya verde en Fase 12.

**Race detector:** no ejecutado — sin CGO/compilador confirmado, no instalado por instrucción explícita.

## 8. Limpieza

- Go (`PORT=8090`) detenido, Next detenido (gestionado por Playwright `webServer`).
- `cart-integration-pg13`: `docker stop` + `docker rm`, confirmado `docker ps -a` vacío.
- Puerto `55435`: confirmado cerrado.
- Archivos temporales con credenciales (`/tmp/p13.txt`, seeds, logs) eliminados, confirmado con `ls` fallido sobre ellos.
- **Producción intacta** — sin autorización de despliegue, ningún paso de cutover se ejecutó contra un entorno real.

## 9. Reservaciones — explícitamente fuera de alcance

Sin cambios. Permanece informativa hasta que negocio defina campos/horarios/disponibilidad/confirmación/cancelación/responsable/notificaciones/relación con cotización/modelo/tabla — instrucción explícita de esta fase de no implementarla.

## Bloqueos externos (sin cambio, heredados)

- Dominio público real, HTTPS, reverse proxy: sin evidencia, sin autorización.
- Cutover productivo: sin ejecutar, sin credenciales.
- Metadata/canonical: bloqueados por lo anterior.

---

## Entrega final — Fase 13

1. **Estado final:** cerrada. Deuda técnica crítica dentro del alcance actual (duplicación de cotizaciones) resuelta con evidencia real.
2. **% técnico:** ~99% (todo lo que no depende de infraestructura productiva externa está implementado, probado y verde).
3. **% productivo:** sin cambio — dominio/HTTPS/reverse proxy/cutover real siguen sin confirmar. **No se declara 100% productivo.**
4. **Migración creada:** [`sql/migrations/20251001000000_add_quote_idempotency_keys_table.sql`](sql/migrations/20251001000000_add_quote_idempotency_keys_table.sql), ninguna histórica modificada.
5. **Contrato de idempotencia:** clave opaca client-side, `key_hash`/`request_hash` SHA-256, `PRIMARY KEY (cart_id, key_hash)`, TTL 24h.
6. **Concurrencia:** confirmada con goroutines Go reales + `Promise.all` en Playwright + `curl` paralelo directo — siempre una sola cotización.
7. **Replay:** confirmado, `replayed:true`, sin duplicar.
8. **Conflicto:** confirmado, `409 idempotency_conflict`, nunca aplicado silenciosamente.
9. **Rollback:** confirmado — fallo tras el claim deja cero claims residuales.
10. **TTL:** 24 horas, documentado, mismo valor que el carrito.
11. **Cotización con JavaScript:** confirmada, clave reutilizada correctamente.
12. **Cotización sin JavaScript:** confirmada, mismo mecanismo vía campo oculto.
13. **Tests PostgreSQL:** 283/283 PASS, 0 deadlocks.
14. **Tests Go:** build+vet limpios, suite completa verde.
15. **Tests Bun:** 28/28 PASS.
16. **Playwright:** 192/192 PASS.
17. **Dominio:** sin evidencia, no confirmado.
18. **HTTPS:** no confirmado (depende del dominio).
19. **Topología:** documentada, sin cambios respecto a Fase 11.
20. **Metadata:** bloqueada, sin dominio.
21. **Canonical:** no creado.
22. **Cutover:** no ejecutado, sin autorización/credenciales.
23. **Rollback productivo:** sigue siendo el plan documentado, no ejercido contra producción real.
24. **Reservaciones:** fuera de alcance, sin cambios, por instrucción explícita.
25. **Archivos creados:** [`sql/migrations/20251001000000_add_quote_idempotency_keys_table.sql`](sql/migrations/20251001000000_add_quote_idempotency_keys_table.sql), [`internal/db/quote_idempotency.go`](internal/db/quote_idempotency.go), [`13-final-release.md`](13-final-release.md).
26. **Archivos modificados:** [`internal/routes/contact_requests.go`](internal/routes/contact_requests.go) (idempotencia JSON), [`internal/routes/quote_request_postgres_test.go`](internal/routes/quote_request_postgres_test.go) (+16 pruebas), [`internal/db/cart_atomic_postgres_test.go`](internal/db/cart_atomic_postgres_test.go) y [`internal/db/images_migration_postgres_test.go`](internal/db/images_migration_postgres_test.go) (versión final de cadena actualizada de `20250902000000` a `20251001000000`), [`frontend/lib/api/quote.ts`](frontend/lib/api/quote.ts), [`frontend/lib/actions/quote-actions.ts`](frontend/lib/actions/quote-actions.ts), [`frontend/lib/types.ts`](frontend/lib/types.ts), [`frontend/app/(site)/solicitar-cotizacion/page.tsx`](frontend/app/(site)/solicitar-cotizacion/page.tsx), [`frontend/e2e/quote-flow.spec.ts`](frontend/e2e/quote-flow.spec.ts) (+2 pruebas de concurrencia/conflicto), `11-release-candidate.md`/`12-final-verification.md` (notas de actualización).
27. **Dependencias:** ninguna nueva.
28. **Contenedores limpiados:** confirmado, `cart-integration-pg13` eliminado.
29. **Credenciales eliminadas:** confirmado.
30. **Producción:** intacta — no se realizó despliegue, no hubo autorización ni credenciales reales.
31. **Conflictos de escritura:** ninguno.
32. **Desviaciones:** ninguna respecto a la instrucción explícita.
33. **Bloqueos externos:** dominio/HTTPS/reverse proxy/cutover real — heredados, sin evidencia nueva, no inventados.
34. **Estado de `13-final-release.md`:** completo, este documento.

No se declara 100% productivo. Fase 13 Final Release cerrada.
