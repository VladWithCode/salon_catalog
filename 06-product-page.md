# 06 — Página Detalle de Producto

## Estado del documento

Fase 6A. Exclusivamente documental — auditoría técnica, funcional y editorial. Ningún código Next, endpoint, proxy, migración ni cambio de carrito se implementa aquí. Este archivo resuelve identidad de ruta, contrato de datos y arquitectura *antes* de que 6B1 escriba una sola línea.

Bloqueo vigente: 5B7B7 (integración del carrito con Next) sigue detenida hasta que la suite PostgreSQL de `internal/dbtest` corra en verde contra una base `cart_integration_*` real. Este documento diseña el detalle de producto asumiendo ese bloqueo como permanente hasta nuevo aviso — ver sección L.

---

## A. Rutas actuales — inventario completo

Todas registradas en `internal/routes/routes.go:41` (`RegisterCatalogRoutes`) y `:37` (`RegisterProductsRoutes`).

| Método | Patrón | Archivo:línea | Handler | Param | Tipo esperado | Respuesta | HTMX | JS | Activa | Tests |
|---|---|---|---|---|---|---|---|---|---|---|
| GET | `/catalogo/producto/{id}` | `internal/routes/catalog.go:20` | `GetProductDetail` | `id` | UUID **o** slug (dual dispatch) | Fragmento HTMX si `HX-Request: true`; si no, página completa de catálogo con modal abierto | Sí (fragmento) | Sí (animación/apertura) | Sí | **No** |
| GET | `/productos/{id}` | `internal/routes/catalog.go:23`, comentario "Compat for old printed QR codes" | `GetProductDetail` (mismo handler) | `id` | UUID o slug, idéntico | Idéntica a la anterior | Sí | Sí | Sí | **No** |
| GET | `/catalog/products` | `internal/routes/catalog.go:19` | `GetCatalogProducts` | query `categoria`,`buscar`,`pagina`,`por_pagina` | — | Fragmento HTMX (grid) | Sí | Sí | Sí | No (cubierto indirectamente por `public_api_test.go` solo en su variante JSON) |
| GET | `/catalog/categories` | `internal/routes/catalog.go:18` | `GetCatalogCategories` | — | — | Fragmento HTMX | Sí | Sí | Sí | No |
| GET | `/static/uploads/{filename}` | `internal/routes/routes.go:52-53` | `http.FileServer` sobre `web/static/` | filename | string | Binario (imagen) | No | No | Sí | No |
| GET | `/api/products/{slug}` | `internal/routes/products.go:56` | `GetProductBySlug` | `slug` (explícito, no dual) | slug | JSON — **struct admin completo `db.Product`** | No | No | Sí, **sin auth** | No |
| GET/PUT/DELETE | `/panel/productos/*` | `internal/routes/products.go:34-49` | CRUD admin | `id` | UUID | HTML/JSON, autenticado | Sí | Sí | Sí | No |

**Redirect entre ID y slug: no existe.** `GetProductDetail` resuelve ambos formatos en el mismo handler sin redirigir nunca (`internal/db/catalog.go:126-138`, ver sección B). No hay 301/302 de UUID → slug canónico ni viceversa.

**Hallazgo de seguridad heredado, no corregido aquí**: `GET /api/products/{slug}` es pública, sin autenticación, y devuelve el struct `db.Product` completo — incluye `main_img_id`, `gallery_ids`, `qrcode_filename`, `category_id` (`internal/routes/products.go:135-137`; ya señalado en `05-catalog-page.md:386,858`). El futuro endpoint de detalle (sección M) **no debe reutilizar esta ruta ni este struct**.

---

## B. Identidad del producto

### Evidencia de esquema
`sql/migrations/20250703200655_add_products_table.sql:2-12`:
```sql
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(200) UNIQUE NOT NULL,
    ...
);
```
- `id`: UUID, PK, default server-side `gen_random_uuid()`.
- `slug`: `VARCHAR(200)`, **NOT NULL**, **UNIQUE** — garantizado a nivel de constraint de base de datos, no solo por convención.
- **No existe generación automática de slug.** El admin lo escribe a mano (`internal/routes/products.go:763`, `product.Slug = strings.TrimSpace(r.FormValue("slug"))`). No hay función `slugify` en el repo.
- **Slug vacío**: el constraint `NOT NULL` lo impide a nivel de base de datos; un `TrimSpace` sin validación adicional en el formulario admin podría, en teoría, intentar insertar cadena vacía (`""` no es NULL) — el constraint UNIQUE la aceptaría una sola vez. No hay evidencia de un `CHECK (slug <> '')`. Riesgo teórico, no confirmado como bug activo.
- **Slug duplicado**: rechazado por la base de datos (violación de UNIQUE), no hay lógica Go de reintento/sufijo.
- **Categorías no tienen slug** (`05-catalog-page.md:320`).

### Mecanismo de resolución dual (evidencia directa)
`internal/db/catalog.go:126-138`, dentro de `FindCatalogProductDetail(id string)`:
```go
if _, err := uuid.Parse(id); err == nil {
    baseQuery += " id = @id"
} else {
    baseQuery += " slug = @slug"
}
```
Esta es la función que usan **ambas** rutas (`/catalogo/producto/{id}` y `/productos/{id}`): si el segmento de ruta parsea como UUID, busca por `id`; si no, lo trata como `slug`. No hay ambigüedad de implementación — es una sola función, un solo criterio.

Existe una **segunda vía de resolución, distinta y no dual**: `db.FindProductBySlug`/`db.FindProductByID` (`internal/db/products.go:140-305`), usadas por el panel admin y por `GET /api/products/{slug}` — cada una asume su propio tipo de identificador sin fallback.

### QR: valor embebido
`internal/qrgen/qrgen.go` genera imágenes a partir de un `Value` string. Los tres sitios que lo invocan para productos (`internal/routes/products.go:236,373,625-629`) construyen:
```go
fmt.Sprintf("https://villachenacolo.com/catalogo/producto/%s", product.ID)
```
**Los códigos QR impresos codifican la URL completa con el UUID, nunca el slug.** Esto es evidencia directa, no inferencia: cualquier decisión de identidad debe mantener viva la resolución por UUID a perpetuidad, porque hay QR físicos ya impresos que Villa Chenacolo no puede reimprimir a voluntad.

### Clasificación
- **Identidad interna**: `products.id` (UUID) — clave primaria real, estable, nunca cambia.
- **Identidad pública actual**: ambas — Go acepta indistintamente UUID o slug en el mismo segmento de ruta.
- **Identidad histórica**: UUID, vía `/productos/{id}` y códigos QR impresos con dominio `villachenacolo.com` hardcodeado (dominio no confirmado como el real de producción — ver sección Q).
- **Identidad recomendada para Next**: slug como URL pública primaria (legible, ya es lo que usan las tarjetas), con UUID aceptado como alias funcional indefinido — nunca 404 en una URL con UUID válido, aunque no sea la forma "canónica".

### Opciones evaluadas

