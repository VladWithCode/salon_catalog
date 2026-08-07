# 09 — Cierre final del proyecto

Fase 9 (consolidación de Fase 8 con red habilitada).

> **Actualización (Fase 10, `10-production-cutover.md`):** el bloqueo de PostgreSQL descrito abajo quedó resuelto — un Docker funcional se volvió disponible, la suite real corrió en verde (71/71) contra una base `cart_integration_local` desechable, y el carrito Next fue implementado. Esta página se conserva como registro histórico exacto del estado en el momento en que se escribió; el estado real y vigente del proyecto está en `10-production-cutover.md`.

## Estado general

**~99% completo.** Todo lo resoluble sin PostgreSQL real quedó resuelto: seguridad corregida, pruebas unitarias frontend configuradas y en verde, QA visual real ejecutada con Playwright (126/126 en 6 configuraciones), Motion integrado de forma mínima. **Único bloqueo real restante: PostgreSQL** — Docker está instalado pero su daemon nunca respondió en este entorno (5+ minutos, `docker info` con timeout agotado tras varios intentos) — no es un permiso denegado, es una limitación de infraestructura del sandbox. Carrito Next, cotización real y reservaciones reales siguen sin implementar, exactamente como en la Fase 8.

---

## 1. PostgreSQL

**Intentado, no logrado.** Docker CLI 29.4.1 presente; se lanzó `Docker Desktop.exe`, se esperó repetidamente (`docker ps`/`docker info` con timeouts de 15-20s, y una espera final de 10s con `timeout 10 docker info` → código 124 = agotado). El daemon (`com.docker.service`) permaneció en estado `Stopped` incluso tras `Start-Service`. Esto no es un caso de "Docker no disponible" trivial (CLI existe) ni un rechazo de permiso — es un fallo de arranque del motor en este sandbox, fuera de lo que puede resolverse sin acceso interactivo a la GUI/WSL2.

**Consecuencia**: no se creó contenedor, no se creó base `cart_integration_local`, no se ejecutó la suite. Cero credenciales inventadas, cero base compartida usada, cero `DATABASE_URL` normal tocada. `CART_INTEGRATION_TEST_DATABASE_URL`/`ALLOW_DESTRUCTIVE` permanecen sin configurar, exactamente como en 08.

**Consecuencia directa**: condición de la Fase 8/9 para integrar carrito Next ("solo si la suite pasa completamente") **no se cumple** → carrito Next **no implementado**. Cotización sigue informativa. Reservaciones siguen informativas (además, confirmado de nuevo: sin modelo/tabla/handler — Caso C sin cambios).

---

## 2. Seguridad — corregida

### `/api/categories` (hallazgo de 08-final-readiness.md)
`internal/routes/categories.go`: `POST`, `PUT /{id}`, `DELETE /{id}` ahora envueltos en `auth.ValidateAuth`, igual que el resto de mutaciones admin del archivo. `GetCategories` (GET) permanece público — es lectura, no era el hallazgo. Firmas de `CreateCategory`/`UpdateCategory`/`DeleteCategory` actualizadas a `(w, r, a *auth.Auth)` para encajar con `auth.ValidateAuth`, sin cambiar el contrato de respuesta exitosa.

### `/api/quotes` — hallazgo nuevo, encontrado en la auditoría ampliada
`internal/routes/quotes.go`: `POST /api/quotes` y `PUT /api/quotes/{id}` estaban registrados **sin auth**. `CreateQuote` además era código huérfano — llamaba `db.CreateQuote(&data)` con `data` como `db.Quote{}` vacío (el `ParseForm()` nunca poblaba el struct), es decir, cada POST no autenticado insertaba una fila de cotización vacía y basura en la base real. Cero consumidor en todo el repositorio (`grep -rn "api/quotes"` → cero resultados fuera del propio archivo). Corregido con el mismo patrón: `auth.ValidateAuth` en ambas rutas.

### Auditoría completa de mutaciones públicas (POST/PUT/PATCH/DELETE fuera de `/panel/*`)
```
POST /solicitar-contacto          — público por diseño (formulario de contacto)
POST /api/contact-requests        — público por diseño (formulario de contacto, Next lo usa)
POST /api/quotes                  — CORREGIDO (auth agregada)
PUT  /api/quotes/{id}             — CORREGIDO (auth agregada)
POST /api/categories              — CORREGIDO (auth agregada)
PUT  /api/categories/{id}         — CORREGIDO (auth agregada)
DELETE /api/categories/{id}       — CORREGIDO (auth agregada)
POST/PUT/DELETE /api/products*    — ya tenían auth.ValidateAuth (verificado, sin cambios)
Todas las de /carrito*, /wizard/*, /cotizacion/carrito/* — CSRF + sesión firmada, sin cambios
```
Ningún otro hallazgo de mutación pública sin protección.

