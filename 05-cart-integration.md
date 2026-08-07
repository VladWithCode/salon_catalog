# Fase 5B7B1 — Auditoría y contrato seguro para integrar Next con el carrito Go

## 0. Alcance, estado y conclusión ejecutiva

Este documento es la fuente de verdad de la futura integración del carrito. La fase es exclusivamente documental: no crea endpoints, no cambia cookies, no modifica rutas, no toca la base de datos y no añade componentes.

La arquitectura preferente **encaja con el código real, pero no está lista para implementarse todavía**:

- Go debe seguir siendo la única fuente de verdad de carrito, stock, cantidades y cotización.
- La mejora con JavaScript debe consumir contratos JSON mínimos servidos por Go bajo el mismo origen público.
- El funcionamiento base debe usar formularios HTML `POST` hacia Go y `303 See Other`; Next no debe convertirse en un Client Component completo.
- Go debe ser el único proceso que genera, firma, valida, rota y expira `cart_id`. Next nunca debe interpretar ni fabricar esa cookie.
- Antes de cualquier mutación desde Next son bloqueantes la autenticidad de la sesión del carrito, la política explícita de cookie, la validación de origen/CSRF, la validación estricta de IDs y cantidades, y la actualización atómica contra el stock vigente.
- No se encontró configuración productiva de proxy, dominio, TLS o despliegue. Sin esa decisión no es posible garantizar una cookie de primera parte ni rutas same-origin.

**Recomendación principal:** una combinación de la **Opción C** para la mejora JavaScript y la **Opción A** como fallback HTML obligatorio. La Opción D permanece como estado seguro y mecanismo de rollback hasta cerrar todos los bloqueantes. La Opción B solo es un fallback topológico si no se puede enrutar directamente a Go bajo el mismo origen; no debe ser la primera elección.

> El carrito no puede integrarse de forma segura hasta confirmar que Next y Go compartirán un origen y una política de cookies compatible.

No se propone precio porque el modelo real no tiene precio.

---

## 1. Evidencia revisada y límites de la auditoría

### 1.1 Fuentes revisadas

- Guías: `AGENTS.md`, `README.md`, `05-catalog-page.md`.
- Next actual: `frontend/app/(site)/catalogo/page.tsx`, `frontend/components/catalog/catalog-product-card.tsx`, `frontend/lib/api/catalog-browse.ts`, `frontend/lib/types.ts`, `frontend/lib/env.ts`, `frontend/next.config.ts`, `frontend/.env.example`.
- Router y handlers Go: `internal/routes/routes.go`, `internal/routes/cart.go`, `internal/routes/catalog.go`, `internal/routes/contact_requests.go`, `internal/routes/quotes.go`, `internal/routes/wizard.go`, `internal/routes/wizard_catalog.go`.
- Persistencia y modelos: `internal/db/cart.go`, `internal/db/catalog.go`, `internal/db/products.go`, `internal/db/quotes.go`.
- UI Go: `internal/templates/components/cart.templ`, `internal/templates/components/catalog.templ`, `internal/templates/pages/catalog.templ`, `internal/templates/pages/quote_request.templ`, `internal/templates/components/wizard.templ`, `internal/templates/components/wizard_script.templ`, `web/static/js/eventManager.js`.
- Esquema: `sql/migrations/20250703200655_add_products_table.sql`, `20250726033344_add_carts_table.sql`, `20250726183719_add_quotes_table.sql`, `20250809175837_add_catalog_products_stock_column.sql`, `20250809194425_add_cart_customer_columns.sql`, `20250813000000_add_quote_details_view.sql`, `20250815000000_add_wizards_tables.sql`.
- Pruebas existentes: `internal/db/catalog_test.go` e `internal/routes/public_api_test.go`. No existen pruebas específicas de carrito, cookie de carrito, cotización con carrito ni concurrencia.

Los archivos `*_templ.go` generados reflejan los `.templ`; las referencias de este documento apuntan a los archivos fuente `.templ`.

### 1.2 Límites

- No hay `DATABASE_URL` segura de staging ni PostgreSQL de pruebas; no se ejecutaron consultas reales ni pruebas de concurrencia DB.
- No existe `.git`; la ausencia de cambios se verifica con inventario y hashes del filesystem, no con `git diff`.
- No hay evidencia ejecutable de proxy inverso, dominio, subdominio, HTTPS ni despliegue productivo.
- No se probó una cookie real en producción ni comportamiento detrás de un terminador TLS.
- La auditoría es estática y de pruebas locales existentes. Las condiciones de carrera se deducen del patrón leer-modificar-escribir sin bloqueo.
- No se repite la auditoría general del catálogo; se usa únicamente lo necesario para definir el límite con el carrito.

---

## 2. Fuente de verdad y Flujo actual

### 2.1 Mapa de rutas efectivo

| Método y ruta | Registro | Middleware | Respuesta actual | Consumidor |
|---|---|---|---|---|
| `GET /carrito` | `internal/routes/cart.go:16-22` | `publicMiddleware` | Fragmentos templ multi-target | HTMX del sidebar |
| `PUT /carrito` | `internal/routes/cart.go:16-22` | `publicMiddleware` | Fragmentos templ + toast | Tarjetas/modal Go |
| `PATCH /carrito/items` | `internal/routes/cart.go:16-22` | `publicMiddleware` | Fragmentos templ + toast | Controles del sidebar |
| `DELETE /carrito/items` | `internal/routes/cart.go:16-22` | `publicMiddleware` | Fragmentos templ + toast | “Limpiar todo” |
| `DELETE /carrito/items/{id}` | `internal/routes/cart.go:16-22` | `publicMiddleware` | Fragmentos templ + toast | Eliminar línea |
| `GET /solicitar-cotizacion` | `internal/routes/contact_requests.go:36-44` | Ninguno | Página templ completa | Navegación pública |
| `POST /solicitar-cotizacion` | `internal/routes/contact_requests.go:36-44` | Ninguno | Fragmentos templ + toast | `hx-post` del formulario |
| `PUT /cotizacion/carrito/items/{id}` | `internal/routes/contact_requests.go:41-44` | Ninguno | Fragmentos templ | Controles en cotización |
| `DELETE /cotizacion/carrito/items/{id}` | `internal/routes/contact_requests.go:41-44` | Ninguno | Fragmentos templ | Eliminar en cotización |
| `POST /wizard/{wizard_id}/complete` | `internal/routes/wizard.go:17-22` | Ninguno | Vista templ de finalización | Wizard HTMX/JS |

En las rutas con `{id}`, el valor real es `product_id`; `cart_items` no tiene un ID autónomo. Su clave primaria es `(cart_id, product_id)` (`20250726033344_add_carts_table.sql:9-18`).

### 2.2 Creación y configuración actual de `cart_id`

`publicMiddleware` vive en `internal/routes/routes.go:284-311`. Se aplica a las páginas públicas registradas en `routes.go:21-30` y a las cinco rutas de carrito, pero **no** a `/solicitar-cotizacion`, sus mutaciones de ítems ni el wizard.

| Propiedad | Estado actual | Evidencia/efecto |
|---|---|---|
| Generación | Server-side, UUIDv7 | `uuid.Must(uuid.NewV7()).String()` en `routes.go:300-308` |
| Solicitudes que la crean | Toda ruta envuelta por `publicMiddleware` cuando `r.Cookie("cart_id")` devuelve error | Incluye páginas editoriales que no usan carrito; no incluye cotización/wizard |
| `Path` | `/` | `routes.go:307` |
| `MaxAge` | No establecido (`0`) | Solo se configura `Expires` |
| `Expires` | `now + 30 días` | `routes.go:306` |
| `HttpOnly` | `false` por valor cero | No se configura |
| `Secure` | `false` por valor cero | No se configura |
| `SameSite` | No especificado (`SameSiteDefaultMode`) | Depende del navegador; no es una política explícita |
| `Domain` | No especificado | Cookie host-only |
| Codificación | UUID canónico ASCII en texto plano | Sin firma, MAC ni cifrado |
| Validación de entrada | Solo comprueba que `r.Cookie` no devuelva error | No valida vacío, UUID, versión, firma ni existencia/propiedad |
| Cookie inválida | Si el header puede parsearse pero el valor no es UUID, se acepta y llega a PostgreSQL; puede producir error de tipo UUID | `GetCartIDFromRequest` devuelve el valor literalmente (`internal/db/cart.go:420-426`) |
| Cookie ajena | Se acepta y se consulta directamente | Permite leer/modificar cualquier carrito cuyo UUID sea conocido |
| Rotación | Ninguna | No rota al detectar valor inválido, cotizar o cambiar de contexto |
| Regeneración | Solo cuando `r.Cookie` devuelve error | No ocurre por valor vacío, UUID ajeno o carrito sometido |

