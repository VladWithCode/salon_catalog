# Fase 12 — Final Verification

> **Actualización (Fase 13 — Final Release, 2026-08-06):** la duplicación persistente de cotizaciones (sección 6 de este documento) fue **resuelta** con una migración nueva (`quote_idempotency_keys`) y protección transaccional real — ya no es un riesgo residual. Ver [`13-final-release.md`](./13-final-release.md).

Fecha: 2026-08-06. Continuación directa de `11-release-candidate.md` (aprobada como Release Candidate técnico). No se desplegó ni modificó producción.

## 1. Protección contra escrituras paralelas

Hashes registrados al inicio para los archivos listados en la instrucción. **Ningún conflicto** — sin modificaciones externas durante la fase.

## 2-3. Entorno real levantado

- PostgreSQL desechable #1 (stack completo): `cart-integration-pg12`, `postgres:16`, `127.0.0.1:55433`, base `cart_integration_local`, usuario dedicado, contraseña de sesión aleatoria.
- PostgreSQL desechable #2 (suite `go test`): `cart-integration-pg-suite`, mismo patrón, `127.0.0.1:55434`.
- Cadena completa de migraciones aplicada en ambos (`goose up`, 23 migraciones, termina en `20250902000000`).
- Fixtures reales insertados (no datos productivos): categoría `Mobiliario`; producto disponible `Mesa Redonda de Prueba` (`mesa-redonda-prueba`, quantity 25); producto no disponible `Silla Sin Stock` (`silla-sin-stock`, quantity 0). Carrito vacío / carrito con producto válido se generan dinámicamente por cada escenario, no como fila precargada.
- Go real: `go run .`, `PORT=8090`, `DATABASE_URL` apuntando al Postgres #1, `CART_COOKIE_SECURE=false` (entorno local sin HTTPS), `CSRF_TRUSTED_ORIGINS=http://127.0.0.1:8090` (el origen que Go presenta a sí mismo en las llamadas server-to-server de Next, no el origen de Next — ver `frontend/lib/api/cart.ts`/`quote.ts`, `goOrigin()`).
- Next real: `bun run build` + Playwright `webServer` (`bunx next start -p 3100`), `GO_API_BASE_URL=http://127.0.0.1:8090`. Sin CORS agregado en ningún punto.

## 4. E2E de cotización — `frontend/e2e/quote-flow.spec.ts` (nuevo)

**6 escenarios reales, 6/6 PASS** en cada uno de los 6 proyectos de Playwright (36 ejecuciones):

1. Cotización con carrito vacío → estado vacío visible, `cart_id` no aparece en el HTML.
2. Flujo completo: agregar producto → abrir cotización → resumen correcto (nombre + cantidad) → envío con datos válidos → confirmación visible → enlace a `/carrito`.
3. Campo requerido ausente → `quote_error=invalid_request`, mensaje visible.
4. Backend unavailable → sin stack/SQL/pgx en el HTML.
5. Sin JavaScript: agregar producto → abrir cotización → llenar formulario → enviar → PRG 303 real a `/solicitar-cotizacion?enviado=1` → confirmación visible, sin ningún fetch de cliente.
6. Sin JavaScript, carrito vacío → estado vacío visible.

**Confirmación en PostgreSQL real** (vía `psql` directo contra `quote_details`, la vista existente — no hay tabla `quote_details` separada, es una vista sobre `quotes` + `cart_items` + `products`): filas de cotización de las corridas de Playwright confirmadas con `cart_items` correcto (`"name":"Mesa Redonda de Prueba"`, `"quantity":1`, `"available":true`, `"max_quantity":25`) — el resumen que Next muestra corresponde exactamente al carrito real en DB, no a datos inventados en el cliente.

Escenarios no cubiertos por Playwright, cubiertos ya por Go (`internal/routes/quote_request_postgres_test.go`, Fase 11, 11/11 PASS, no repetidos aquí): email inválido, teléfono inválido, producto unavailable, quantity inválida, CSRF inválido, cookie inválida, campo desconocido, `/api/quotes` sin auth.

## 5. PRG y flujo sin JavaScript