| | Estabilidad | SEO | QR | Duplicados | Rename | Enumeración | Simplicidad | Rollback | Compat Go/Next |
|---|---|---|---|---|---|---|---|---|---|
| **A. Solo slug** | Rompe si slug cambia sin redirect | Mejor (legible) | Rota — QR usa UUID, no slug | N/A (UNIQUE) | Requiere redirect no implementado | Baja (no enumerable) | Alta | Trivial | Compat total: Go ya resuelve por slug |
| **B. Solo UUID** | Máxima | Peor (URL opaca) | Compatible nativamente | N/A | No aplica | Media (UUID no es secuencial, enumeración impráctica) | Alta | Trivial | Compat total |
| **C. `/{id}/{slug}`** | Alta (id fija, slug decorativo) | Buena si Google indexa por id+slug juntos | Compatible si se genera con ambos | N/A | Slug decorativo puede cambiar sin romper nada | Baja | Media (dos segmentos, requiere validar coherencia) | Requiere generar nuevas URLs | No coincide con ninguna ruta Go existente — requeriría nueva ruta Go |
| **D. Múltiples rutas + redirect a canónica** | Alta | Buena (una canónica) | Compatible | N/A | Redirect resuelve rename | Baja | Media (lógica de redirect nueva) | Parcial — Go no redirige hoy, habría que añadirlo |

### Recomendación
**Ninguna reescritura de identidad.** El esquema real ya es exactamente la opción D sin el redirect explícito: Go acepta slug **y** UUID en el mismo patrón de ruta, hoy, sin cambios. La recomendación es:
1. Next enlaza por **slug** (ya lo hace — `catalog-product-card.tsx:41-48`).
2. El futuro endpoint Go de solo lectura (sección M) acepta **el mismo identificador dual** (`{identifier}` = UUID o slug), reutilizando literalmente el criterio de `FindCatalogProductDetail`.
3. `/productos/{id}` se preserva sin tocar, indefinidamente, por los QR impresos — no es competencia de Next.
4. No se introduce redirect 301 slug↔UUID en esta fase: no hay evidencia de que un slug haya cambiado nunca (no existe historial de slugs, no hay tabla de redirects), así que no hay problema real que resolver todavía. Se documenta como decisión pendiente (sección W) para si algún día un admin edita un slug ya público.

No se inventan constraints nuevos: la recomendación es "no romper lo que ya funciona", no una relectura ideal del modelo de datos.

---

## C. Verificación de los enlaces actuales de Next

`frontend/components/catalog/catalog-product-card.tsx:41-48`:
```tsx
<Link href={`/catalogo/producto/${encodeURIComponent(product.slug)}`} prefetch={false} ...>
```

- Usa `product.slug`, codificado con `encodeURIComponent`.
- Contra el patrón Go `GET /catalogo/producto/{id}` (`net/http.ServeMux`, sintaxis 1.22+, wildcard de **un solo segmento**, sin `...`): el router decodifica `%XX` antes de exponer el valor vía `PathValue`, así que espacios, acentos y Unicode correctamente codificados **sí resuelven** — no hay incompatibilidad ahí.
- **Riesgo real, no confirmado como bug activo**: un slug que contuviera literalmente `/` sin codificar rompería el matching de un solo segmento — pero `encodeURIComponent` codifica `/` como `%2F`, y no hay evidencia en el código de que algún slug en la base de datos contenga `/` (el formulario admin no lo prohíbe explícitamente, tampoco hay CHECK constraint). Esto se clasifica como **bloqueo teórico, no confirmado**, no como hallazgo verificado — no hay test ni caso reproducido.
- **Conclusión**: los enlaces actuales de `/catalogo` **sí funcionan** contra el router Go tal como está. No se identifica una incompatibilidad activa que bloquee 6B1.
- **Inconsistencia menor encontrada en el modal Go mismo** (no en Next): los productos relacionados dentro de `ProductModal` usan `hx-get` con `.ID` pero `hx-push-url` con `.Slug` (`catalog.templ:446` vs `:450`) — la URL visible en el navegador es por slug, pero la petición real es por ID. Funciona (dual dispatch lo tolera), pero es una inconsistencia de estilo a nivel de la implementación Go existente, documentada aquí, no corregida.

---

## D. Datos disponibles

Fuente principal para el detalle: `db.CatalogProd` (`internal/db/catalog.go:28-40`), poblada por `FindCatalogProductDetail` desde la vista `catalog_products`.

| Campo | Fuente | Tipo Go | Nulable en DB | ¿En catálogo? | ¿En modal hoy? | ¿En detalle Next? | ¿Público? | ¿En carrito? | ¿Puede desactualizarse? |
|---|---|---|---|---|---|---|---|---|---|
| `id` | `products.id` | `string` (UUID) | No | Sí | Sí | Sí | Sí | Sí (clave) | No |
| `name` | `products.name` | `string` | No | Sí | Sí | Sí | Sí | Sí (display) | No |
| `slug` | `products.slug` | `string` | No | Sí | Sí | Sí | Sí | Sí (enlace) | Si se edita en panel |
| `description` | `products.description` | `string` | No | Sí | Sí | Sí | Sí | No | Si se edita |
| `long_description` | `products.long_description` | `string` (vacío si NULL, vía COALESCE de la vista) | Sí a nivel DB, resuelto a `""` | No | Sí | Sí | Sí | No | Si se edita |
| `category_id` | `products.category` | `string` (UUID) | Sí (FK nullable) | Sí | Sí | Sí | Sí | No | Si cambia categoría |
| `category_name` | `categories.name` (join) | `string` | Depende del join | Sí | Sí | Sí | Sí | No | Si se renombra categoría |
| `available` | `products.available` | `bool` | No | Sí | Sí | Sí | Sí | Sí (bloquea agregar) | **Sí, siempre puede estar desactualizado — Go revalida en la mutación, no confiar en el valor leído** |
| `quantity` | `products.quantity` | `int` | No | No (no se ve en tarjeta) | Sí | Sí, como "stock aproximado", nunca como cantidad garantizada | Sí, con advertencia | Sí (validado atómicamente por `internal/cart.Service`, no por este campo) | **Sí, es una lectura no transaccional — el valor real se valida solo dentro de la transacción de `internal/cart`** |
| `image_url` (imagen principal) | `images.filename` vía `products.main_img` | `string` (`""` si sin imagen, vía COALESCE) | Resuelto a `""` | Sí | Sí | Sí | Sí | No | Si se reemplaza |
| `images[]` (galería) | `images_products` join | `[]string` (filenames) | Resuelto a `[]` | No | Sí | Sí | Sí | No | Si se agregan/quitan |
| `source` (cart_items.source) | No es un campo de producto — pertenece a `cart_items` | — | — | No | No | No | No | Sí, pero no es dato de producto | N/A |
| `created_at`/`updated_at` | No presentes en `catalog_products` ni en `CatalogProd` | — | — | No | No | No | No | No | N/A |
| `main_img` (UUID de imagen) | `products.main_img` | — | Sí | No | No | **No** — administrativo | No | No | — |
| `qrcode_filename` | `products.qrcode_filename` | — | Con default `''` | No | No | **No** — administrativo, ya señalado como fuga en `/api/products/{slug}` | No | No | — |
| `gallery_ids` (IDs de imagen, no filenames) | Solo en `db.Product` (admin) | — | — | No | No | **No** — administrativo | No | No | — |
| `search_vector` | `products.search_vector` | — | — | No | No | **No** — interno de búsqueda | No | No | — |