**Fallo de primera solicitud:** `publicMiddleware` añade `Set-Cookie` a la **respuesta**, pero llama al handler con un request que todavía no contiene esa cookie (`routes.go:300-310`). Por tanto, una primera llamada directa a `GET /carrito` o `PUT /carrito` puede emitir una cookie nueva y simultáneamente fallar al leerla mediante `GetCartIDFromRequest`. El ID debe estar disponible en el contexto del mismo request cuando se implemente 5B7B2.

La constante `db.DefaultCartCookieDuration` vale 30 días (`internal/db/cart.go:16`), pero el middleware repite el cálculo en vez de usarla.

### 2.3 Lectura del carrito

1. `db.GetCartIDFromRequest` devuelve `cookie.Value` sin validación (`internal/db/cart.go:420-426`).
2. `db.GetOrCreateCart` llama a `FindCartByID`; solo crea si el error es exactamente `ErrCartNotFound` (`cart.go:388-400`). Un ID vacío o inválido no se recupera.
3. `FindCartByID` consulta `carts` por `$1`, carga datos del cliente y después `LoadItems` (`cart.go:339-385`).
4. `LoadItems` une `cart_items` con la vista `catalog_products` y obtiene nombre, categoría, imagen y **cantidad vigente del producto** como `max_quantity` (`cart.go:302-336`).
5. Un carrito inexistente con UUID bien formado se crea con ese mismo UUID. Dos primeras solicitudes concurrentes pueden competir por el mismo `INSERT` porque no hay `ON CONFLICT` útil cuando no existen campos a actualizar.
6. Un carrito existente sin ítems devuelve `Items: []`; un error DB se convierte en fragmento de error o, en cotización, en un carrito vacío de fallback.

`GET /carrito` no devuelve una página JSON ni una representación estable. `renderCart` produce juntos `cartToggle`, `cartSidebar` y opcionalmente `toaster-toast`, usando `util.RenderMixedWithFragments` (`internal/routes/cart.go:237-255`). La UI depende de:

- HTMX para `hx-get`, `hx-put`, `hx-patch`, `hx-delete` y swaps fuera de banda.
- `web/static/js/eventManager.js` para delegación de eventos HTMX.
- `CartScript` en `internal/templates/components/cart.templ:307-426` y GSAP para abrir, cerrar, contar y animar el sidebar.

Sin JavaScript el botón flotante no carga el carrito y las mutaciones con métodos distintos de `GET`/`POST` no tienen fallback nativo.

### 2.4 Añadir un producto

| Aspecto | Comportamiento real |
|---|---|
| Ruta/método | `PUT /carrito` |
| Content-Type efectivo | HTMX envía campos de formulario, normalmente `application/x-www-form-urlencoded`; el handler llama `r.ParseForm()` pero no exige media type |
| Campos | `product_id`, `source` |
| Validación `product_id` | No exige UUID ni presencia. `FindCatalogProductDetail` interpreta un texto no UUID como slug; luego el código conserva el valor original como `CartItem.ProductID`, que puede fallar al insertarse en una columna UUID |
| Validación `source` | Ninguna. El cliente controla el valor que se guarda en `VARCHAR(8)` |
| Cantidad solicitada | No existe campo: siempre añade una unidad |
| Duplicado | Busca por `ProductID`; suma 1 y aplica `UpdateItemQty` |
| Producto inexistente | El error de consulta termina como error genérico HTML y normalmente 500, no 404 |
| Disponibilidad | No se comprueba `prod.Available` |
| Stock cero | En una línea nueva se guarda `Quantity: 1` aunque `prod.Quantity` sea 0; `AddItem` no aplica el límite |
| Stock máximo | En duplicados `UpdateItemQty` limita con `min(quantity, MaxQty)` |
| Cookie ausente | En la primera solicitud puede establecer cookie en respuesta y aun fallar el handler |
| Error DB | Se registra internamente y se devuelve fragmento genérico; el manejo de status escribe 500 incluso después de un 400 previo |
| Respuesta | Fragmentos HTML para sidebar/contador/toast; no reutilizable como contrato React |

Evidencia principal: `internal/routes/cart.go:44-118`; botones Go en `internal/templates/components/catalog.templ:165-175`, `207-243` y `310-330`.

**Corrección a la hipótesis anterior sobre `MaxQty`:** `MaxQty` no se persiste en `cart_items`. Se toma de `products.quantity` cada vez que `LoadItems` lee el carrito (`internal/db/cart.go:310-329`). Sí queda congelado durante una operación individual entre lectura y escritura, y una cantidad ya almacenada no se corrige automáticamente si el stock baja. El riesgo real es falta de revalidación atómica, no un valor máximo permanentemente congelado en la tabla.

### 2.5 Actualizar cantidades

- Ruta principal: `PATCH /carrito/items`, campos `id`, `action` y, para `set`, `quantity` (`internal/routes/cart.go:120-171`).
- `action` admite `increase`, `decrease` y `set`.
- `set` ignora el error de `strconv.Atoi`; texto inválido se vuelve 0 y elimina el ítem (`cart.go:146-153`).
- Cero y negativos eliminan la línea por diseño de `UpdateItemQty` (`internal/db/cart.go:151-164`).
- Valores excesivos se limitan a `MaxQty`, pero no se informa el valor efectivo en un contrato estructurado.
- Un ID inexistente no modifica nada, pero el handler responde éxito y ejecuta `Save`.
- El endpoint alterno de cotización `PUT /cotizacion/carrito/items/{id}` sí rechaza cantidades no numéricas, pero cero/negativo eliminan y mantiene las mismas condiciones de carrera (`contact_requests.go:570-673`).
- No hay transacción que bloquee la fila de producto o carrito durante leer-modificar-escribir. Dos incrementos simultáneos pueden leer la misma cantidad y ambos escribir el mismo resultado, perdiendo una actualización.
- La tabla PostgreSQL usa `INTEGER`; el handler acepta el rango nativo de `int` de Go y no impone un máximo de negocio antes de limitar. Se debe validar un rango pequeño y explícito antes de operar.
- El único vínculo de propiedad es el `cart_id` de la cookie. Con una cookie ajena, todas estas operaciones actúan sobre el carrito ajeno.

### 2.6 Eliminar y vaciar

- `DELETE /carrito/items/{id}` elimina por `(cart_id de cookie, product_id de path)` (`internal/routes/cart.go:173-204`).
- `DELETE /carrito/items` carga el carrito, acumula todos los `product_id` y borra las filas; no elimina la fila `carts` (`cart.go:206-235`, `internal/db/cart.go:181-187`, `286-297`).
- `RemoveItem` añade el ID a `removedItems` incluso si no existe; el `DELETE` DB no comprueba filas afectadas. En la práctica eliminar una línea inexistente es idempotente, pero la respuesta afirma que se eliminó.
- La propiedad queda acotada por `cart_id` en la consulta DB, pero esa propiedad no es confiable mientras la cookie no esté autenticada.
- No hay validación UUID del path ni protección CSRF/origen.

### 2.7 Wizard

El wizard conserva selecciones en `localStorage`, las adjunta como `wizard_localStorage` a peticiones HTMX y `POST /wizard/{wizard_id}/complete` vuelve a resolver cada producto y escribe en el mismo carrito (`internal/templates/components/wizard_script.templ:27-109`; `internal/routes/wizard_catalog.go:142-229`). Comparte los problemas de cookie, CSRF, disponibilidad, stock y concurrencia. No debe migrarse ni cambiarse como parte inicial del carrito Next; sí debe entrar en regresiones.

### 2.8 Cotización y persistencia histórica

#### Lectura y vínculo

- `GET /solicitar-cotizacion` intenta leer `cart_id`; si falta usa `cartID = ""`, `GetOrCreateCart` falla y la página degrada a carrito vacío (`internal/routes/contact_requests.go:299-331`). La ruta no está envuelta por `publicMiddleware`.
- `POST /solicitar-cotizacion` lee otra vez la cookie, carga el carrito y asigna `quote.CartID` solo si `cart.ID != ""` (`contact_requests.go:333-455`).
- `db.CreateQuote` inserta nombre, teléfono, fechas, estado, comentarios, `cart_id`, tipo y evento (`internal/db/quotes.go:84-127`). **No copia los ítems**.

#### Después del envío