Confirmado con JavaScript desactivado (`javaScriptEnabled: false`, proyecto `chromium-js-disabled` + prueba dedicada en cada viewport vía `browser.newContext`): el formulario de `/solicitar-cotizacion` es un `<form action={submitQuoteRequestAction}>` real — un Server Action de Next se invoca como un submit nativo de formulario (no fetch de cliente), Next resuelve la redirección `303` sin ningún script ejecutándose. Confirmado también para el carrito (`cart-flow.spec.ts`, ya existente, sin cambios). No depende de Server Actions del lado cliente para completar — la garantía de "funciona sin JS" viene de que un Server Action **es** una submission de formulario real a nivel de protocolo HTTP, no una capa de JavaScript sobre ella.

**Corrección de expectativa inicial:** la instrucción pedía que el formulario apuntara "a una ruta Go progresiva real" cuando JS está desactivado. La arquitectura real (igual que el carrito, aprobada en Fase 10) apunta al Server Action de Next, que a su vez llama a Go server-to-server — nunca al revés. Esto es intencional: el navegador nunca debe hablar directo con Go (evita CORS, mantiene el diseño same-origin). Confirmado funcional igualmente sin JS porque el Server Action mismo es la ruta progresiva.

## 6. Doble envío — riesgo residual confirmado, no oculto

**Evidencia directa** (dos requests JSON reales, disparadas casi simultáneamente contra Go, misma cookie de carrito, mismos datos):

```
curl (paralelo x2) POST /solicitar-cotizacion → {"status":"success"} × 2
SELECT count(*) FROM quotes WHERE customer_name='DoubleSubmitTest'; → 2
```

**Confirmado: dos cotizaciones reales se crean.** No existe protección persistente contra doble envío (sin `Idempotency-Key` en este contrato, sin restricción única en `quotes`). Esto reafirma —con evidencia empírica directa, no solo análisis de código— el hallazgo ya documentado en `11-release-candidate.md`. **No se implementó una migración nueva en esta fase** (instrucción explícita). El formulario mantiene protección de cliente razonable (botón único, sin duplicar controles), pero eso no es una garantía real contra un reenvío de red, doble clic capturado antes del primer re-render, o un actor deliberado. **Garantía multi-instancia requeriría una clave persistente del lado servidor y una decisión de esquema** (tabla o columna de deduplicación) — fuera de alcance de esta fase.

## 7. `CartItem.Available` — regresión confirmada

Nuevo: `internal/db/cart_available_postgres_test.go::TestPostgresLoadItemsReportsAvailableIndependentlyOfQuantity` — **PASS real** contra PostgreSQL:
- Producto `available=true` con `quantity=0` en DB → `CartItem.Available == true`, `MaxQty == 0` (no se infiere disponibilidad de la cantidad).
- Producto `available=false` con `quantity=999` en DB → `CartItem.Available == false`, `MaxQty == 999`.

**Contrato público del carrito confirmado sin cambio:** `GET /api/cart` (`internal/routes/cart_api.go`) usa su propio DTO `cartAPIItem`, que **ya incluía** `"available": bool` en el JSON antes de esta fase — `db.CartItem.Available` (agregado en Fase 11 para `validateQuoteCart`) es un campo interno separado, nunca serializado directamente al navegador; el mapeo pasa siempre por el DTO. No se agregó ni removió ningún campo del contrato público. `internal/cart` (paquete de servicio, tipo `Item` propio) no usa `db.CartItem` y compila sin cambios. Cotización (`internal/routes/contact_requests.go`, `validateQuoteCart`) usa `item.Available` internamente — confirmado en el flujo real de Playwright arriba.

## 8. Matriz Playwright completa

Ejecutada de nuevo íntegra (no reutilizado el 144/144 de Fase 10 — instrucción explícita, justificada porque Fase 11 tocó cotización, `CartItem` y rutas Go):

```
bunx playwright test
```

**Resultado: 180/180 PASS** en los 6 proyectos:

| Proyecto | Resolución / modo |
|---|---|
| `mobile-360` | 360×800, JS on |
| `tablet-768` | 768×1024, JS on |
| `small-desktop-1024` | 1024×768, JS on |
| `desktop-1280` | 1280×800, JS on |
| `wide-1440` | 1440×900, JS on |
| `chromium-js-disabled` | JavaScript desactivado |