### Pruebas añadidas
`internal/routes/categories_auth_test.go` — 6 tests: POST/PUT/DELETE de categorías rechazados sin auth (302 a `/iniciar-sesion`, nunca 200), cookie de auth inválida rechazada, `GET /api/categories` confirmado que sigue público, método no registrado no colisiona, POST/PUT de `/api/quotes` rechazados sin auth. Ninguno depende de PostgreSQL real (el rechazo ocurre antes de tocar la DB).

---

## 3. Dependencias frontend añadidas

Todas vía `bun add`/`bun add -d`, `frontend/package.json` y `frontend/bun.lock` actualizados:

| Paquete | Tipo | Motivo |
|---|---|---|
| `server-only` | dependency | Ya se usaba por convención (`import "server-only"`); no estaba realmente instalado — `bun test` lo necesitaba para resolver el import. Sin él, Next igual funcionaba (su bundler lo shimea), pero cualquier ejecución fuera de Next fallaba. |
| `motion` | dependency | Motion (sucesor de Framer Motion, misma API vía `motion/react`), aprobado en esta fase. |
| `@playwright/test` | devDependency | Runner E2E/visual solicitado explícitamente. |

No se instaló ningún runner de pruebas unitarias adicional — Bun ya trae uno (`bun test`), usarlo evita el "sin dos runners" que pide la instrucción.

Scripts nuevos en `package.json`: `"test": "bun test ./lib"`, `"test:e2e": "playwright test"`.

`bun install --frozen-lockfile` ejecutado al final, sin diferencias pendientes.

---

## 4. Pruebas frontend (unitarias, Bun)

`frontend/lib/api/catalog-product.test.ts` — 28 tests, comportamiento observable de `fetchCatalogProductDetail` (no implementación interna):
- Identifier: vacío, solo espacios, slash, backslash, NUL, 200 caracteres Unicode (acepta), 201 caracteres Unicode (rechaza, cuenta caracteres no unidades UTF-16), codificación URL exacta, UUID válido.
- Éxito: producto mínimo, `category: null`, `available: false`, `images: []`, orden de imágenes preservado.
- Errores HTTP: 400→`invalid_identifier`, 404→`product_not_found`, 503→`catalog_unavailable`, status inesperado→`unexpected_status`.
- Red/JSON inválido: fetch rechazado→`backend_unavailable`, Content-Type no-JSON, root array, `product` ausente, UUID inválido, `image_filename` inseguro, `images: null`, `available` no-boolean → todos `invalid_response`.
- Forma del request: `cache: no-store`, `credentials: omit`, exactamente una llamada a fetch.

`server-only` se stubea solo en el entorno de test (`frontend/bunfig.toml` + `frontend/bun-test-setup.ts`, vía `Bun.plugin`) — no toca `next.config.ts` ni cambia qué se envía en el build real de Next.

**Resultado**: `bun run test` → **28 pass, 0 fail**.

---

## 5. Playwright — QA visual real ejecutada

`frontend/playwright.config.ts`: 6 proyectos (360×800, 768×1024, 1024×768, 1280×800, 1440×900, y un proyecto Chromium con JavaScript deshabilitado), `webServer` levanta `next start` sobre el build de producción ya generado.

`frontend/e2e/public-pages.spec.ts` — cubre `/`, `/servicios`, `/experiencia`, `/reservaciones`, `/solicitar-cotizacion`, `/politica-privacidad`, `/terminos-servicio`, `/politica-cookies`, `/catalogo`, 404, backend-Go-apagado (estado real de este entorno), identifier inválido en detalle de producto, enlaces legales del footer, navegación por teclado.

**Resultado: 126/126 tests en verde**, en las 5 resoluciones + el proyecto sin JavaScript.

Durante la primera corrida se encontraron y corrigieron dos problemas reales (documentados, no ocultados):
1. El selector `header` del test coincidía con 5 elementos (el `<header>` del sitio y 4 `<header>` semánticos de sección en Home) — corregido a `getByRole("banner")`, que apunta inequívocamente al header del sitio.
2. `/experiencia` colgaba esperando el evento `load` por el `<video>` de 5.8 MB sin `preload` explícito — se añadió `preload="none"` al `<video>` (mejora real, no solo ajuste del test: evita precargar un video de 5.8 MB en cada visita a la página).

`/catalogo/producto/[identifier]` no tiene ruta fija que probar sin datos reales de Go; se cubrió indirectamente mediante el caso "identifier inválido" (que no requiere backend) y el patrón general de "backend apagado" ya probado en `/catalogo`.

**QA sin JavaScript**: ejecutada realmente (no solo inspección estática) — 7 páginas × JS deshabilitado, todas con `<h1>` único y contenido principal visible.