- No se limpia el carrito.
- No se rota la cookie.
- No se marca `carts.is_submitted`.
- No se bloquean cambios posteriores.
- No existe idempotencia: repetir el POST puede crear varias cotizaciones y varias notificaciones.
- El éxito devuelve fragmentos HTMX y un script redirige al inicio 2.6 segundos después (`quote_request.templ:301-317`); no hay `303` del servidor.

#### Stock e historial

`quote_details` reconstruye los ítems en cada lectura mediante un `LEFT JOIN` vivo entre `quotes`, `cart_items` y `products` (`20250813000000_add_quote_details_view.sql:3-44`). Por ello:

- Cambiar el carrito después de cotizar cambia lo que el panel ve como detalle de esa cotización.
- Borrar un producto elimina su `cart_item` por cascada y puede quitarlo del detalle histórico.
- Borrar el carrito pone `quotes.cart_id` en `NULL`.
- `max_quantity` y `available` reflejan el producto actual, no el estado al cotizar.
- La cantidad del carrito no se revalida contra stock dentro de la creación de cotización.

Esto preserva el vínculo funcional actual, pero **no es un snapshot histórico**. La decisión entre congelar un snapshot o conservar la vista viva pertenece a 5B7B9 y puede requerir migración explícita; no debe improvisarse durante la API de carrito.

#### Sin JavaScript hoy

El formulario tiene `hx-post` pero no `action` ni `method` HTML (`internal/templates/pages/quote_request.templ:35-47`). Sin HTMX el navegador usa el comportamiento por defecto del formulario, por lo que el envío actual no conserva la semántica POST. Los controles de cantidad/eliminación también son solo HTMX. La futura fase debe corregirlo sin eliminar la ruta Go.

---

## 3. Auditoría de seguridad y robustez

Las clasificaciones son decisiones de orden, no correcciones realizadas.

| ID | Hallazgo y evidencia | Impacto | Clasificación | Recomendación documental |
|---|---|---|---|---|
| CART-01 | `cart_id` se acepta literalmente (`internal/db/cart.go:420-426`) | Fijación, lectura y modificación de carrito ajeno | **Bloqueante antes de integrar Next** | Cookie autenticada y validación central; nunca aceptar un UUID desnudo como autoridad |
| CART-02 | Cookie sin `HttpOnly`, `Secure` ni `SameSite` explícito (`routes.go:300-308`) | Robo por XSS, transporte HTTP, política CSRF ambigua | **Bloqueante antes de integrar Next** | Política de §7, con `Secure` condicionada a HTTPS confirmado |
| CART-03 | Primera respuesta crea cookie pero el handler no recibe el ID | Primera operación directa puede fallar; sesión inconsistente | **Bloqueante antes de integrar Next** | Resolver una vez y propagar ID confiable por contexto en el mismo request |
| CART-04 | Mutaciones sin token, Origin o Referer; no existe middleware CORS/CSRF | CSRF sobre añadir, cambiar, borrar, wizard y cotización | **Bloqueante antes de integrar Next** | Validación estricta de origen §8 antes de exponer mutaciones |
| CART-05 | No se valida `product_id` como UUID y un no UUID se trata como slug (`cart.go:66-100`, `catalog.go:117-169`) | Errores DB, semántica ambigua, abuso de IDs | **Bloqueante antes de integrar Next** | UUID canónico, existencia y respuesta 404 segura |
| CART-06 | Añadir no comprueba `Available` ni stock positivo | Se puede guardar producto agotado/no disponible | **Bloqueante antes de integrar Next** | Validar disponibilidad y stock en la misma operación DB |
| CART-07 | Actualizaciones leen, mutan en memoria y hacen upsert absoluto sin bloqueo | Lost update y sobreasignación con solicitudes concurrentes | **Bloqueante antes de integrar Next** | Operación atómica/transacción con bloqueo apropiado y relectura de stock |
| CART-08 | `Atoi` silencioso convierte texto inválido en 0 y elimina | Pérdida de línea por entrada inválida | **Bloqueante antes de integrar Next** | JSON/form estricto; rechazar con 400/422, nunca mutar |
| CART-09 | Cero/negativos equivalen a eliminar y no hay rango explícito | Semántica sorprendente; enteros extremos | **Bloqueante antes de integrar Next** | PATCH solo acepta `1..max`; DELETE es la única eliminación |
| CART-10 | `source` es controlado por el cliente y no validado | Datos falsos y errores por `VARCHAR(8)` | **Debe corregirse antes del cutover** | El servidor fija `catalog`/`wizard`; excluirlo del request público |
| CART-11 | `MaxQty` se relee, pero stock puede cambiar entre lectura y `Save`; cantidad almacenada no se normaliza al cargar | Stock desactualizado y respuesta incoherente | **Bloqueante antes de integrar Next** | Revalidación atómica en cada mutación y al cotizar |
| CART-12 | Borrar/actualizar un `product_id` inexistente responde éxito | Observabilidad y UX engañosas | **Debe corregirse antes del cutover** | Semántica explícita: DELETE idempotente 204; PATCH 404 |
| CART-13 | Errores de carrito pueden escribir 400 y luego 500; respuestas son HTML no tipado (`cart.go:258-285`) | Status incorrecto; cliente no distingue fallos | **Debe corregirse antes del cutover** | Escritor JSON único, códigos seguros y logs con correlación |
| CART-14 | No se exige `Content-Type`, tamaño máximo ni cuerpo JSON con campos desconocidos rechazados | Ambigüedad de parser y abuso de cuerpo | **Bloqueante antes de integrar Next** | JSON estricto y `MaxBytesReader`; forms solo urlencoded/multipart esperado |
| CART-15 | No hay rate limiting en carrito/cotización | Spam, presión DB, crecimiento de carritos/cotizaciones | **Debe corregirse antes del cutover** | Límite por sesión/IP en mutaciones y cotización; métricas de rechazo |
| CART-16 | POST de cotización no tiene idempotencia ni bloqueo de doble envío | Cotizaciones/mensajes duplicados | **Debe corregirse antes del cutover** | Idempotency key/nonce de operación y restricción transaccional |
| CART-17 | Cotización guarda solo `cart_id`; el detalle es un join vivo | Cambios posteriores alteran el historial visible | **Debe corregirse antes del cutover** | Aprobar snapshot o declarar formalmente semántica viva antes de 5B7B9 |
| CART-18 | `/solicitar-cotizacion` y wizard no usan el middleware que crea cookie | Entrada directa degrada a carrito vacío o falla | **Debe corregirse antes del cutover** | Un único resolver de sesión en todas las rutas dependientes |
| CART-19 | `Expires` existe pero `MaxAge` no; no hay limpieza DB | Cookies y filas abandonadas divergen; crecimiento indefinido | **Puede corregirse después** | Alinear ambos atributos y diseñar job de retención con protección de quotes |
| CART-20 | No hay rotación al valor inválido, firma fallida, cotización o transición de estado | Persistencia excesiva/fijación | **Bloqueante antes de integrar Next** para inválidos; **Debe corregirse antes del cutover** para post-cotización | Rotar en inválido; decidir rotación tras cotización |
| CART-21 | Las queries de carrito usan parámetros (`$1`, `NamedArgs`) | No se observa SQL injection en este flujo | **Fuera del alcance** como hallazgo | Mantener queries parametrizadas y columnas dinámicas solo desde allowlist interna |
| CART-22 | Templ y React escapan texto por defecto | No se observa XSS directo en tarjeta/carrito | **Fuera del alcance** como hallazgo | Mantener escape; no usar `dangerouslySetInnerHTML` |
| CART-23 | Wizard convierte nombres leídos con `textContent` y los reinserta con `innerHTML` (`wizard_script.templ:255-292`) | Posible XSS almacenado si un nombre contiene markup | **Fuera del alcance** de 5B7; debe auditarse antes de migrar wizard | Sustituir construcción HTML por nodos/textContent en fase propia |
| CART-24 | Carrito anónimo sin login; “autorización” depende solo de cookie | Correcto como intención anónima, inseguro con cookie no autenticada | **Bloqueante antes de integrar Next** | Tratar cookie autenticada como credencial bearer; no usar auth de panel |
| CART-25 | No hay CORS y los fetchers actuales usan `credentials: "omit"` | El catálogo read-only funciona server-side; carrito aún no transporta cookie | **Fuera del alcance** como hallazgo actual | Mantener sin CORS; mutaciones futuras `same-origin` desde navegador |
| CART-26 | Cookie robada sigue siendo credencial válida aun si está firmada | Secuestro de sesión hasta expiración | **Puede corregirse después** mediante detección/rotación; atributos son bloqueantes | HttpOnly/Secure/SameSite, TTL, rotación y revocación cuando exista almacenamiento |
| CART-27 | Sin cache headers estructurados en respuestas de carrito | Riesgo de cache compartida si se introduce proxy/API | **Bloqueante antes de integrar Next** | `Cache-Control: no-store, private`, nunca cachear por sesión |
| CART-28 | No hay snapshot/lock de stock al cotizar | Cotización puede contener cantidades superiores al stock actual | **Debe corregirse antes del cutover** | Revalidar y devolver conflictos visibles antes de crear quote |
| CART-29 | No hay métricas/correlation ID específicos | Diagnóstico difícil entre Next, edge y Go | **Puede corregirse después**, pero logging mínimo antes de cutover | ID de request no sensible; logs sin cookie completa |
| CART-30 | No hay pruebas del carrito | Los problemas anteriores no tienen red de regresión | **Bloqueante antes de integrar Next** | Pruebas de §13 por cada subfase |