Cobertura por spec: `public-pages.spec.ts` (home, servicios, catálogo, producto, experiencia, reservaciones, legales ×3, 404, backend unavailable, teclado/skip-link, footer), `cart-flow.spec.ts` (ciclo completo, header count, sin JS), `quote-flow.spec.ts` (nuevo, 6 escenarios × 6 proyectos = 36). `reduced-motion` cubierto por la config existente de Motion (`MotionConfig reducedMotion="user"`, Fase 9, sin prueba dedicada nueva esta fase — sin cambios en esa área). Cero scroll horizontal: aserción explícita en cada página de `public-pages.spec.ts`, confirmada en las 5 resoluciones.

30/144 pruebas de Fase 10 no repetidas individualmente en este conteo porque **180 ya las incluye** (mismo archivo, misma ejecución) — no hay doble conteo ni reutilización de resultado antiguo.

## 9. Validaciones finales

**PostgreSQL real, `-p 1`:**
```
go test -mod=vendor -p 1 -count=1 -v ./internal/cart/... ./internal/db/... ./internal/routes/...
```
**273/273 PASS, 0 FAIL, 0 deadlocks.**

**Hallazgo actualizado:** `-p 1` ahora es obligatorio para **tres** paquetes, no dos — `internal/routes` ganó pruebas PostgreSQL reales en Fase 11/12 (`quote_request_postgres_test.go`) que también destruyen/recrean el schema `public` de la misma base dedicada. Confirmado por colisión real al correr sin `-p 1` con las variables de entorno de integración exportadas (`goose: ERROR ... no schema has been selected`, `schema "public" already exists`) — no oculto, documentado aquí.

**Sin PostgreSQL (variables de integración sin exportar):**
```
go test -mod=vendor . ./cmd/... ./internal/...
```
Todo verde — las pruebas PostgreSQL se saltan limpiamente (`t.Skip`) cuando `CART_INTEGRATION_TEST_DATABASE_URL` no está definida; confirmado que esto es lo que hace que el comando "plano" de la instrucción no colisione.

`go vet -mod=vendor ./internal/cart/... ./internal/db/... ./internal/routes/...`: limpio.
`go build -mod=vendor ./...`: limpio.

**Frontend:**
```
bun install --frozen-lockfile   → sin cambios (371 instalaciones, 450 paquetes)
bun run lint                     → limpio
bun run test                     → 28/28 PASS
bun run build                    → compilación limpia
bun run test:e2e (bunx playwright test) → 180/180 PASS
```

**Race detector:** no ejecutado — requiere CGO + compilador C; no confirmado disponible en este entorno, y la instrucción prohíbe instalar uno global solo para esto.

## 10. Limpieza

- Go (`PORT=8090`) detenido — `taskkill` sobre el PID real que escuchaba el puerto, confirmado sin listener después.
- Next (`next start -p 3100`, gestionado por `webServer` de Playwright) detenido automáticamente al terminar la ejecución.
- `cart-integration-pg12` y `cart-integration-pg-suite`: `docker stop` + `docker rm` — confirmado `docker ps -a` sin contenedores.
- Puertos `55433`/`55434`: confirmado cerrados (`netstat` sin listeners).
- Archivos temporales con credenciales (`/tmp/p12.txt`, `/tmp/dsn.txt`, `/tmp/seed.sql`, logs con posibles rastros) eliminados — confirmado `ls` falla sobre ellos.
- Producción: sin cambios en ningún punto de la fase.

## Bloqueos productivos

Sin cambio respecto a Fase 11: dominio real y topología de despliegue siguen sin confirmar (sin evidencia en el repositorio); cutover productivo sigue sin ejecutar. **No se declara 100% de producción.**

---

## Entrega final — Fase 12