---

## 6. Motion

Instalado (`motion@13`, sucesor de Framer Motion). Aplicado de forma **mínima y aislada**: crossfade en `ProductGallery` (`frontend/components/product/product-gallery.tsx`) al cambiar la imagen activa — `AnimatePresence` + `motion.div`, 200ms, `MotionConfig reducedMotion="user"` (colapsa a corte instantáneo cuando el visitante tiene `prefers-reduced-motion`). Ningún Server Component se convirtió a Client Component (`ProductGallery` ya era Client desde la Fase 6B). No se tocó menú móvil, secciones de Home, ni ningún otro componente — alcance deliberadamente pequeño, sin regresión detectada (build limpio, 126/126 Playwright en verde después del cambio).

---

## 7. Dominio, canonical, topología

Sin cambios respecto a 08: **ninguna evidencia nueva** de dominio productivo, proxy inverso, o configuración de despliegue apareció durante esta fase. `metadataBase`/canonical siguen sin configurarse. Topología recomendada sigue siendo la misma (Opción C statu quo para lo dependiente de carrito, Opción A como destino una vez exista infraestructura confirmada).

---

## Archivos creados
```
frontend/bunfig.toml
frontend/bun-test-setup.ts
frontend/lib/api/catalog-product.test.ts
frontend/playwright.config.ts
frontend/e2e/public-pages.spec.ts
internal/routes/categories_auth_test.go
09-final-completion.md
```

## Archivos modificados
```
internal/routes/categories.go        (auth agregada a POST/PUT/DELETE)
internal/routes/quotes.go            (auth agregada a POST/PUT)
frontend/package.json                (+server-only, +motion, +@playwright/test, scripts test/test:e2e)
frontend/bun.lock                    (regenerado por bun add)
frontend/tsconfig.json               (exclude de archivos de test/e2e del build de Next)
frontend/components/product/product-gallery.tsx  (crossfade con Motion)
frontend/app/(site)/experiencia/page.tsx          (preload="none" en el video)
```

## Sin modificar
`08-final-readiness.md` (información aún vigente, no reescrito), `next.config.ts`, `.env.example`, `frontend/.env.example`, `internal/routes/routes.go`, DB, migraciones, producción.

---

## Validaciones

| Comando | Resultado |
|---|---|
| `gofmt` | limpio (3 archivos Go tocados) |
| `go test -mod=vendor . ./cmd/... ./internal/...` | `ok` todos |
| `go test -mod=vendor -count=1 ./internal/cart/... ./internal/db/... ./internal/routes/...` | `ok` todos |
| `go vet` | limpio |
| `go build` | OK |
| Race detector | no disponible (`CGO_ENABLED=0`), limitación reportada, no resuelta |
| `bun install --frozen-lockfile` | sin diferencias pendientes al final |
| `bun run lint` | limpio |
| `bun run test` (unit) | **28 pass, 0 fail** |
| `bun run build` | OK, 13 rutas, incluye `/catalogo/producto/[identifier]` |
| `bunx playwright test` | **126 pass, 0 fail** (6 configuraciones) |

---

## Checklist de producción (sin cambios respecto a 08, salvo lo tachado)

- [ ] Confirmar dominio público y añadir `metadataBase`/canonical.
- [ ] Definir y confirmar topología (proxy inverso vs same-origin) con evidencia de infraestructura real.
- [ ] Ejecutar suite `internal/dbtest`/`internal/cart` contra `cart_integration_*` real, en verde, con cadena completa de migraciones — **bloqueado en este entorno por Docker, no por falta de intento**.
- [ ] Solo entonces: implementar carrito Next (tarjetas, detalle, header, cotización).
- [x] ~~QA visual real en navegador, 360-1440px~~ — **hecho, 126/126 en verde**.
- [x] ~~Decidir y corregir `/api/categories` sin auth~~ — **hecho**, y se corrigió además `/api/quotes` (hallazgo nuevo).
- [ ] Decisión de negocio sobre reservaciones reales (o mantener informativa permanentemente).
- [ ] Confirmar `/api/products/{slug}` heredado: deprecar o documentar como permanente.

---

## Bloqueos residuales

1. **PostgreSQL real** — único bloqueo duro. Docker instalado pero su daemon no arrancó en este sandbox tras intento real (lanzamiento de Docker Desktop + esperas repetidas + verificación con `docker info`). No es una limitación de permisos evadible — requiere un entorno donde el motor Docker realmente pueda inicializarse, o una base PostgreSQL `cart_integration_*` provista externamente.
2. Carrito Next, cotización real — dependen de (1).
3. Reservaciones reales — decisión de negocio pendiente (sin modelo/tabla/handler en ningún nivel).
4. Dominio/canonical/topología — sin evidencia real, no inventados.