### 3.1 Método HTTP, CORS y errores

- Los métodos actuales son semánticamente razonables para HTMX, pero `PUT/PATCH/DELETE` no son progresivos con formularios HTML nativos.
- No se encontró `Access-Control-Allow-*`; no se debe añadir CORS si la topología es same-origin.
- Los endpoints JSON propuestos deben rechazar media types no soportados con 415, métodos no registrados con 405, cuerpo mal formado con 400 y fallos internos con 503/500 controlado.
- Nunca se deben devolver SQL, stack traces, `DATABASE_URL`, `GO_API_BASE_URL`, nombres de host internos ni el valor de `cart_id`.
- Los logs pueden contener el error interno, pero deben identificar la sesión con un hash truncado no reversible, nunca con la cookie completa.

---

## 4. Opciones de arquitectura

| Criterio | A. Formularios Next → Go same-origin | B. Route Handlers Next → Go | C. JSON Go consumido directo | D. Next sin carrito |
|---|---|---|---|---|
| Cookie | Viaja directo a Go | Next debe reenviar `Cookie` y devolver cada `Set-Cookie` | Viaja directo a Go | No se usa desde Next |
| Primera parte | Sí, si edge comparte esquema/host/puerto | Sí hacia navegador; salto interno adicional | Sí, si edge comparte esquema/host/puerto | Irrelevante para Next |
| Topología | Requiere rutas exactas Go bajo origen público | Puede ocultar Go detrás de Next, pero exige conectividad interna | Requiere rutas `/api/cart*` a Go bajo origen público | No requiere cambio |
| CSRF | Origin/Referer + SameSite; token si Go puede emitirlo al formulario | Debe validarse en Next y/o Go; fácil crear una falsa confianza | Origin/Referer + SameSite; token para fetch opcional | No amplía superficie |
| CORS | No | No en navegador; tampoco debe abrirse Go | No | No |
| Progressive enhancement | Excelente para add/update/remove con POST/303 | Posible, pero Next duplica transporte y redirects | JSON por sí solo no funciona sin JS | Conserva enlaces actuales |
| Sin JavaScript | Sí | Sí, si Route Handler soporta form y 303 | No por sí solo; necesita A | Sí, sin carrito Next |
| Reglas de negocio | Solo Go | Go sigue mandando, pero Next puede duplicar validación/errores | Solo Go | No nuevas |
| Observabilidad | Un salto público | Dos saltos y dos logs | Un salto público | Simple |
| Errores | HTML/redirect, claros para fallback | Traducción de errores obligatoria | JSON estable | Sin errores de carrito Next |
| Seguridad | Buena tras resolver sesión/origen | Más superficie de cookie y cabeceras | Buena tras resolver sesión/origen | Menor superficie |
| Complejidad | Media | Alta | Media | Baja |
| Rollback | Quitar forms/enlaces Next; Go legado sigue | Quitar Route Handlers y UI | Quitar fetch/UI; Go legado sigue | Ya es el estado actual |
| HTMX | Puede coexistir con rutas actuales | Debe transportar fragmentos o usar JSON separado | Endpoints nuevos no rompen HTMX | HTMX sigue solo en Go |
| Cotización | Cookie llega naturalmente a Go | Depende de reenvío correcto | Cookie compartida conserva vínculo | Se enlaza a Go como hoy |

### 4.1 Recomendación respaldada por el código

**Principal: C + fallback A obligatorio.**

1. El navegador carga `/catalogo` desde Next; el catálogo SSR mantiene su fetch read-only server-to-server con `credentials: "omit"` (`frontend/lib/api/catalog-browse.ts:105-136`, `288-342`).
2. Sin JavaScript, `AddToCartForm` envía `POST` a una ruta HTML exacta de Go. Go valida sesión, origen, producto y stock, muta y responde `303` hacia un `return_to` relativo validado.
3. Con JavaScript, un componente pequeño intercepta el mismo intento y usa `/api/cart*` same-origin con `credentials: "same-origin"`. Go devuelve el estado canónico.
4. Go sigue sirviendo `/carrito` y `/solicitar-cotizacion` durante la coexistencia. No se eliminan HTMX ni templates.
5. No habrá proxy catch-all `/api/*`, CORS ni reglas de stock en Next.

Esta combinación corrige la principal limitación de C —sin JavaScript— sin introducir el reenvío de cookies de B. También permite desactivar la mejora JS sin perder el formulario.

**D es el estado seguro hasta aprobación:** el botón no debe aparecer mientras CART-01 a CART-08, CART-14, CART-20, CART-27 y CART-30 sigan abiertos.

**B solo como contingencia:** si la plataforma no puede enrutar rutas exactas de Go bajo el mismo origen, Route Handlers específicos podrían transportar cookies. Antes habría que diseñar y probar `Cookie`, múltiples `Set-Cookie`, `Origin`, timeouts, body limits, redirects y errores. No debe existir proxy global.

### 4.2 Dato topológico exacto que falta

Se necesita una decisión escrita con:

- URL pública canónica: esquema, host y puerto.
- Si Next y Go se publican en el mismo host o en subdominios distintos.
- Proceso que termina TLS y si producción obliga HTTPS.
- Mapeo exacto de `/catalogo`, `/api/cart*`, rutas HTML de carrito, `/carrito`, `/solicitar-cotizacion`, wizard y assets.
- Puertos internos de Next y Go y conectividad entre procesos.
- Cabeceras `Forwarded`/`X-Forwarded-*` emitidas por el proxy y lista de proxies confiables.
- Estrategia de despliegue/rollback y health checks por proceso.
- Dominio efectivo de cookie; se recomienda host-only, no `.dominio` compartido.

La copia solo prueba que Go exige `PORT` (`main.go:20-34`), que Next usa `GO_API_BASE_URL` privada (`frontend/lib/env.ts`) y que el ejemplo local apunta a `http://127.0.0.1:8080`. `frontend/next.config.ts` no contiene rewrites. No hay Docker, Compose, Nginx, Caddy, Apache, Traefik, Vercel, Railway, Render, Fly.io ni Kubernetes en el repositorio.

---

## 5. Contrato público propuesto

### 5.1 Convenciones comunes