**No hay campo de precio en el sistema.** No se documenta ninguno, consistente con `05-catalog-page.md` y con el propio esquema de `products` (sin columna `price`).

---

## E. Imágenes

### Resolución actual
- Imagen principal: `COALESCE(main_img.filename, '')` vía `products.main_img → images.id` (`catalog_products` view, `sql/migrations/20250809175837_add_catalog_products_stock_column.sql:9-14`).
- Galería: `json_agg(i.filename ORDER BY i.filename)` desde `images_products` (join many-to-many), ordenado alfabéticamente por filename — **no hay orden explícito editorial**, es orden alfabético incidental.
- Producto sin imágenes: `image_url = ""`, `images = []` — ambos casos ya cubiertos por COALESCE, nunca NULL a nivel Go.
- Filename inválido / archivo inexistente: no validado por Go al leer — el `FileServer` simplemente devolverá 404 si el archivo físico no existe en `web/static/uploads/`.
- Alt text actual: `alt={ product.Name }` para imagen principal (`catalog.templ:143`), `alt={ fmt.Sprintf("Imagen de producto %s No. %d", state.Product.Name, i) }` para thumbnails de galería (línea ~392) — genérico, no describe el contenido visual, solo nombra el producto.
- Aspect ratio: `aspect-square` en tarjetas de catálogo; en el modal la imagen hero no fuerza aspect ratio fijo (`h-2/3 xl:h-full`).
- Dependencia de `/static/uploads`: total, es la única fuente física de imágenes de producto.
- **Path traversal**: no mitigado en el `FileServer` de Go directamente (comportamiento estándar de `http.FileServer`, que sí resuelve `..` de forma segura por diseño de la librería estándar de Go), **y mitigado explícitamente por el proxy Next** (`frontend/app/api/catalog/media/[filename]/route.ts:4,23-33`): patrón `^[\p{L}\p{N}._:-]+$`, rechaza `/`, `\`, NUL, `.`, `..` — el proxy ya existe y ya es seguro, confirmado por lectura completa del archivo.
- Caché: el proxy responde `Cache-Control: no-store` (línea ~76) — decisión ya tomada para el catálogo, no inventada aquí.
- Tipo MIME: proxy valida que la respuesta de Go tenga `Content-Type` que empiece con `image/`, si no, 502 (`route.ts:66-69`) — mitiga que el proxy sirva contenido no-imagen aunque el filename pase el regex.
- Layout shift: no hay `width`/`height` explícitos documentados en el templ actual; sección R (Responsive) y T (Accesibilidad) del futuro diseño Next deben fijar dimensiones para `next/image` o equivalente.

### Estrategia propuesta para Next (conceptual, no implementada)
- Next recibe `image_filename` (imagen principal) e `images: string[]` (galería, filenames) crudos desde el futuro endpoint Go — nunca una URL completa construida por Go.
- Next construye la URL pública usando el proxy **ya existente** `/api/catalog/media/{filename}` — mismo mecanismo que el catálogo, cero código proxy nuevo.
- Fallback: si `image_filename` es `""`, Next debe mostrar un placeholder — sin inventar aquí el asset final, queda como decisión pendiente de diseño visual (sección W).
- Galería: mostrar todas las `images[]`; si vacío, ocultar la sección de galería (no mostrar galería vacía con mensaje de error, es un estado normal — ver sección N).
- Imagen LCP: la imagen principal del hero es candidata a LCP — cargar con prioridad alta, sin lazy-load, siempre que se implemente (fuera de esta fase).
- Thumbnails: reutilizables desde el mismo array `images[]`, cada uno resuelto por el mismo proxy.

No se implementa nada de esto en 6A.

---

## F. Comportamiento actual del detalle Go

Componente: `templ ProductModal(state *ProductModalState)`, `internal/templates/components/catalog.templ:287-470`.

- **No existe una página de detalle standalone.** Visitar `/catalogo/producto/{slug}` sin `HX-Request` renderiza la **página completa de catálogo** (`pages.Catalog`) con el modal ya abierto encima (`internal/routes/catalog.go:130-144`, `renderCatalogWithModal`). Es decir: hoy, "la página de detalle" **es** el catálogo con un overlay. Esto es importante: no hay contenido de detalle aislado que Next pueda replicar 1:1 como "una página" — hoy es "una página + un modal".
- **Fragmento HTMX**: la misma marca exacta (`templ.Fragment("productModalContent")`) se usa para la respuesta AJAX del modal — un único componente sirve ambos casos.
- Header/hero: imagen principal a ancho de columna, sin breadcrumbs.
- Breadcrumbs: **no existen** en el modal actual.
- Nombre: `<h2>`, no `<h1>` — dentro del modal, el `<h1>` real de la página pertenece al catálogo detrás. Esto es una consideración crítica para SEO/accesibilidad de la futura página Next standalone (sección T exige un solo `<h1>` real de producto, cosa que Go hoy no ofrece de forma aislada).
- Categoría: texto plano sobre el nombre.
- Descripción: `LongDescription` si no vacío, si no `Description`, con `line-clamp-3` y expansión JS ("Leer más").
- Galería: solo visible por debajo de `xl:`, con thumbnails y panel de despliegue JS.
- Disponibilidad: ícono + texto "Disponible"/"No Disponible" — no depende solo de color.
- Cantidad: "`{quantity} pcs`" — mostrado siempre, sin distinguir "aproximado" de "exacto".
- Formulario de carrito: **formulario HTML real** (`method="post" action="/carrito/items"`), funciona sin JavaScript, se progresivamente mejora con HTMX — patrón ya migrado en fase 5B7B6/5B7B6A, documentado en `05-cart-integration.md`.
- Wizard: sin enlace directo desde el modal.
- CTA de cotización: no presente en el modal; el flujo de cotización vive en `/solicitar-cotizacion`, fuera de este componente.
- Botón volver: no hay botón "volver" explícito — el cierre es `data-close-modal`, JS-only, sin fallback de navegación nativa (ni `<a href="/catalogo">`).
- Mensajes de error: `modalState.Error` se **asigna** en el handler (`internal/routes/catalog.go:88-104`) pero **no se encontró markup en el templ que lea `state.Error`** — el cuerpo del modal sigue intentando renderizar un `CatalogProd` vacío en vez de mostrar el mensaje de error. **Esto se documenta como gap real del código Go actual**, no como algo a corregir en esta fase — es información que 6B2/6B4 deben tener en cuenta al diseñar el endpoint y los estados de error de Next (no se puede copiar este comportamiento).
- Responsive: diseño mobile-first con `xl:` como quiebre principal para el layout de dos columnas.
- Accesibilidad: iconos con texto acompañante para disponibilidad (bien); falta de `<h1>` real y de breadcrumbs (gap).
- Sin JavaScript: el formulario de agregar al carrito funciona; apertura/cierre de modal, expansión de descripción, galería y navegación entre relacionados **no** funcionan sin JS. Además, sin JS la grilla de catálogo detrás del modal está vacía (carga por `hx-trigger="load"`, `05-catalog-page.md:594`), aunque el contenido del modal mismo sí llega server-renderizado en la carga inicial.

**Distinción explícita para 6B**: la futura página Next **no debe copiar la estructura "modal sobre catálogo"** — debe ser una página aislada real, con su propio `<h1>`, breadcrumbs, y sin dependencia del grid de catálogo detrás. Esto es una mejora intencional sobre el comportamiento Go actual, no una preservación de paridad exacta.

---

## G. Reglas editoriales

Copy respaldado únicamente por lo observado en `catalog.templ` y datos reales de producto — nada inventado.

- **Eyebrow**: nombre de categoría (`category_name`), tal como ya se usa en el modal.
- **H1**: nombre del producto (`name`) — hoy es un `<h2>` en Go; en Next debe ser el `<h1>` real de la página.
- **Breadcrumbs**: no existen hoy en Go. Propuesta mínima, respaldada por la IA ya usada en `05-catalog-page.md` (Catálogo → Categoría → Producto): "Catálogo" enlazando a `/catalogo`, categoría enlazando a `/catalogo?categoria={category_id}` (mismo parámetro que ya usa el catálogo), nombre del producto sin enlace (posición actual).
- **Categoría**: texto plano, `category_name`.
- **Disponibilidad**: reutilizar el copy existente de Go — "Disponible" / "No disponible" (ver también `internal/routes/cart.go` `cartErrorMessages["product_unavailable"]`: "Ese producto ya no está disponible" — vocabulario ya aprobado para el mismo concepto).
- **CTA de consulta**: no hay copy de "consulta" existente en el modal — no se inventa uno nuevo aquí; queda pendiente (sección W) hasta que exista una decisión editorial explícita.
- **CTA de cotización**: el flujo real es "Solicitar Cotización" — mismo texto ya usado en `internal/templates/pages/cart.templ:54` ("Solicitar Cotización", enlace a `/solicitar-cotizacion`). Reutilizable tal cual.
- **Texto para producto no disponible**: "Ese producto ya no está disponible." — reutilizado literalmente de `cartErrorMessages` (`internal/routes/cart.go:34`), ya aprobado y en producción.
- **Texto para producto inexistente**: Go usa "No se encontró el producto" (`internal/routes/catalog.go`, `modalState.Error`) — reutilizable como base editorial para el 404 de Next.
- **Texto para galería vacía**: no existe copy actual (Go simplemente no renderiza sección de galería si está vacía). Se propone: sin mensaje visible, la sección de galería se omite por completo — consistente con el comportamiento Go, no es un error, es ausencia de contenido.
- **Texto de regreso al catálogo**: no existe hoy (falta de botón volver, ver sección F). Propuesta mínima sin inventar tono nuevo: "Volver al catálogo", enlazando a `/catalogo` — copy funcional, no promocional, pendiente de aprobación editorial formal si se requiere un tono de marca específico.

**No se inventa**: precio, tiempos de entrega, renta, medidas, material, capacidad, garantías, "stock en tiempo real" (el campo `quantity` existe pero se documenta explícitamente como no-transaccional, ver sección D), ni políticas comerciales. Ninguno de estos conceptos tiene respaldo en el modelo de datos actual.

---

## H. Comportamiento del carrito en esta página

### Estado inicial seguro (recomendado para 6B4-6B6)
- Página de detalle **de consulta únicamente**.
- Sin mutaciones desde Next (ni fetch a JSON API de carrito, ni componente React con estado de carrito).
- El botón "Agregar" puede enlazar o incrustar el **flujo Go existente** — es decir, la página Next puede embeber o enlazar hacia el formulario HTML progresivo ya construido en 5B7B6/5B7B6A (`POST /carrito/items`, formularios reales, funcionan sin JS), **sin duplicar ninguna regla de stock ni de idempotencia** — Go sigue siendo la única fuente que valida disponibilidad y cantidad.
- Concretamente: la opción más simple y seguro es un `<form>` HTML nativo apuntando directamente al endpoint Go de fallback (`POST /carrito/items`, cross-origin si Next y Go están en dominios/puertos distintos — esto exige resolver CORS o same-origin antes, lo cual **no está resuelto** y se marca como decisión pendiente, sección W) — o, más conservador aún, un enlace a la página Go de carrito/producto en vez de un formulario embebido, hasta que la topología same-origin esté decidida.

### Estado posterior (bloqueado hasta suite PostgreSQL en verde)
- `AddToCartForm` como Client Component React.
- `Idempotency-Key` generada en cliente o servidor Next, validada por Go.
- Mismo origen (same-origin) obligatorio, o CORS explícitamente diseñado — ninguno de los dos existe hoy.
- Origin/Referer, cookie firmada, CSRF — todo ya existe en Go (`internal/security/csrf.go`, `internal/session`), pero Next nunca lo ha ejercitado en producción.
- Fallback sin JavaScript.
- Mensajes accesibles (`role="status"`/`role="alert"`, ya usado en Go).
- Recarga del carrito tras mutación.

**No se aprueba ningún paso del estado posterior en este documento.** Bloqueado explícitamente hasta que `internal/dbtest` corra en verde contra `cart_integration_*` real (5B7B6C, pendiente de ejecución). `05-cart-integration.md` no se modifica.

---

## I. Endpoint read-only propuesto (conceptual, no implementado)

`GET /api/catalog/products/{identifier}` — nombre consistente con el patrón ya usado por el catálogo (`05-catalog-page.md` sección K, endpoints `/api/catalog/*`, todos de solo lectura y públicos).

`{identifier}` acepta **UUID o slug**, mismo criterio que `FindCatalogProductDetail` (sección B) — no se inventa un mecanismo nuevo de resolución.

### Contrato conceptual
```json
{
  "product": {
    "id": "uuid",
    "name": "Producto",
    "slug": "producto",
    "description": "",
    "long_description": "",
    "category": { "id": "uuid", "name": "Categoría" },
    "available": true,
    "quantity": 0,
    "image_filename": "",
    "images": []
  }
}
```
Ajustado al modelo real: `long_description` nunca es `null` en la vista actual (viene con `COALESCE` a `""`), así que el contrato lo refleja como string, no como nullable — corrigiendo el ejemplo especulativo original de la instrucción de fase para no contradecir el esquema real.

### Detalles del contrato
- **Ruta**: `GET /api/catalog/products/{identifier}`.
- **Identificador**: UUID o slug, dual dispatch idéntico a `FindCatalogProductDetail`.
- **200**: cuerpo como arriba.
- **400**: identificador vacío o con caracteres claramente inválidos (no aplica realmente si se reutiliza el dispatch actual, que ya trata "no es UUID" como "es slug" — un 400 solo tendría sentido para un segmento vacío, que el propio router de Go ya rechaza al no hacer match de patrón).
- **404**: producto no encontrado — mismo criterio que el modal hoy (`db.FindCatalogProductDetail` retorna error si no hay fila).
- **500/503**: fallo de base de datos — nunca debe filtrar detalle del error (mismo estándar ya aplicado en `internal/routes/cart.go`, mensajes fijos, nunca texto de PostgreSQL).
- **Headers**: `Content-Type: application/json`; `Cache-Control` a definir — dato de catálogo cambia con poca frecuencia salvo `available`/`quantity`, que son precisamente los campos que **no deben cachearse agresivamente**. Recomendación conceptual: sin caché o TTL muy corto, decisión final pendiente (sección W).
- **Campos**: exactamente los listados en sección D como "público: sí". Explícitamente excluidos: `main_img` (UUID interno), `qrcode_filename`, `gallery_ids`, `search_vector`, cualquier campo administrativo de `db.Product`.
- **Orden de imágenes**: igual que hoy — alfabético por filename (heredado de la vista SQL, no una elección de este documento).
- **Filename seguro**: el endpoint solo expone el filename crudo; la validación de seguridad ya vive en el proxy Next (sección E) — el endpoint Go no necesita reinventar esa lógica, solo no debe devolver rutas ni paths, solo filenames, igual que la vista actual ya hace.
- **Producto no disponible**: se **incluye** en la respuesta (`available: false`) — no se oculta como 404, es un estado válido de detalle (un producto no disponible sigue siendo consultable).
- **Producto eliminado**: si `products` no tiene la fila, 404 — no hay soft-delete evidenciado en el esquema, se asume borrado real.
- **Slug duplicado**: imposible a nivel de esquema (constraint UNIQUE), no requiere manejo especial.
- **Loader de DB reutilizable**: reutilizar `db.FindCatalogProductDetail` tal cual, no crear una tercera función de resolución de identidad.
- **Pruebas necesarias**: ver sección V.

No se implementa el endpoint en esta fase.

---

## J. Arquitectura Next propuesta (conceptual)

| Archivo | Tipo | Props/fuente | JS | Fallback sin JS | Riesgos |
|---|---|---|---|---|---|
| `frontend/app/(site)/catalogo/producto/[identifier]/page.tsx` | **Server Component** | fetch server-side al endpoint de sección I, usando `identifier` del segmento dinámico | No requerido para contenido base | Página completa SSR | Manejo de 404/500 del backend Go debe traducirse a `notFound()`/error boundary de Next, sin filtrar detalles |
| `frontend/components/product/product-hero.tsx` | Server Component | imagen principal + nombre + categoría | No | Imagen `<img>` nativa | Layout shift si no se fijan dimensiones |
| `frontend/components/product/product-gallery.tsx` | **Client Component solo si hay interacción de cambio de imagen activa** (thumbnails clicleables) — si se implementa como enlaces `<a>`/anchors o CSS puro, puede quedar Server Component | `images[]` | Depende del diseño final de interacción — a decidir en 6B5, no aquí | Debe funcionar como lista de imágenes visible sin JS como mínimo | No convertir toda la galería en carrusel JS-only sin alternativa |
| `frontend/components/product/product-details.tsx` | Server Component | descripción, categoría, disponibilidad | No | Texto plano | Ninguno significativo |
| `frontend/components/product/product-availability.tsx` | Server Component | `available`, `quantity` | No | Texto + ícono, no solo color | Mostrar `quantity` como si fuera stock garantizado — debe llevar advertencia de que es aproximado (ver sección D) |
| `frontend/components/product/product-actions.tsx` | **Client Component** únicamente si incrusta interacción real (ver sección H) — en el estado inicial seguro, puede ser un simple `<a>`/`<form>` Server Component apuntando al flujo Go | product id/slug | Estado inicial: no. Estado posterior: sí | Formulario HTML nativo si se implementa como tal | Este es el único componente con riesgo real de sobre-alcance — no adelantar el estado posterior del carrito aquí |
| `frontend/components/product/related-products.tsx` | Server Component | lista de productos relacionados — **requiere que el endpoint de sección I también exponga relacionados, o un segundo endpoint; no resuelto en esta fase** | No | Enlaces `<a>` nativos | Depende de una decisión de contrato no tomada todavía (sección W) |
| `frontend/lib/api/product.ts` | Server-only fetcher | igual patrón que `frontend/lib/api/catalog-browse.ts` | N/A | N/A | Debe fallar de forma segura si el backend Go no responde (mismo patrón ya usado en catálogo) |
| Tipos (`frontend/lib/types.ts` o archivo dedicado) | — | Reflejar el contrato de sección I exactamente | N/A | N/A | No inventar campos que el backend no envía |

**Preferencia explícita**: página y contenido principal como Server Components; galería funcional sin JavaScript en su forma mínima (lista de imágenes, no necesariamente carrusel); Client Component solo donde exista una interacción real que lo justifique (posible selector de imagen activa, futuro formulario de carrito). No se convierte toda la página en Client Component.

---

## K. Estados

| # | Estado | HTTP | Copy | Navegación | SEO | ¿CTA? | ¿Carrito? | ¿Error o normal? |
|---|---|---|---|---|---|---|---|---|
| 1 | Producto válido | 200 | Datos reales | Normal | Indexable | Sí | Solo estado inicial seguro (enlace a Go) | Normal |
| 2 | Producto no disponible | 200 | "Ese producto ya no está disponible." | Normal | Indexable (sigue siendo contenido válido) | CTA de cotización sí, agregar no | No | Normal |
| 3 | Producto inexistente | 404 | "No se encontró el producto." (base editorial de Go) | Enlace a catálogo | No indexable | No | No | Error esperado, no crítico |
| 4 | Identificador inválido | 404 (tratado igual que inexistente — el dispatch dual no distingue "mal formado" de "no encontrado" hoy) | Igual que #3 | Igual que #3 | No indexable | No | No | Normal (no se distingue de #3 con la lógica actual) |
| 5 | Slug antiguo (si algún día cambia) | Sin mecanismo hoy — ver sección B, decisión pendiente | — | — | — | — | — | Pendiente de diseño futuro |
| 6 | Sin imagen | 200 | Sin mensaje, placeholder visual | Normal | Indexable | Sí | Sí | Normal, no es error |
| 7 | Imagen principal inexistente (filename apunta a archivo físico ausente) | 200 en el detalle, 404 en la imagen individual vía proxy | Placeholder o imagen rota manejada por el componente | Normal | Indexable | Sí | Sí | Normal a nivel de página, error aislado a nivel de imagen |
| 8 | Sin galería | 200 | Sección de galería omitida | Normal | Indexable | Sí | Sí | Normal |
| 9 | Backend Go no disponible | 502/503 desde Next | Mensaje genérico de error, sin detalle técnico | Reintentar / volver | No indexable | No | No | Error |
| 10 | Contrato inválido (respuesta Go no matchea el shape esperado) | 502 desde Next (mismo patrón que el proxy de imágenes) | Mensaje genérico | Reintentar / volver | No indexable | No | No | Error |
| 11 | Carrito bloqueado (5B7B7 sin resolver) | 200 (la página en sí funciona) | Sin mensaje de error — simplemente no se ofrece el flujo Next del carrito, se usa el flujo Go | Normal | Indexable | Sí (cotización), agregar vía Go | Sí, vía Go, no vía Next | Normal, es el estado esperado actual |
| 12 | `quantity` = 0 | 200 | Tratar igual que "no disponible" a nivel de copy si `available=false`; si `available=true` y `quantity=0` (estado inconsistente posible) mostrar igual "no disponible" por seguridad, nunca prometer stock | Normal | Indexable | CTA de cotización sí | No | Normal |
| 13 | Producto eliminado después de enlazado (link roto por borrado real) | 404 | Igual que #3 | Enlace a catálogo | No indexable | No | No | Normal, esperado con contenido dinámico |
| 14 | Ruta histórica QR (`/productos/{id}`) | No es responsabilidad de Next — sigue siendo Go puro | — | — | — | — | — | Fuera de alcance de esta página Next; Go la sirve indefinidamente |

---

## L. SEO

Estado real: Go no gestiona metadata dinámica de producto de forma diferenciada por el momento (el modal no tiene `<title>`/`<meta>` propio distinto del catálogo, ya que ni siquiera es una página aislada — sección F). No hay evidencia de Open Graph, canonical, ni metadataBase configurado en ningún lugar del repo para producto.

Propuesta (conceptual, no implementada):
- **Title**: `{name} | Villa Chenacolo` — patrón ya usado en otras páginas Go (`internal/templates/pages/cart.templ:23`, `"Tu selección | Villa Chenacolo"`).
- **Description**: `description` del producto, truncado si excede el límite recomendado por buscadores.
- **H1**: nombre del producto, único en la página (a diferencia del `<h2>` actual de Go).
- **Breadcrumbs**: ver sección G.
- **URLs**: `/catalogo/producto/{slug}` como forma pública primaria.
- **Metadata dinámica**: usar el mecanismo `generateMetadata` de Next (a implementar en 6B4, no aquí).
- **Open Graph**: pendiente — no se define en esta fase.
- **Canonical**: pendiente — depende de dominio de producción, no confirmado.
- **Indexación**: producto disponible y no disponible se indexan igual (contenido válido); producto eliminado, no indexar (404 real).
- **Slugs antiguos**: sin política todavía (sección B, decisión pendiente).
- **QR**: no afecta SEO de Next — apunta a `/catalogo/producto/{uuid}`, servido por Go, fuera del árbol Next.

**Explícitamente pendientes, no resueltos aquí**: dominio público, canonical final, política de producto no disponible en buscadores, redirects de slug, Open Graph, política de indexación fina. No se agrega schema.org en esta fase.

---

## M. Accesibilidad

- Un solo `<h1>` real por página — corrige el gap actual de Go (`<h2>` dentro de modal).
- `nav` de breadcrumb con `aria-label` apropiado.
- Alt text descriptivo, mejor que el genérico actual de Go (sección E).
- Galería navegable por teclado si se implementa como Client Component interactivo.
- Imagen activa anunciada (si hay selector de imagen).
- Thumbnails con `aria-current`/equivalente si aplican.
- Zoom: no se propone en esta fase — no hay evidencia de que sea un requisito.
- Disponibilidad en texto, nunca solo color (ya es el patrón actual de Go, se conserva).
- Focus visible en todos los controles interactivos.
- Touch targets ≥44px (mismo estándar ya aplicado en 5B7B6A a los formularios de carrito).
- Jerarquía de encabezados correcta y sin saltos.
- `prefers-reduced-motion` respetado si se agrega cualquier animación (relevante de cara a 6B9/Motion, no a esta fase).
- Formularios (si el estado posterior de carrito llega): mismos estándares ya vigentes en `internal/templates/components/cart.templ`.
- Mensajes con `role="status"`/`role="alert"` — mismo patrón que Go.
- Funciona sin JavaScript en su forma base: contenido de producto siempre visible vía SSR; solo interacciones opcionales (galería avanzada, carrito Next) dependen de JS.
- Compatible con lectores de pantalla: estructura semántica HTML estándar, sin dependencia de `div`-soup.

---

## N. Responsive

| Ancho | Comportamiento |
|---|---|
| 360px | Una columna: imagen arriba, contenido abajo. CTA de ancho completo. Sin scroll horizontal. |
| 768px | Similar a 360px, posible ajuste de padding/tipografía. |
| 1024px | Posible transición a dos columnas (imagen + contenido lado a lado), siguiendo el quiebre `xl:` que Go ya usa como referencia conceptual (no literal, Tailwind de Next puede diferir). |
| 1280px | Dos columnas consolidadas, galería de thumbnails visible junto al hero. |
| 1440px | Igual que 1280px con mayor `max-width` de contenido, sin estirar el layout a ancho completo de pantalla. |

- Orden imagen-contenido: imagen primero en mobile (ya es el patrón de Go), lado a lado en desktop.
- Galería: scroll horizontal en mobile, grid o fila fija en desktop.
- Ancho de texto: limitado con `max-width` para legibilidad, no ancho completo en desktop.
- CTA: ancho completo en mobile, ancho de contenido en desktop.
- Descripción larga: con expansión si es muy extensa (patrón "leer más" ya usado en Go, decisión de UX a confirmar en 6B6).
- Breadcrumbs: visibles en todos los anchos, pueden truncarse en mobile.
- FAB (carrito flotante): si existe en el layout compartido de Next (`02-shared-layout.md`), se hereda, no se rediseña aquí.
- Footer: heredado del layout compartido.
- Sin scroll horizontal en ningún ancho.

No se implementa CSS en esta fase.

---

## O. Seguridad

- **Identificador manipulado**: el dispatch dual (sección B) ya maneja "no es UUID → trátalo como slug" de forma segura — un string arbitrario simplemente no encontrará fila y producirá 404, no un error de SQL (usa parámetros nombrados vía `pgx.NamedArgs`, no concatenación — confirmado por lectura de `internal/db/catalog.go:126-138`).
- **UUID inválido**: cae naturalmente en la rama de slug, sin error.
- **Slug con caracteres especiales**: mismo tratamiento, consulta parametrizada — no hay vector de inyección SQL confirmado.
- **XSS**: no se identifica uso de `dangerouslySetInnerHTML` en ningún componente relacionado a esta fase (no se encontró en el research; la descripción se renderiza como texto plano en templ Go, que escapa por defecto). Next deberá mantener la misma disciplina — no renderizar `long_description` como HTML crudo salvo decisión explícita futura y justificada.
- **Path traversal en imágenes**: mitigado por el proxy Next ya existente (sección E) — riesgo teórico cerrado, no nuevo hallazgo.
- **Filename**: mismo punto anterior.
- **URLs absolutas**: el filename servido por el endpoint propuesto (sección I) es relativo — Next construye la URL final vía el proxy, nunca se expone una URL absoluta de Go directamente al navegador (evita fuga de host interno de Go).
- **Campos administrativos**: explícitamente excluidos del contrato (sección I/D) — riesgo real ya observado en `/api/products/{slug}` (sección A), pero **ese es un hallazgo del código existente, no del diseño propuesto aquí**, que evita repetirlo.
- **Quantity/Stock**: el endpoint propuesto expone `quantity` como dato informativo, nunca como fuente de verdad transaccional — la validación real sigue ocurriendo exclusivamente dentro de `internal/cart.Service` (sección D, H).
- **Cache**: recomendación de TTL corto/nulo para no servir disponibilidad desactualizada por mucho tiempo (sección I) — riesgo de negocio, no de seguridad, pero documentado igual.
- **Errores DB**: el contrato exige nunca filtrar mensaje de PostgreSQL (mismo estándar que `internal/routes/cart.go` ya aplica) — riesgo mitigado por diseño, no confirmado como vulnerabilidad activa hoy porque el endpoint no existe todavía.
- **Enumeración**: un UUID no es enumerable de forma práctica; un slug es legible por diseño (es su propósito) — no se considera una vulnerabilidad, es el comportamiento esperado de un catálogo público.
- **Redirect abierto**: no se propone ningún redirect en esta fase (sección B) — no hay superficie nueva.
- **QR malformado**: fuera de alcance de Next — la validación de la URL en el QR ya pasa por el mismo dispatch dual de Go, sin cambios propuestos aquí.
- **CORS**: no resuelto — mencionado como bloqueante para el estado posterior del carrito (sección H), no para la página de consulta en sí, que puede ser puramente SSR server-to-server sin CORS del navegador.
- **Cookies**: la página de consulta en su estado inicial seguro no necesita leer ni escribir cookies de carrito — se difiere al estado posterior.

No se convierten estos puntos en "hallazgos confirmados" salvo el ya documentado y preexistente de `/api/products/{slug}` (sección A), que es evidencia directa de código ya en el repo, no una especulación.

---

## P. Arquitecturas comparadas

| | Fuente de verdad | Seguridad | Duplicación | SSR | SEO | Caché | Carrito | Topología | Rollback | Complejidad | Tests |
|---|---|---|---|---|---|---|---|---|---|---|---|
| **A. Next consume endpoint JSON de Go** | Go (sin cambio) | Alta — contrato explícito, campos controlados | Ninguna — un solo loader Go reutilizado | Sí, en el servidor de Next | Buena, metadata generada en Next | Controlable por Next | Compatible con el estado inicial seguro (sección H) | Requiere que Next pueda alcanzar la red de Go (ya es el patrón del catálogo) | Trivial — apagar la ruta Next, Go sigue sirviendo `/catalogo/producto/*` sin cambios | Baja-media | Mismo patrón ya probado en catálogo (`public_api_test.go`) |
| **B. Next reutiliza HTML/fragmentos del detalle Go** | Go | Media — depende de cuánto HTML crudo se inyecte | Alta — dos sistemas de render coexistiendo | Parcial, mezcla renderizados | Pobre — HTML ajeno no optimizado para metadata de Next | Difícil de razonar | No aplica de forma limpia | Acopla fuertemente Next a la estructura interna de templ Go | Difícil — cambios en templ Go rompen Next silenciosamente | Alta | Frágil, ningún test cubre esto hoy |
| **C. El detalle permanece en Go** | Go | Sin cambio (statu quo) | Ninguna | Sí (ya lo es) | Statu quo (sin `<h1>` real, sin breadcrumbs — gaps ya documentados) | Statu quo | Ya funciona | Ninguna migración | N/A | Mínima | Ya "probado" en producción, aunque sin tests automatizados (sección A) |
| **D. Next accede directo a PostgreSQL** | Ambiguo — dos sistemas leyendo la misma base sin capa de servicio compartida | Baja — duplica lógica de autorización/validación que hoy vive solo en Go | Alta — reimplementa consultas ya existentes en `internal/db` | Sí | Buena | Controlable | No resuelve nada del carrito (que ya está bloqueado por falta de capa transaccional compartida) | Requiere exponer PostgreSQL fuera de la red interna de Go, sin evidencia de que la topología lo permita | Muy difícil — dos codebases dependientes del mismo esquema sin contrato | Alta | Sin infraestructura de pruebas compartida hoy |

### Recomendación
**Opción A.** Es la continuación directa del patrón ya validado y ya en producción para el catálogo (`05-catalog-page.md`, endpoints `/api/catalog/*`), reutiliza `FindCatalogProductDetail` sin reescribirlo, no introduce acoplamiento a la implementación interna de templ Go, y mantiene a Go como única fuente de verdad de productos/stock/disponibilidad — consistente con el contexto de la fase. **No se recomienda la opción D**: no hay evidencia arquitectónica (ni de infraestructura, ni de decisión previa en ningún documento del repo) de que Next deba o pueda acceder directamente a PostgreSQL.

---

## Q. Fases futuras

### 6B1 — Resolver identidad y compatibilidad de rutas
**Objetivo**: confirmar en código (no solo en documento) que el dispatch dual UUID/slug se mantiene y agregar los tests que hoy faltan (sección A/V) sin cambiar comportamiento.
**Archivos**: `internal/routes/catalog_test.go` (nuevo), posible `internal/db/catalog_test.go` (nuevo).
**Riesgos**: ninguno de comportamiento si es solo cobertura de tests; riesgo de descubrir un bug real al escribir tests (p. ej. el gap de `state.Error` en sección F).
**Pruebas**: las de sección V, subsección Go.
**Dependencias**: ninguna, puede iniciar de inmediato tras aprobación de 6A.
**Criterio de cierre**: tests en verde, comportamiento actual documentado y confirmado sin regresiones.

### 6B2 — Endpoint Go read-only de detalle
**Objetivo**: implementar `GET /api/catalog/products/{identifier}` (sección I).
**Archivos**: nuevo handler en `internal/routes/public_api.go` (o archivo dedicado), reutilizando `db.FindCatalogProductDetail`.
**Riesgos**: exponer accidentalmente campos administrativos si no se serializa con cuidado (mismo error ya cometido en `/api/products/{slug}`, a no repetir).
**Pruebas**: contrato completo (200/404/500), campos excluidos, no filtración de errores DB.
**Dependencias**: 6B1 cerrado (identidad confirmada estable).
**Criterio de cierre**: endpoint responde con el contrato exacto de sección I, con tests en verde, sin tocar Next todavía.

### 6B3 — Tipos y fetcher server-only de Next
**Objetivo**: `frontend/lib/api/product.ts` + tipos, replicando el patrón de `catalog-browse.ts`.
**Archivos**: `frontend/lib/api/product.ts`, tipos asociados.
**Riesgos**: desalineación de tipos si el contrato de 6B2 cambia después.
**Pruebas**: unitarias del fetcher (manejo de 404/500/backend caído).
**Dependencias**: 6B2 cerrado.
**Criterio de cierre**: fetcher probado contra el endpoint real (en entorno de desarrollo), sin página todavía.

### 6B4 — Página SSR y estados
**Objetivo**: `frontend/app/(site)/catalogo/producto/[identifier]/page.tsx`, cubriendo los 14 estados de sección K salvo el flujo de carrito activo.
**Archivos**: la página, `product-hero.tsx`, `product-details.tsx`, `product-availability.tsx`.
**Riesgos**: acoplar la página al carrito antes de tiempo (evitarlo explícitamente).
**Pruebas**: SSR con backend real, backend apagado, contrato inválido, cada estado de sección K.
**Dependencias**: 6B3 cerrado.
**Criterio de cierre**: página navegable en desarrollo, todos los estados de sección K cubiertos salvo carrito activo.

### 6B5 — Galería e imágenes
**Objetivo**: `product-gallery.tsx`, integración con el proxy existente.
**Archivos**: el componente de galería.
**Riesgos**: convertir la galería en Client-Component-only sin fallback (evitarlo, sección J).
**Pruebas**: sin imagen, imagen rota, galería vacía, teclado.
**Dependencias**: 6B4 cerrado.
**Criterio de cierre**: galería funcional con y sin JS en su forma mínima.

### 6B6 — Responsive, accesibilidad y SEO
**Objetivo**: aplicar secciones M/N/L de este documento.
**Archivos**: ajustes de estilo y `generateMetadata` en la página.
**Riesgos**: ninguno estructural, es refinamiento.
**Pruebas**: sección V, subsección Next (responsive, reduced motion, metadata).
**Dependencias**: 6B4/6B5 cerrados.
**Criterio de cierre**: checklist de accesibilidad y SEO de este documento satisfecho.

### 6B7 — Integración del carrito, condicionada a PostgreSQL real
**Objetivo**: estado posterior de sección H — `AddToCartForm`, Idempotency-Key, same-origin/CORS resuelto.
**Archivos**: `product-actions.tsx` promovido a Client Component real, nueva infraestructura de fetch de mutación.
**Riesgos**: los ya documentados extensamente en `05-cart-integration.md` — no se repiten aquí.
**Pruebas**: las ya exigidas por la fase 5B7B6 y sucesoras.
**Dependencias explícitas y bloqueantes**: **suite de `internal/dbtest` corriendo en verde contra `cart_integration_*` real.** No inicia antes de eso, sin excepción.
**Criterio de cierre**: el mismo que ya rige 5B7B7.

### 6B8 — QA y preparación de cutover
**Objetivo**: validar regresiones completas (sección V), confirmar que `/catalogo/producto/*` y `/productos/*` de Go siguen intactos y que Next puede convivir con ellos sin conflicto de topología.
**Archivos**: ninguno nuevo necesariamente, es validación.
**Riesgos**: descubrir incompatibilidades de topología no anticipadas (dominio, proxy inverso).
**Pruebas**: toda la matriz de sección V.
**Dependencias**: 6B1-6B7 cerrados.
**Criterio de cierre**: checklist de aceptación completo, sin regresión confirmada.

### 6B9 — Motion posterior
**Objetivo**: animaciones (Framer Motion u otra librería), explícitamente diferido — "Framer Motion sigue pendiente" por instrucción de esta misma fase.
**Archivos**: no determinados aún.
**Riesgos**: no evaluados, fuera de alcance hasta que llegue su turno.
**Pruebas**: `prefers-reduced-motion` como mínimo no negociable.
**Dependencias**: 6B8 cerrado.
**Criterio de cierre**: no definido en esta fase.

No se agrupa la implementación — cada fase tiene su propio cierre y aprobación, siguiendo el mismo patrón ya usado en 5B7B6/5B7B6A/5B7B6B/5B7B6C.

---

## R. Plan de pruebas

### Go
- Identificador válido por UUID.
- Identificador válido por slug.
- Identificador inválido (ninguno de los dos, string arbitrario) → 404, no error 500.
- No encontrado → 404 con mensaje fijo.
- No disponible → 200, `available: false`, resto de campos presentes.
- Sin imágenes → `image_filename: ""`, `images: []`, no error.
- Galería con múltiples imágenes → orden alfabético confirmado.
- Filename con caracteres inválidos en la base (si existiera) → el endpoint no debe romperse, debe devolver el filename tal cual (la validación de seguridad es responsabilidad del proxy Next, no del endpoint Go — separación de responsabilidades ya establecida en sección E).
- Error de base de datos simulado (fake, sin PostgreSQL real necesario para esto, mismo patrón que `internal/cart` ya usa con fakes) → 500/503 sin filtración de detalle.
- Headers de respuesta correctos (`Content-Type`, cache).
- Solo GET permitido — otros métodos, 404/405 según comportamiento del mux.
- Campos administrativos ausentes del JSON de respuesta — test explícito de "no filtración", inspirado en los tests ya existentes de `internal/routes/cart_forms_test.go` que verifican ausencia de campos prohibidos.

### Next
- SSR exitoso con backend disponible.
- Backend apagado → estado de error, no crash de build ni página en blanco.
- Contrato inválido (backend devuelve JSON que no matchea el tipo esperado) → mismo tratamiento que "backend apagado", con log pero sin filtrar detalle al usuario.
- Metadata generada correctamente (`title`, `description`) para producto real.
- Imagen principal carga vía proxy.
- Galería completa e imagen faltante.
- Funciona sin JavaScript (contenido base visible, sin depender de hidratación para lo esencial).
- Navegación por teclado en los elementos interactivos que existan.
- Responsive en los anchos de sección N.
- `prefers-reduced-motion` respetado si aplica (aunque Motion está diferido a 6B9, cualquier transición CSS mínima que se agregue antes debe respetarlo).
- URLs codificadas (slug con acentos/espacios) resuelven correctamente end-to-end.

### Regresiones
- Home (`/`) sin cambios.
- Servicios (`/servicios`) sin cambios.
- Catálogo (`/catalogo`) Next sin cambios de comportamiento.
- Carrito Go (todas las rutas de 5B7B6/5B7B6A) sin cambios.
- Wizard sin cambios.
- Cotización (`/solicitar-cotizacion`) sin cambios.
- QR: `/productos/{id}` sigue respondiendo idéntico desde Go.
- Panel admin sin cambios.
- APIs existentes (`/api/catalog/*`, `/api/products/*`) sin cambios de contrato.

---

## S. Decisiones pendientes

- **Ruta canónica**: `/catalogo/producto/{slug}` como forma pública primaria; UUID aceptado como alias permanente. Confirmado por evidencia (sección B), no pendiente de re-decisión salvo cambio de requisitos.
- **Slug o UUID como identificador de endpoint**: ambos, dual — decidido, no pendiente.
- **Compatibilidad QR**: preservar `/productos/{id}` en Go indefinidamente — decidido, no pendiente.
- **Redirect histórico** (slug que cambia): **pendiente** — no hay caso real documentado todavía, se diseñará si/cuando ocurra.
- **Endpoint**: forma y ruta propuestas en sección I — **pendiente de aprobación explícita** antes de 6B2.
- **Campos públicos**: listado cerrado en sección D — pendiente solo de aprobación, no de investigación adicional.
- **Galería**: orden alfabético heredado, sin cambios — decidido.
- **Producto no disponible**: se muestra, no se oculta — decidido.
- **Producto eliminado**: 404 — decidido.
- **Metadata**: estrategia conceptual en sección L — **pendiente** dominio y canonical real.
- **Canonical**: **pendiente**, depende de dominio de producción no confirmado.
- **Carrito**: bloqueado hasta suite PostgreSQL en verde — **pendiente**, sin fecha.
- **CTA de cotización**: copy reutilizable ya existe — decidido.
- **Productos relacionados**: **pendiente** — requiere decisión de contrato adicional no resuelta en esta fase (sección J).
- **Cache**: TTL corto o nulo propuesto — **pendiente** de decisión final.
- **Topología**: same-origin vs proxy inverso vs CORS — **pendiente**, bloqueante para 6B7.
- **Cutover**: **pendiente**, depende de 6B8.
- **Rollback**: trivial en la opción A recomendada (sección P) — decidido en principio, sin necesidad de mecanismo especial.
- **Motion**: diferido a 6B9 — decidido como orden de trabajo, no como diseño.

---

## Restricciones respetadas en esta fase

Solo se creó `06-product-page.md`. No se tocó código Go, código Next, templates, base de datos, migraciones, tests, assets, `package.json`, `bun.lock`, `next.config.ts`, `05-catalog-page.md` ni `05-cart-integration.md`. No se creó endpoint, página Next, redirect, ni se integró el carrito. No se instaló ninguna dependencia, incluyendo Framer Motion. No se inició 6B1 ni se continuó 5B7B7.