1. **Estado final:** cerrada, evidencia de navegador para cotización completada.
2. **% técnico:** ~97% (todo lo verificable localmente está verde y probado de punta a punta).
3. **% productivo:** sin cambio — dominio/cutover real no confirmados.
4. **PostgreSQL utilizado:** 2 contenedores desechables `postgres:16`, localhost únicamente, credenciales de sesión aleatorias, ambos eliminados.
5. **Fixtures:** categoría + 2 productos (disponible/no disponible) + carritos generados dinámicamente por escenario.
6. **E2E de cotización:** 6 escenarios reales, PASS en los 6 proyectos Playwright (36 ejecuciones).
7. **Flujo con JavaScript:** confirmado, ciclo completo con confirmación visible.
8. **Flujo sin JavaScript:** confirmado, mismo resultado vía PRG 303 nativo.
9. **PRG 303:** confirmado real, redirección `/solicitar-cotizacion?enviado=1`.
10. **Confirmación en DB:** confirmado vía `psql` contra la vista `quote_details`, `cart_items` correcto.
11. **Validación de campos:** campo requerido ausente probado en Playwright; email/teléfono/CSRF/cookie ya probados en Go (Fase 11).
12. **Carrito vacío:** probado con y sin JS.
13. **Producto unavailable:** ya probado en Go (Fase 11); UI ya lo bloquea (Fase 10, `cart-flow.spec.ts`).
14. **Quantity inválida:** ya probada en Go (Fase 11).
15. **Error de backend:** probado, sin leak de detalles internos.
16. **CSRF:** ya probado en Go (Fase 11, 15/15 + prueba dedicada de cotización).
17. **Cookie:** ya probada en Go (Fase 11, 12 pruebas).
18. **Auth administrativa:** confirmada sin debilitar (`/api/quotes` sigue protegido).
19. **Doble envío:** demostrado con evidencia real — 2 requests paralelas → 2 cotizaciones.
20. **Riesgo residual de duplicados:** confirmado y documentado, no resuelto (requiere decisión de esquema).
21. **`CartItem.Available`:** cubierto con prueba PostgreSQL dedicada, comportamiento independiente de `quantity` confirmado.
22. **Contrato público del carrito:** sin cambios, confirmado (`available` ya estaba en el DTO público antes de esta fase).
23. **Tests PostgreSQL:** 273/273 PASS, 0 deadlocks.
24. **Tests Go:** build+vet limpios, suite completa verde con y sin PostgreSQL.
25. **Tests Bun:** 28/28 PASS.
26. **Playwright:** 180/180 PASS.
27. **Total de pruebas E2E:** 180 (6 proyectos × specs existentes + 6 nuevas de cotización).
28. **Cinco resoluciones:** confirmadas (360×800, 768×1024, 1024×768, 1280×800, 1440×900).
29. **Reduced motion:** configuración existente sin cambios, no reprobada individualmente esta fase.
30. **Teclado:** confirmado (`keyboard navigation reaches the main nav and skip link works`).
31. **Focus:** cubierto por las mismas pruebas de teclado y foco visible ya existentes, sin cambios.
32. **Scroll horizontal:** cero confirmado en las 5 resoluciones, todas las páginas.
33. **Resultado de lint:** limpio.
34. **Resultado de build:** limpio (Go y Next).
35. **Race detector:** no ejecutado — sin CGO/compilador confirmado disponible, no instalado por instrucción explícita.
36. **Contenedor eliminado:** confirmado, ambos.
37. **Credenciales eliminadas:** confirmado, archivos temporales inexistentes tras limpieza.
38. **Procesos detenidos:** confirmado, sin listeners en 8090/3100/55433/55434.
39. **Producción intacta:** confirmado, sin cambios.
40. **Archivos creados:** `frontend/e2e/quote-flow.spec.ts`, `internal/db/cart_available_postgres_test.go`, `12-final-verification.md`.
41. **Archivos modificados:** ninguno de producción — solo el nuevo doc y el nuevo spec/test.
42. **Conflictos de escritura:** ninguno.
43. **Desviaciones:** `-p 1` ahora aplica también a `internal/routes` (hallazgo nuevo, documentado); formulario sin JS apunta al Server Action de Next (no a una ruta Go directa), por diseño arquitectónico ya aprobado, no una desviación real de la instrucción una vez explicado.
44. **Bloqueos externos:** dominio productivo y cutover real siguen sin confirmar — nada nuevo, mismo bloqueo heredado.
45. **Estado de `12-final-verification.md`:** completo, este documento.

No se implementó dominio, canonical ni despliegue. No se declara 100% de producción. Fase 12 Final QA cerrada.