- Rutas propuestas: `/api/cart` y `/api/cart/items`. No chocan con rutas existentes.
- Debido a la clave real, el path debe usar `{product_id}`, no `{item_id}`. Introducir un `item_id` implicaría inventar identidad o migrar el esquema.
- JSON: `Content-Type: application/json; charset=utf-8`, `X-Content-Type-Options: nosniff`, `Cache-Control: no-store, private`.
- Requests JSON mutables: `Content-Type: application/json`, cuerpo limitado, una sola entidad JSON, campos desconocidos rechazados.
- Cookie: credencial HttpOnly administrada por Go; no aparece en el JSON.
- CSRF: validación estricta de `Origin`/`Referer` según §8 en cada método inseguro. `SameSite` es defensa adicional.
- No se reenvían cookies a servicios terceros ni se aceptan credenciales cross-origin.
- Error común:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "No se pudo procesar la solicitud",
    "fields": {
      "quantity": "Debe ser un entero positivo"
    }
  }
}
```

`fields` solo aparece para errores de entrada. El mensaje no contiene detalles internos.

### 5.2 Representación canónica

```json
{
  "cart": {
    "items": [
      {
        "product_id": "00000000-0000-0000-0000-000000000000",
        "name": "Producto",
        "slug": "producto",
        "image_filename": "imagen.jpg",
        "quantity": 1,
        "max_quantity": 4,
        "available": true
      }
    ],
    "total_items": 1
  }
}
```

`total_items` significa **número de líneas distintas**, equivalente al `len(items)` que usa la UI actual, no la suma de unidades. El ID del carrito se excluye deliberadamente: es una credencial interna transportada por cookie y no tiene uso legítimo en la UI.

### 5.3 `GET /api/cart`

| Aspecto | Diseño |
|---|---|
| Request | Sin body; `Accept: application/json` |
| Éxito | 200 con representación canónica; carrito vacío usa `items: []`, `total_items: 0` |
| Carrito inexistente | Go crea una sesión nueva confiable, emite cookie y devuelve carrito vacío |
| Cookie inválida/firma inválida | Descartar, rotar, crear carrito vacío; nunca consultar el ID recibido |
| DB | `carts` + `cart_items` + proyección pública de `products/categories/images`; revalida disponibilidad y stock |
| CSRF | No aplica por ser lectura; sí `no-store` |
| Errores | 406 opcional si no acepta JSON; 503 `cart_unavailable` para DB |
| Idempotencia | La lectura es idempotente; la creación de sesión es efecto lateral controlado |

No debe devolver `customer_*`, `created_at`, `is_submitted`, `source`, category interna, galería, descripciones largas ni el `cart_id`.

### 5.4 `POST /api/cart/items`

Request:

```json
{ "product_id": "00000000-0000-0000-0000-000000000000", "quantity": 1 }
```

| Aspecto | Diseño |
|---|---|
| Semántica | Añadir `quantity` unidades al valor actual, preservando el comportamiento de duplicados actual |
| Validación | UUID canónico; `quantity` entero en rango aprobado, inicialmente 1; producto existente, disponible y stock positivo |
| DB | Operación atómica en Go/DB; `source` lo fija el servidor a catálogo |
| Éxito | 201 si crea línea, 200 si incrementa; ambos devuelven representación canónica |
| Stock insuficiente | 409 `stock_insufficient`, puede incluir `max_quantity` seguro; no muta |
| No disponible | 409 `product_unavailable`; no muta |
| No existe | 404 `product_not_found`; no muta |
| Carrito inexistente | Crear sesión/carrito confiable y ejecutar una sola vez |
| Cookie | `Set-Cookie` solo si se crea/rota sesión |
| CSRF | Origin/Referer estricto; credenciales same-origin |
| Idempotencia | POST aditivo no es idempotente. Antes del cutover necesita nonce/idempotency key para doble envío o una decisión aprobada de cambiar a “asegurar una línea” |

La semántica de reintento debe cerrarse antes de 5B7B5. Si se preserva incremento, el servidor debe recordar temporalmente una clave de operación por carrito; deshabilitar un botón no basta.

### 5.5 `PATCH /api/cart/items/{product_id}`

Request:

```json
{ "quantity": 2 }
```

| Aspecto | Diseño |
|---|---|
| Semántica | Fijar cantidad absoluta; no admite `action` |
| Validación | UUID canónico; entero `>=1`; límite de negocio y stock vigente |
| Éxito | 200 con carrito canónico |
| Línea inexistente | 404 `cart_item_not_found` |
| Cero/negativo/texto | 400/422 `invalid_quantity`; no elimina |
| Exceso | 409 `stock_insufficient`; no clampa silenciosamente |
| Producto agotado/no disponible | 409 con código estable; la UI ofrece eliminar |
| Idempotencia | Sí para la misma cantidad |
| Cookie/CSRF | Cookie confiable; Origin/Referer estricto |

### 5.6 `DELETE /api/cart/items/{product_id}`

- Request sin body.
- Valida UUID canónico y sesión.
- Responde 204 tanto si la línea existía como si ya no existe; idempotente.
- No borra el producto ni el carrito.
- Origin/Referer estricto y cookie same-origin.
- Error DB: 503 `cart_unavailable`; sin cuerpo en éxito.

### 5.7 `DELETE /api/cart`

- Vacía `cart_items` del carrito confiable; conserva la sesión para futuras acciones.
- 204 aunque ya esté vacío; idempotente.
- No elimina `carts` mientras una quote pueda referenciarlo.
- Origin/Referer estricto; error DB controlado.

### 5.8 Códigos comunes

| Status | Código | Uso |
|---|---|---|
| 200 | — | Lectura/actualización exitosa |
| 201 | — | Primera creación de línea |
| 204 | — | Eliminación/vaciado exitoso |
| 400 | `invalid_request`, `invalid_json` | Sintaxis/campos/UUID/cantidad inválidos |
| 403 | `csrf_rejected` | Origin/Referer/token inválido o ausente |
| 404 | `product_not_found`, `cart_item_not_found` | Recurso público no existe |
| 409 | `product_unavailable`, `stock_insufficient` | Regla de negocio vigente |
| 413 | `request_too_large` | Límite de body |
| 415 | `unsupported_media_type` | No es JSON en API |
| 429 | `too_many_requests` | Rate limit |
| 500/503 | `cart_unavailable` | Error interno/DB sin detalles |

---

## 6. Modelo público de ítem

| Campo candidato | Fuente real | Tipo/nulabilidad | Uso UI | Riesgo | ¿Recalcular? | Decisión |
|---|---|---|---|---|---|---|
| `id` | No existe en `cart_items` | — | Ninguno | Inventaría identidad y ocultaría la clave compuesta | — | **Excluir** |
| `product_id` | `cart_items.product_id` | UUID string, no nulo | key, rutas de mutación | No usar como autorización por sí solo | No, pero validar | **Incluir** |
| `name` | `products.name` vía vista/join | string, no nulo | Nombre visible | Contenido administrado; React/templ deben escapar | Sí, para reflejar catálogo | **Incluir** |
| `slug` | `products.slug` | string, no nulo | Enlace al detalle | Debe codificarse en URL | Sí | **Incluir** |
| `image_filename` | imagen principal / `catalog_products.image_url` | string o `null` | Proxy de imagen Next | Debe normalizar vacío y validar filename en proxy | Sí | **Incluir** |
| `quantity` | `cart_items.quantity` | int, no nulo | Control y resumen | Puede quedar sobre stock si baja | Leer y validar en cada operación | **Incluir** |
| `max_quantity` | `products.quantity` actual | int, no nulo | Límite/estado UI | Es informativo; no confiar en cliente | **Sí, cada lectura** | **Incluir** |
| `available` | `products.available && products.quantity > 0` | bool, no nulo | Deshabilitar/explicar | Puede cambiar después de render | **Sí, cada lectura** | **Incluir** |

No se incluyen precio, datos del cliente, `source`, timestamps, galería, descripción larga, QR ni campos administrativos.

---

## 7. Cookie y sesión recomendadas

### 7.1 Estado recomendado

| Propiedad | Recomendación |
|---|---|
| Propietario | Go exclusivamente |
| Nombre | Conservar `cart_id` para compatibilidad, salvo decisión explícita de migración |
| Identificador DB | UUIDv7 generado server-side |
| Autenticidad | Cookie versionada y autenticada, por ejemplo payload UUID + HMAC-SHA-256 con clave separada; no aceptar UUID plano |
| Codificación | Base64url sin padding o formato seguro de `net/http`; versión incluida para rotar |
| Entropía | UUIDv7 generado por biblioteca segura más MAC; la firma elimina la capacidad de elegir un UUID ajeno |
| `HttpOnly` | `true` |
| `Secure` | `true` en todo entorno HTTPS; `false` solo en desarrollo HTTP explícito |
| `SameSite` | `Lax` explícito; evaluar `Strict` solo si no rompe navegación/cotización |
| `Path` | `/` mientras `/catalogo`, `/carrito` y `/solicitar-cotizacion` la compartan |
| `Domain` | Omitido: host-only |
| `MaxAge` | 30 días, alineado con `Expires` |
| `Expires` | 30 días desde emisión/rotación; decidir si es fija o deslizante |
| Valor inválido/firma inválida | Ignorar sin consultar DB, expirar valor viejo y emitir sesión nueva |
| Carrito inexistente | Crear con ID server-side confiable; no conservar un ID elegido por cliente |
| Rotación de claves | Key ring con clave activa y anterior; nombre/config exactos pendientes; no reutilizar secreto JWT |
| Logs | Nunca registrar cookie completa; usar request ID y huella truncada |

Una firma impide fijar un UUID conocido, pero una cookie completa robada continúa siendo una credencial bearer. `HttpOnly`, `Secure`, SameSite, expiración y rotación reducen ese riesgo; revocación inmediata requeriría estado adicional.

### 7.2 Compatibilidad Next/Go

- El navegador envía la cookie directamente a las rutas Go same-origin.
- Next no usa `NEXT_PUBLIC_*`, no lee la cookie en Client Components y no conoce la clave de firma.
- Los fetches mutables del navegador usan `credentials: "same-origin"`.
- Los fetches SSR read-only actuales del catálogo siguen con `credentials: "omit"`; no se convierten en fetches de carrito.
- Si en el futuro Next hace SSR del carrito, debe existir un diseño separado para reenviar solo la cookie necesaria y transportar `Set-Cookie`; no se asume en esta fase.

### 7.3 Proxy, local y producción

- Desarrollo con `localhost:3000` y `127.0.0.1:8080` son orígenes distintos. Se necesita un entrypoint local same-origin o se debe elegir B de forma explícita.
- Producción debe forzar HTTPS antes de `Secure=true` y definir el proxy confiable. Go no debe confiar ciegamente en `X-Forwarded-Proto` de clientes directos.
- La política de `Secure` debe venir de configuración validada al iniciar, no de heurísticas ambiguas.
- No se recomienda `Domain=.example.com`: ampliaría la cookie a subdominios no necesarios.

---

## 8. Estrategia CSRF recomendada

### 8.1 Comparación

| Defensa | Ventajas | Límites en esta arquitectura | Decisión |
|---|---|---|---|
| `SameSite=Lax` | Simple, compatible con formularios y navegación | No cubre todos los escenarios same-site/subdominio; depende de navegador | Obligatoria como defensa adicional |
| `Origin` estricto | Compatible con fetch y POST HTML; no requiere JS/token bootstrap | Requiere origen canónico y política de proxy; algunos clientes pueden omitirlo | **Defensa primaria recomendada** |
| `Referer` | Fallback para forms cuando falta Origin | Puede omitirse por privacidad; contiene URL | Fallback exacto, validar solo origen |
| Token sincronizado | Fuerte, ligado a sesión | Next no puede renderizar un token de Go en la primera página sin bootstrap o reenvío | Defensa adicional cuando se diseñe bootstrap |
| Double-submit cookie | Stateless y accesible a JS | Requiere cookie no HttpOnly, comparación robusta y coordinación Next/Go | No preferida |
| Token por formulario | Bueno para no-JS y puede mitigar replay si es de un uso | Solo Go puede emitirlo; un form renderizado por Next necesita canal previo y estado de consumo | Reservado para acciones sensibles/cotización si se resuelve bootstrap |

### 8.2 Política propuesta para 5B7B3

1. Configurar una allowlist exacta de orígenes públicos; no aceptar comodines ni derivarla sin validar de `Host`.
2. En `POST`, `PUT`, `PATCH` y `DELETE`, exigir `Origin` igual al origen canónico.
3. Si un POST HTML legítimo no incluye `Origin`, validar el origen de `Referer`. Si faltan ambos, responder 403; documentar y probar compatibilidad de navegadores objetivo.
4. Verificar esta política **antes** de parsear un body grande o ejecutar DB.
5. Mantener `SameSite=Lax`, `HttpOnly`, `Secure` y ausencia de CORS.
6. Responder JSON `csrf_rejected` en APIs y una página/mensaje genérico en formularios; nunca hacer redirect de éxito.
7. No registrar Origin completo si contiene datos inesperados; normalizar esquema/host/puerto.

Esta estrategia funciona en Go, fetch, formularios HTML, same-origin, sin JavaScript y detrás de proxy, siempre que la topología provea el origen canónico. Por eso la topología es un bloqueante.

### 8.3 Token futuro opcional

Si se exige token además de Origin, Go debe generarlo y validarlo, vinculado criptográficamente a la sesión autenticada, acción y expiración. Para fetch se entregaría en una respuesta de lectura same-origin y viajaría en `X-CSRF-Token`; para HTML debe existir una forma aprobada de incluirlo en el markup Next sin compartir la clave de Go. Rotaría con `cart_id` y expiraría como máximo con la sesión. No se implementará hasta resolver ese bootstrap.

### 8.4 Pruebas CSRF mínimas

- Origin correcto, incorrecto, `null`, malformado y con subdominio parecido.
- Puerto/esquema distintos.
- Referer correcto/incorrecto cuando Origin falta.
- Ambos ausentes.
- Métodos seguros no bloqueados.
- Proxy confiable/no confiable.
- SameSite, Secure y ausencia de CORS en navegador real.
- Si hay token: válido, inválido, expirado, de otra sesión, replay y rotación.

---

## 9. Progressive enhancement

### 9.1 Base sin JavaScript

Rutas HTML futuras, separadas de la API JSON:

| Acción | Form nativo propuesto | Respuesta |
|---|---|---|
| Añadir | `POST /carrito/items` con `product_id`, `quantity=1`, `return_to` | 303 a URL validada con estado accesible |
| Actualizar | `POST /carrito/items/{product_id}/cantidad` con cantidad absoluta | 303 a `/carrito` o quote |
| Eliminar | `POST /carrito/items/{product_id}/eliminar` | 303 |
| Vaciar | `POST /carrito/vaciar` | 303 |
| Cotizar | `POST /solicitar-cotizacion` con `action`/`method` reales | 303 a confirmación; 4xx vuelve a página completa con errores |

Los métodos command POST son necesarios porque HTML nativo no emite PATCH/DELETE. Las rutas HTMX actuales se conservan durante la transición.

- `return_to` debe ser una ruta relativa permitida; rechazar esquema, host, `//`, backslash, NUL y destinos fuera de catálogo/carrito/cotización.
- Debe conservar `buscar`, `categoria`, `pagina` y `por_pagina` del catálogo sin confiar en un URL absoluto.
- El mensaje de éxito/error debe ser server-rendered y anunciado con `role="status"`/`aria-live`; decidir entre flash server-side o código de query allowlisted. No reflejar mensajes arbitrarios.
- PRG con 303 evita reenvío por reload, pero no sustituye idempotencia ante doble click/red.
- La navegación a `/carrito` y `/solicitar-cotizacion` sigue siendo enlace real.
- La cotización debe poder editar/eliminar ítems y enviar el formulario sin HTMX.

### 9.2 Mejora JavaScript posterior

- Solo `AddToCartForm`, contador y controles interactivos mínimos son Client Components.
- Estado pending deshabilita el control y usa `aria-busy`; idempotency key protege el servidor.
- Tras éxito se usa la respuesta canónica de Go para contador y mensajes; no se incrementa optimistamente sin reconciliar.
- Mensajes en región `aria-live`, foco preservado y errores asociados al control.
- En 409 se muestra stock/disponibilidad vigente y se revalida la vista.
- Backend apagado conserva el formulario/navegación o muestra error recuperable; nunca borra estado local canónico.
- No se convierte `frontend/app/(site)/catalogo/page.tsx` en Client Component.

---

## 10. Límite de componentes futuros

| Componente | Tipo inicial | Props mínimas | Fuente | ¿JS? | Fallback | Error/accesibilidad |
|---|---|---|---|---|---|---|
| `AddToCartForm` | Server | `productId`, `available`, `returnTo`, nombre accesible | Producto SSR | No en base; Client child opcional | Form POST/303 | Disabled real si no disponible; estado asociado |
| `CartCount` | Server si existe lectura SSR segura; de otro modo Client enhancement | conteo inicial opcional | GET cart canónico | Opcional | Enlace “Carrito” sin número | `aria-label` incluye conteo; no anunciar cada hydration |
| `CartDrawer` o página | **Decisión pendiente; página primero recomendada** | carrito canónico | Go | Drawer sí; página no necesariamente | Página `/carrito` Go | Trap/restauración de foco solo si drawer; escape y scroll |
| `CartItem` | Server/presentacional | item público | Parent | No | HTML completo | Imagen con alt, nombre/enlace semántico |
| `CartQuantityForm` | Server + enhancement pequeño | `productId`, quantity, max, available | Item canónico | Opcional | POST/303 | Label, input numérico, error ligado, targets ≥44px |
| `CartRemoveForm` | Server + enhancement pequeño | `productId`, nombre | Item canónico | Opcional | POST/303 | Confirmación no solo JS; nombre en label |
| `CartStatusMessage` | Server | código allowlisted/mensaje controlado | Respuesta/redirect | No | Visible | `role=status` o `alert` según severidad |

No se reutilizan componentes HTMX dentro de React. Se conserva su flujo como legado/rollback hasta que una experiencia equivalente esté validada.

---

## 11. Relación con `/solicitar-cotizacion`

La sesión firmada debe llegar sin traducción desde `/catalogo` Next a `/solicitar-cotizacion` Go. La integración no puede cambiar `quote.CartID` ni el panel en las primeras subfases.

Antes de 5B7B9 se debe decidir:

1. Si la quote captura un snapshot inmutable de ítems o conserva el join vivo actual.
2. Si el stock insuficiente bloquea el envío, reduce cantidades con confirmación o permite una solicitud informativa. No se recomienda clamp silencioso.
3. Si el carrito se conserva, limpia, bloquea o rota después de éxito.
4. Cómo impedir doble envío y notificaciones duplicadas.
5. Cómo renderizar errores y éxito con HTML completo y 303 sin romper HTMX.

Hasta cerrar esas decisiones, `/solicitar-cotizacion` y el panel permanecen en Go sin cambios.

---

## 12. Fases futuras

Los archivos listados son el alcance previsto; cada fase debe revalidar el árbol antes de editar. Ninguno se crea o modifica en 5B7B1.

### 5B7B2 — Cookie y sesión bloqueantes

- **Objetivo:** resolver autenticidad, primera solicitud, validación, expiración y rotación de `cart_id` sin cambiar UI.
- **Archivos previstos:** crear `internal/routes/cart_session.go`, `internal/routes/cart_session_test.go`; modificar `internal/routes/routes.go`, `internal/db/cart.go`; documentar configuración en un ejemplo de entorno solo si se aprueba su ubicación.
- **Go:** resolver/firma/rotación central, contexto tipado, cookie segura; aplicar a carrito, quote y wizard dependientes.
- **Next:** ninguno.
- **Riesgos:** perder carritos existentes al dejar de aceptar UUID plano; secreto mal gestionado; Secure en HTTP local.
- **Pruebas:** cookie ausente, válida, UUID desnudo legacy, firma inválida, vacía, ajena, expiración, primera request, key rotation.
- **Cierre:** ningún ID del cliente llega a DB sin autenticidad; flujo legado sigue funcionando.
- **Dependencias:** topología, HTTPS, estrategia de compatibilidad/rotación de cookies existentes.

### 5B7B3 — CSRF/Origin

- **Objetivo:** proteger todas las mutaciones de carrito, quote y wizard antes de exponer Next.
- **Archivos previstos:** crear `internal/routes/cart_security.go`, `internal/routes/cart_security_test.go`; modificar registros en `internal/routes/cart.go`, `internal/routes/contact_requests.go`, `internal/routes/wizard.go` o el punto común aprobado.
- **Go:** allowlist exacta, Origin/Referer, errores diferenciados HTML/JSON, body antes de DB.
- **Next:** ninguno salvo configuración documental de origen si se aprueba.
- **Riesgos:** bloquear navegadores legítimos o confiar en forwarded headers falsos.
- **Pruebas:** matriz de §8.4 y ausencia de CORS.
- **Cierre:** toda mutación rechaza cross-origin antes de ejecutar lógica.
- **Dependencias:** 5B7B2 y origen canónico confirmado.

### 5B7B4 — JSON read-only

- **Objetivo:** exponer `GET /api/cart` mínimo y seguro.
- **Archivos previstos:** crear `internal/routes/public_cart_api.go`, `internal/routes/public_cart_api_test.go`; modificar `internal/routes/routes.go`, `internal/db/cart.go` y, si se separa proyección, crear `internal/db/public_cart.go` con pruebas.
- **Go:** DTO público, lectura/revalidación, headers no-store, errores seguros.
- **Next:** ninguno.
- **Riesgos:** exponer credenciales/campos internos, N+1, crear carritos por crawlers.
- **Pruebas:** sesión, vacío, ítems, stock cambiado, media null, DB error, headers, métodos.
- **Cierre:** contrato §5.3 estable, sin `cart.id` ni customer/source/timestamps.
- **Dependencias:** 5B7B2; 5B7B3 si el GET emite material CSRF.

### 5B7B5 — Mutaciones JSON

- **Objetivo:** implementar POST/PATCH/DELETE atómicos con stock vigente e idempotencia aprobada.
- **Archivos previstos:** modificar `internal/routes/public_cart_api.go`, sus pruebas, `internal/db/cart.go`/`public_cart.go`; migración solo si la solución aprobada de idempotencia la exige.
- **Go:** parsing estricto, transacciones, source server-side, errores 400/404/409/415/429/503.
- **Next:** ninguno.
- **Riesgos:** deadlocks, lost updates, cambio semántico en duplicados.
- **Pruebas:** mutaciones, carrera, replay/idempotencia, límites, producto agotado.
- **Cierre:** ninguna regla de stock vive fuera de Go; pruebas concurrentes pasan.
- **Dependencias:** 5B7B2–B4 y decisión de duplicados/idempotencia.

### 5B7B6 — Forms sin JavaScript y redirects

- **Objetivo:** rutas HTML POST/303 equivalentes, manteniendo HTMX.
- **Archivos previstos:** modificar `internal/routes/cart.go`, `internal/routes/contact_requests.go`, `internal/templates/components/cart.templ`, `internal/templates/pages/quote_request.templ`; pruebas nuevas/extendidas en `internal/routes/cart_test.go` y `contact_requests_test.go`.
- **Go:** command routes POST, `return_to` allowlisted, PRG, mensajes controlados.
- **Next:** ninguno todavía.
- **Riesgos:** open redirect, duplicar handlers, romper fragments HTMX.
- **Pruebas:** JS off, redirects 303, filtros preservados, destino malicioso rechazado.
- **Cierre:** add/update/delete/clear/quote funcionan con forms reales sin eliminar rutas legacy.
- **Dependencias:** 5B7B2–B5.

### 5B7B7 — `AddToCartForm` en catálogo

- **Objetivo:** añadir el formulario server-rendered a tarjetas Next y una mejora JS pequeña opcional.
- **Archivos previstos:** crear `frontend/components/catalog/add-to-cart-form.tsx`, posiblemente `add-to-cart-enhancement.tsx` y `cart-status-message.tsx`; modificar `frontend/components/catalog/catalog-product-card.tsx`, `frontend/lib/types.ts` solo si el contrato lo requiere; pruebas frontend correspondientes.
- **Go:** ninguno salvo corrección objetiva de contrato.
- **Next:** form POST/303, estado pending, fetch same-origin, mensaje accesible.
- **Riesgos:** doble submit, hydration, anidar controles en links, regresión responsive.
- **Pruebas:** disponible/agotado, JS on/off, teclado, 360–1440 px, backend apagado, 409.
- **Cierre:** formulario funciona sin JS; JS no duplica reglas; catálogo sigue SSR.
- **Dependencias:** 5B7B2–B6 y edge same-origin.

### 5B7B8 — Experiencia de carrito

- **Objetivo:** decidir y entregar página primero o drawer accesible sin retirar `/carrito` Go prematuramente.
- **Archivos previstos si se aprueba página Next:** `frontend/app/(site)/carrito/page.tsx`, componentes `cart-item.tsx`, `cart-quantity-form.tsx`, `cart-remove-form.tsx`, `cart-status-message.tsx`; header solo si se aprueba contador. Si se conserva Go, limitar cambios a enlaces/contrato.
- **Go:** conserva fuente de verdad y fallback; ningún panel cambia.
- **Next:** representación y forms; drawer solo tras prueba de foco.
- **Riesgos:** SSR de cookie, Set-Cookie, focus trap, duplicar estado.
- **Pruebas:** vacío, reload, pestañas, JS off, foco, móvil, concurrencia.
- **Cierre:** experiencia elegida es accesible, reversible y usa contrato canónico.
- **Dependencias:** decisión página/drawer/contador y transporte de cookie en SSR.

### 5B7B9 — Cotización

- **Objetivo:** preservar y endurecer el puente carrito→quote con semántica histórica aprobada.
- **Archivos previstos:** modificar `internal/routes/contact_requests.go`, `internal/db/quotes.go`, `internal/templates/pages/quote_request.templ`, pruebas; migración nueva solo si se aprueba snapshot/idempotencia persistente.
- **Go:** transacción de validación/quote, idempotencia, POST/303, política post-éxito.
- **Next:** solo enlaces/estado si son necesarios; la lógica sigue en Go.
- **Riesgos:** alterar panel/historial, mensajes WhatsApp duplicados, stock cambiante.
- **Pruebas:** quote con/sin ítems, stock cambia, doble submit, panel, notificación fake.
- **Cierre:** quote estable, una sola persistencia, política de carrito posterior verificable.
- **Dependencias:** decisión snapshot, stock y expiración; 5B7B2–B8.

### 5B7B10 — QA, seguridad y cutover

- **Objetivo:** regresión integral, observabilidad, rollback ensayado y decisión de tráfico.
- **Archivos previstos:** principalmente pruebas/documentación/config de despliegue ya aprobada; no se asume proveedor.
- **Go/Next:** solo correcciones mínimas respaldadas por regresión.
- **Riesgos:** diferencias de proxy/HTTPS frente a local, cookies legacy.
- **Pruebas:** §13 completo, navegador real, carga/concurrencia, seguridad y rollback.
- **Cierre:** criterios de §14 cumplidos, runbook aprobado y Go legacy conservado hasta estabilización.
- **Dependencias:** todas las subfases y entorno seguro no productivo.

---

## 13. Plan de pruebas

### 13.1 Unitarias

- Cookie ausente, válida, vacía, UUID inválido, UUID desnudo, firma inválida, expirada, key anterior y cookie de otro carrito.
- Primera request dispone de la sesión emitida.
- Carrito inexistente y vacío.
- Producto inexistente, UUID manipulado, no disponible y stock cero.
- Cantidad válida, cero, negativa, texto, decimal, excesiva y límite entero.
- Item inexistente y product ID de otra línea.
- Error DB en lectura, inserción, actualización, eliminación y commit.
- Errores JSON/HTML seguros sin SQL, stack, host ni secrets.
- CSRF/Origin/Referer según §8.4.
- Content-Type válido/inválido, body grande, JSON doble y campos desconocidos.
- Métodos permitidos y 405.
- `return_to` relativo válido y open redirects rechazados.

### 13.2 Integración con PostgreSQL aislado

- Añadir una línea y añadir repetido.
- Actualizar cantidad absoluta; eliminar; eliminar repetido; vaciar repetido.
- Dos incrementos simultáneos; actualización simultánea con baja de stock.
- Producto desactivado/eliminado entre lectura y mutación.
- Cookie compartida entre `/catalogo` Next, API Go, `/carrito` y quote.
- Reload y múltiples pestañas.
- Backend apagado/timeout sin perder fallback HTML.
- Sin JavaScript: add, carrito, actualización, eliminación y quote.
- Cotización: vínculo, idempotencia, política de snapshot y estado post-éxito.

### 13.3 Seguridad

- Fijación con UUID conocido y cookie firmada de otra sesión.
- Cookie robada: documentar que sigue siendo bearer hasta revocación/expiración.
- CSRF desde otro origen y subdominio same-site.
- IDs y quantities manipulados.
- Race/lost update y replay de idempotency key.
- Doble click/doble submit.
- Fugas en body, headers y logs.
- CORS ausente; preflight no abre credenciales.
- SameSite/HttpOnly/Secure/Path/Domain/MaxAge/Expires en navegador.
- HTTP local vs HTTPS detrás de proxy confiable.
- Rate limiting y recuperación después de ventana.

### 13.4 Accesibilidad y UX

- Teclado, focus visible, mensajes `aria-live`, labels y error associations.
- Un control no queda permanentemente disabled tras error.
- Drawer, si se aprueba: trap, Escape, restore focus y scroll lock.
- 360, 768, 1024 y 1440 px sin overflow.
- Estado agotado, stock cambiado, vacío y backend apagado legibles.
- Reduced motion si se añade animación; Framer Motion sigue fuera de esta fase.

### 13.5 Regresiones

- Home, Servicios, catálogo SSR, filtros, paginación e imágenes.
- Detalle Go y compatibilidad `/productos/{id}`/slugs.
- Sidebar HTMX actual, wizard y su finalización.
- `/solicitar-cotizacion`, contacto y WhatsApp fake.
- Panel de solicitudes, catálogo, productos, stock y wizards.
- APIs `_health`, socials, catalog listings/categories/products, media y contact requests.
- Sin CORS, sin proxy global, sin filtración de `GO_API_BASE_URL`.

---

## 14. Criterios de aceptación para considerar listo el carrito

- [ ] Go es la única fuente de verdad; Next no calcula stock, clamps ni merge de líneas.
- [ ] `cart_id` es server-side, autenticada, HttpOnly, SameSite explícita, host-only y Secure en HTTPS.
- [ ] Primera solicitud y cookie inválida tienen comportamiento determinista.
- [ ] No se puede elegir un UUID ajeno; el modelo de cookie robada está documentado y mitigado.
- [ ] Topología same-origin, dominio, TLS y proxy confiable están confirmados por configuración ejecutable.
- [ ] CSRF/Origin está resuelto para API y forms sin JavaScript.
- [ ] IDs, Content-Type, body y cantidades se validan estrictamente.
- [ ] Stock/disponibilidad se revalidan atómicamente en toda mutación y al cotizar.
- [ ] Add repetido e idempotencia tienen semántica aprobada y pruebas concurrentes.
- [ ] JSON es mínimo, no-store, seguro y estable; no expone cart ID ni modelos administrativos.
- [ ] Funciona sin JavaScript mediante POST/303 y con JavaScript mediante mejora pequeña.
- [ ] Errores son accesibles, recuperables y no filtran internos.
- [ ] Página/drawer y contador cumplen accesibilidad aprobada.
- [ ] `/solicitar-cotizacion` conserva vínculo y tiene política histórica/post-éxito aprobada.
- [ ] Go/HTMX permanece disponible para rollback durante estabilización.
- [ ] Pruebas unitarias, integración, seguridad, browser y regresión pasan.
- [ ] No hay regresiones en Home, Servicios, catálogo, detalle, wizard, panel ni APIs.

---

## 15. Rollback

1. Mantener las rutas y templates Go actuales durante todas las subfases.
2. Activar la UI Next solo después de que endpoints/forms estén desplegados y verificados.
3. El rollback inmediato consiste en ocultar/desactivar `AddToCartForm` Next y volver al estado D; no borrar datos ni migraciones.
4. Las rutas JSON nuevas son aditivas y pueden quedar inaccesibles desde UI sin afectar HTMX.
5. No retirar `/carrito`, `/solicitar-cotizacion` ni el wizard hasta un cutover aprobado y un periodo de observación.
6. Si cambia el formato de cookie, mantener ventana de lectura/rotación controlada o invalidar explícitamente con comunicación; nunca volver a confiar en UUID desnudo.
7. Ensayar rollback detrás de la topología real antes de producción.

---

## 16. Estado documental de 5B7B1

- Único artefacto previsto: `05-cart-integration.md`.
- No se iniciaron 5B7B2 ni fases posteriores.
- No se aprobó infraestructura inexistente ni sintaxis de un proveedor.
- No se cambió ningún contrato actual.
- La recomendación queda condicionada a las decisiones de §17 y a pruebas con PostgreSQL aislado y topología real.

---

## 17. Decisiones pendientes para aprobación

1. Arquitectura principal C + fallback A, con D como rollback.
2. Entry point/proxy same-origin exacto.
3. Esquema, dominio, puerto y subdominios públicos.
4. HTTPS, terminación TLS y proxies confiables.
5. Go como único propietario de `cart_id`.
6. Firma/versionado de cookie y ubicación/nombre del secreto.
7. Compatibilidad o invalidación de cookies UUID existentes.
8. TTL fijo o deslizante y rotación tras cotización.
9. Origin/Referer estricto como defensa primaria; si además se exige token y cómo hacer bootstrap desde Next.
10. Endpoints JSON y uso de `{product_id}` en lugar de `{item_id}`.
11. Semántica de añadir repetido e idempotencia.
12. Rango máximo de cantidad.
13. Resultado cuando stock baja: rechazar sin mutar, y copy aprobado.
14. Resultado cuando producto queda agotado/no disponible.
15. Página de carrito o drawer; se recomienda página primero.
16. Contador en header y si cuenta líneas o unidades; este contrato propone líneas.
17. Lectura SSR del carrito o mejora client-side; transporte de `Set-Cookie` si SSR.
18. Mensaje post-redirect: flash server-side o query allowlisted.
19. Snapshot inmutable de quote o join vivo declarado.
20. Conservar, vaciar, bloquear o rotar carrito tras cotización.
21. Limpieza de carritos abandonados y protección de carritos referenciados por quotes.
22. Rate limiting e idempotency storage.
23. Alcance del wizard y cuándo recibe las mismas protecciones.
24. Relación con el futuro detalle de producto.
25. Criterio y fecha de cutover/retirada eventual de HTMX, fuera de esta fase.
