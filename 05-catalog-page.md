# 05 — Página Catálogo

## Estado del documento

- **Fase:** 5A, auditoría y plan.
- **Estado:** aprobado para planificación; la implementación permanece pendiente.
- **Ruta objetivo:** `/catalogo`.
- **Alcance de este documento:** auditar la página de listado del catálogo Go actual y definir su futura implementación en Next.js sin sustituir todavía la ruta existente.
- **Regla de precedencia:** el código activo confirma el comportamiento real; `01-design-system.md` gobierna el lenguaje visual; `02-shared-layout.md` y los componentes de `frontend/components/site/` gobiernan el layout compartido; `04-services-page.md` fija los precedentes de migración de página (SSR, sin Framer Motion inicial, cutover diferido); este documento decide únicamente lo específico del listado de catálogo.
- **Fuera de este documento:** la página de detalle de producto se planifica en `06-product-page.md`. Aquí solo se define el límite y la integración mínima.

---

## A. Goal

### Propósito

Permitir que un visitante explore el inventario real de mobiliario y piezas decorativas de Villa Chenacolo, lo filtre por categoría, lo busque por texto, navegue por páginas y llegue al detalle de un producto o a una solicitud de cotización, con contenido servido desde el servidor y URLs compartibles.

### Visitante principal

Una persona que ya considera el salón y necesita responder:

1. qué mobiliario y piezas existen realmente;
2. si hay piezas de la categoría que le interesa;
3. si una pieza concreta está disponible y en qué cantidad;
4. cómo pedir esas piezas (cotización o WhatsApp).

No es un visitante de comercio electrónico: el sistema no expone precios ni compra. La acción terminal es una **solicitud de cotización**, no un pago.

### Acciones principales

1. Filtrar por categoría.
2. Buscar por texto.
3. Paginar.
4. Abrir el detalle de un producto.
5. Añadir a la selección (carrito) — hoy existe en Go; su alcance en la primera versión Next se decide en §L.
6. Ir a `/solicitar-cotizacion`.

### Relación con Home

`frontend/components/home/catalog-preview.tsx` ya muestra hasta 4 productos por categoría y enlaza a `/catalogo?categoria={nombre}` y a `/catalogo`. Esos enlaces son un contrato existente: la página nueva **debe** aceptar `categoria` por nombre. Home no debe cambiar.

### Relación con el detalle de producto

El listado enlaza al detalle. Hoy el detalle se abre como modal HTMX sobre `/catalogo` y también responde como página completa. `frontend/components/home/product-card.tsx` ya enlaza a `/catalogo/producto/{slug}`. El comportamiento definitivo del detalle se decide en `06-product-page.md`; ver §M.

### Relación con el carrito

El carrito vive completo en Go, con cookie `cart_id`, tabla propia y unión con cotización. El listado es hoy el punto de entrada principal al carrito. Ver §L.

### Relación con la cotización

`/solicitar-cotizacion` (Go) lee el mismo `cart_id` y precarga los ítems del carrito en el formulario. Migrar el listado sin conservar la cookie rompería ese puente.

### Qué debe sentirse rápido, claro y confiable

- Primer render con productos ya presentes en el HTML (hoy el catálogo Go llega vacío y carga por HTMX tras `load`).
- Filtros y búsqueda reflejados en la URL y compartibles.
- Estado de disponibilidad legible sin abrir el detalle.
- Cero contenido que dependa de JavaScript para existir.

---

## B. Source of truth

### Rutas Go

| Ruta | Método | Registro | Handler | Respuesta |
|---|---|---|---|---|
| `/catalogo` | GET | `internal/routes/routes.go` | `RenderCatalog` | HTML completo |
| `/catalog/categories` | GET | `internal/routes/catalog.go` | `GetCatalogCategories` | Fragmento HTML |
| `/catalog/products` | GET | `internal/routes/catalog.go` | `GetCatalogProducts` | Fragmento HTML |
| `/catalogo/producto/{id}` | GET | `internal/routes/catalog.go` | `GetProductDetail` | Fragmento HTML (HX) o página completa |
| `/productos/{id}` | GET | `internal/routes/catalog.go` | `GetProductDetail` | Igual; compatibilidad con QR impresos |
| `/carrito` | GET | `internal/routes/cart.go` | `GetCart` | Fragmentos HTML |
| `/carrito` | PUT | `internal/routes/cart.go` | `AddToCart` | Fragmentos HTML |
| `/carrito/items` | PATCH | `internal/routes/cart.go` | `UpdateCartQuantity` | Fragmentos HTML |
| `/carrito/items` | DELETE | `internal/routes/cart.go` | `ClearCart` | Fragmentos HTML |
| `/carrito/items/{id}` | DELETE | `internal/routes/cart.go` | `RemoveFromCart` | Fragmentos HTML |
| `/wizard/selection` | GET | `internal/routes/wizard.go` | `RenderWizardModal` | Fragmento HTML |
| `/wizard/{wizard_id}/start` | GET | `internal/routes/wizard.go` | `RenderWizardStep` | Fragmento HTML |
| `/wizard/{wizard_id}/step/{step_id}` | GET | `internal/routes/wizard.go` | `RenderWizardStep` | Fragmento HTML |
| `/wizard/{wizard_id}/complete` | POST | `internal/routes/wizard.go` | `CompleteWizardAndAddToCart` | Fragmentos HTML |
| `/solicitar-cotizacion` | GET/POST | `internal/routes/contact_requests.go` | `RenderQuoteRequest`, `HandleQuoteRequestSubmission` | HTML |
| `/api/catalog/listings` | GET | `internal/routes/public_api.go` | `GetPublicAPICatalogListings` | JSON |
| `/static/uploads/{filename}` | GET | `internal/routes/routes.go` (FileServer sobre `web/static/`) | — | Binario |

Todas las rutas públicas del catálogo pasan por `publicMiddleware` (`internal/routes/routes.go`), que consulta redes sociales y **crea la cookie `cart_id` si falta**.

### Templates y componentes templ

| Archivo | Rol |
|---|---|
| `internal/templates/pages/catalog.templ` | Página `/catalogo`: hero, contenedores HTMX, modal, carrito flotante, banner de wizard, script GSAP/HTMX |
| `internal/templates/components/catalog.templ` | `CategoryFilters`, `ProductGrid`, `ProductCard`, `ProductModal`, `SearchBar` |
| `internal/templates/components/cart.templ` | `FloatingCart`, `CartSidebar` |
| `internal/templates/components/wizard.templ`, `wizard_banner.templ`, `wizard_script.templ` | Asistente |
| `internal/templates/components/gallery_modal.templ` | Galería del modal |
| `internal/templates/components/icons.templ`, `misc.templ` | Iconos y `LoadingIcon` |
| `internal/templates/base_layout.templ` | Layout global Go (HTMX, GSAP, toaster) |
| `internal/templates/util/` | `ExtractURLQuery`, `RenderMixedWithFragments` |

### Consultas y modelos

| Elemento | Ubicación |
|---|---|
| `FindCatalogCategories(search)` | `internal/db/catalog.go` |
| `FindCatalogProducts(categoryID, search, page, limit)` | `internal/db/catalog.go` (envoltorio de `FilterCatalogProducts`) |
| `FilterCatalogProducts(filters)` | `internal/db/catalog.go` |
| `FindCatalogProductDetail(id)` | `internal/db/catalog.go` |
| `FindRelatedProducts(productID, limit)` + fallback | `internal/db/catalog.go` |
| `FindCatalogListings()` | `internal/db/catalog.go` (solo Home) |
| `CatalogCtg`, `CatalogProd`, `CatalogProductFilterParams`, `CatalogProductFilterResult` | `internal/db/catalog.go` |
| `Product` (modelo administrativo) | `internal/db/products.go` |
| `Cart`, `CartItem`, `GetCartIDFromRequest`, `GetOrCreateCart` | `internal/db/cart.go` |

### Migraciones y vistas SQL

| Migración | Aporta |
|---|---|
| `sql/migrations/20250703200655_add_products_table.sql` | `products(id, name, slug, description, long_description, main_img, category, available, quantity)` |
| `sql/migrations/20250703200230_add_categories_table.sql` | `categories` |
| `sql/migrations/20250703195451_add_images_table.sql`, `20250703200851_add_images_products_table.sql` | `images`, `images_products` |
| `sql/migrations/20250724233216_add_catalog_views.sql` | Vistas `catalog_categories` y `catalog_products` |
| `sql/migrations/20250724233342_add_catalog_views_indexes.sql` | `idx_products_category`, `idx_images_products_product_id` |
| `sql/migrations/20250728173855_add_product_fulltext_search.sql` | `search_vector` en `products` y `categories`, triggers, índices GIN |
| `sql/migrations/20250809175837_add_catalog_products_stock_column.sql` | Recrea `catalog_products` con `quantity` y `search_vector` |
| `sql/migrations/20250809012610_add_related_product_optimization_view.sql` | Vista materializada `product_similarities` |
| `sql/migrations/20250726033344_add_carts_table.sql`, `20250809194425_add_cart_customer_columns.sql` | Carrito |
| `sql/migrations/20250815000000_add_wizards_tables.sql` | Wizard |
| `sql/migrations/20250807122419_add_product_and_categories_qrcode_col.sql` | `qrcode_filename` en `products` y `categories` |

Definición vigente de `catalog_products` (tras la migración de stock): `id, name, description, long_description, category_id, category_name, image_url, available, slug, quantity, search_vector, images`. `image_url` es `COALESCE(main_img.filename, '')` — **nombre de archivo**, no ruta. `images` es un `json_agg` de nombres de archivo ordenados por `filename`.

`catalog_categories` = `id, name, product_count`, con `product_count = COUNT(p.id)` sobre **todos** los productos de la categoría, sin filtrar por `available` ni por `quantity`.

### Cookies

`cart_id`, creada en `publicMiddleware` con `Path=/`, `Expires` a 30 días, **sin `HttpOnly`, sin `Secure`, sin `SameSite` explícito**. Leída por `db.GetCartIDFromRequest` en todas las rutas de carrito y por el formulario de cotización.

### JavaScript y CSS

| Archivo | Rol |
|---|---|
| `internal/templates/pages/catalog.templ` → `CatalogScript()` | GSAP + ScrollTrigger, animación de productos, sticky de filtros, botón "subir", modal de producto, galería |
| `web/static/js/eventManager.js` | Registro de handlers `click`/HTMX usados por `data-click-handler-selector` |
| `web/static/js/htmx.js`, `htmx-json-enc.js`, `htmx-response-targets.js` | HTMX y extensiones |
| `web/static/js/gsap.js`, `scroll-trigger.min.js` | Animación |
| `web/static/js/popup.js` | Popups |
| `web/static/css/styles.css`, `web/style/styles.css` | CSS del frontend Go |

### Assets

- Hero del catálogo Go: `web/static/assets/chenacolo-st-1.jpg`. Existe también como `frontend/public/assets/chenacolo-st-1.jpg`.
- Imágenes de producto: `web/static/uploads/{filename}` (propiedad de Go), expuestas a Next mediante `frontend/app/api/catalog/media/[filename]/route.ts`.
- QR generados: `web/static/qrcodes/`.

### Parámetros de URL actuales

| Parámetro | Consumido por | Semántica real |
|---|---|---|
| `buscar` | `RenderCatalog`, `GetCatalogProducts`, `CatalogState` | Término de búsqueda full-text |
| `categoria` | `RenderCatalog`, `GetCatalogCategories`, `GetCatalogProducts` | UUID de categoría **o** nombre exacto de categoría |
| `pagina` | `GetCatalogProducts`, `CatalogState` | Número de página, entero > 0 |
| `por_pagina` | `GetCatalogProducts`, `CatalogState` | Tamaño de página, 1–100 |

No existe parámetro de ordenamiento expuesto: `CatalogProductFilterParams.Sort` existe en la capa DB pero `GetCatalogProducts` nunca lo llena.

---

## C. Scope

### Incluido

- Página de listado `/catalogo` en Next, SSR.
- Búsqueda por texto respaldada por la búsqueda del servidor.
- Filtrado por categoría, compatible con `?categoria={nombre}` generado por Home.
- Paginación con URLs reales.
- Tarjetas de producto con datos reales (nombre, descripción, imagen, categoría, disponibilidad, cantidad).
- Estados: vacío global, categoría vacía, búsqueda sin resultados, página fuera de rango, backend caído, imagen ausente.
- Navegación hacia el detalle de producto (enlace, no implementación del detalle).
- Entrada a cotización.
- Contratos JSON nuevos de solo lectura en Go, mínimos y públicos.

### Fuera de alcance

- Implementación completa del detalle de producto (`06-product-page.md`).
- Rediseño del carrito y de su modelo de datos.
- Panel administrativo y edición de productos.
- Migraciones de base de datos sin justificación explícita.
- Reglas de stock nuevas o cambios de disponibilidad.
- Wizard completo (ver §N).
- Framer Motion (Fase 1B).
- Cutover productivo (§W).
- Precios, promociones o cualquier dato comercial inexistente.

---

## D. Information architecture

Orden vertical propuesto para `/catalogo` en Next:

| # | Bloque | Objetivo | Encabezado | Copy | Componentes | Datos | SSR/cliente | Responsive | Accesibilidad |
|---|---|---|---|---|---|---|---|---|---|
| 1 | `SiteHeader` | Navegación global | — | — | Existente | — | Client existente | Fase 3C intacta | Skip link ya existe |
| 2 | Hero de catálogo | Situar la página | `h1` único | Copy respaldado por la página Go (ver abajo) | `CatalogPageHero` (nuevo) | Estático | SSR | Altura estable | Un solo `h1` |
| 3 | Controles | Buscar y filtrar | `h2` visualmente oculto o visible, a decidir | Etiquetas de formulario | `CatalogFilters`, `CatalogSearchForm`, `CategoryFilter` | Categorías | SSR + form GET | Ver §S | `role="search"`, labels reales |
| 4 | Resumen de resultados | Confirmar filtros y total | — | "N productos" | Parte de `CatalogResults` | `total` | SSR | 1 línea | `aria-live="polite"` solo si se actualiza sin recarga |
| 5 | Resultados | Mostrar productos | `h2` de sección | — | `CatalogResults` + `CatalogProductCard` | `items[]` | SSR | Grid §S | `h3` por producto |
| 6 | Paginación | Navegar páginas | — | — | `CatalogPagination` | `pagination` | SSR | Ventana §J | `nav` + `aria-current="page"` |
| 7 | Estados | Vacío / error | `h2` cuando reemplaza a los resultados | Copy nuevo mínimo | `CatalogEmptyState`, `CatalogErrorState` | — | SSR | 1 columna | `role="status"` |
| 8 | CTA a cotización | Convertir | `h2` | Reutilizar el patrón de `ServicesCta` | Componente propio de catálogo | `contact.ts` | SSR | Botones apilables | Contraste AA |
| 9 | `ContactStrip`, `SiteFooter`, `WhatsAppFab` | Layout | — | — | Existentes | Redes | Heredado | Fase 3C | Intacto |

Copy respaldado del hero Go (`internal/templates/pages/catalog.templ`), a conservar literalmente salvo decisión editorial explícita:

> **Catálogo Villa Chenacolo**
> Explora nuestro catálogo de productos. Cada uno de ellos ha sido elegido buscando el balance perfecto para ser funcional y tener gran estilo.

Elementos del Go que **no** se migran a la primera versión: botón flotante "subir", filtros con `position: fixed` por IntersectionObserver, spinners GSAP, banner de wizard, carrito flotante (según §L).

---

## E. Catalog controls

### Comportamiento actual (Go)

- **Búsqueda:** `internal/templates/components/catalog.templ` → `SearchBar`. Input `type="search"`, `name="buscar"`, `hx-get="/catalog/products"`, `hx-trigger="input changed delay:300ms"`, `hx-push-url="/catalogo"`. Sin JavaScript, el input no hace nada: no hay `<form>` ni botón de envío. Además `hx-push-url="/catalogo"` **borra** el término de la URL, así que una búsqueda no es compartible ni sobrevive a una recarga.
- **Categorías:** `CategoryFilters`. Botón "Todos" + un botón por categoría, todos `hx-get="/catalog/products?categoria={Nombre}"` con `hx-push-url="/catalogo?categoria={Nombre}"`. El nombre va **sin codificar** en la URL construida por `fmt.Sprintf`, lo que rompe con nombres que contengan espacios, `&` o acentos según el navegador. El atributo `data-category-filter` sí guarda el UUID, pero la petición usa el nombre.
- **Limpieza de filtros:** solo mediante el botón "Todos"; no limpia `buscar`.
- **Estado activo:** comparación por `activeCategory == category.Name` en el servidor, más manipulación de clases en el cliente.
- **Sin JavaScript / sin HTMX:** la página se sirve con los contenedores vacíos ("Cargando categorías...", contenedor de productos vacío). El catálogo **no existe** sin JS.

### Propuesta Next

- Un único `<form method="get" action="/catalogo">` que envuelve búsqueda y, si se elige el patrón de `<select>`, también la categoría. Funciona sin JavaScript.
- Botón de envío visible ("Buscar"), no solo `Enter`.
- Categorías como enlaces `<a href>` reales (progressive enhancement natural, compartibles, indexables) o como `<select>` + submit. Recomendación: enlaces.
- "Limpiar filtros" = enlace a `/catalogo`.
- Estado activo con `aria-current="page"` en la categoría seleccionada.
- **Sin debounce en la primera versión**: sin JS de por medio no hay peticiones incrementales que amortiguar. Un debounce solo se justificaría si más adelante se añade filtrado incremental como mejora progresiva, y requeriría decisión aparte.
- Back/forward funcionan porque todo el estado vive en la URL y cada combinación es una navegación real.

### Parámetros

Conservar exactamente los nombres actuales: **`buscar`**, **`categoria`**, **`pagina`**, **`por_pagina`**. No renombrar a `q`/`page`: los nombres actuales ya están en URLs públicas generadas por Home y por el propio catálogo, y renombrarlos obligaría a una capa de compatibilidad sin beneficio. Si más adelante se quieren alias en inglés, deben aceptarse **además** de los actuales, nunca en su lugar.

---

## F. URL state and routing

- Ruta: `/catalogo`.
- Estado completo en query string: `buscar`, `categoria`, `pagina`, `por_pagina`.
- `categoria` acepta hoy UUID **o** nombre exacto (`FindCatalogProducts` hace `SELECT id FROM categories WHERE name = $1` cuando el valor no parsea como UUID). La versión Next debe aceptar ambos para no romper `catalog-preview.tsx`, que genera `/catalogo?categoria=${encodeURIComponent(category.name)}`.
- Codificación: Next debe **codificar siempre** los valores al construir enlaces (el Go actual no lo hace) y decodificar con la semántica estándar de `URLSearchParams`.
- Parámetros inválidos: hoy `pagina` no numérica se ignora silenciosamente (queda 1); `por_pagina` fuera de 1–100 se ignora; `categoria` inexistente produce un resultado vacío sin mensaje específico. La versión Next debe normalizar igual y **no** devolver 4xx por un parámetro decorativo.
- Página fuera de rango: hoy devuelve HTTP 200 con cero productos y una barra de paginación que sigue mostrando el rango real. Propuesta: mantener 200 y mostrar un estado explícito con enlace a la página 1.
- Compartir una búsqueda por URL: hoy imposible (`hx-push-url` fijo). Con el formulario GET queda resuelto sin trabajo extra.
- Enlaces entrantes que no deben romperse: `/catalogo`, `/catalogo?categoria={nombre codificado}` (Home), `/catalogo?pagina=N&categoria=...` (paginación Go), `/catalogo/producto/{slug}` (Home y tarjetas).
- Canonical: **pendiente** hasta que exista URL pública configurada (misma situación declarada en `04-services-page.md`).

---

## G. Product cards

### Campos realmente disponibles

De la vista `catalog_products`, a través de `CatalogProd`:

| Campo | Existe | Comentario |
|---|---|---|
| `id` (UUID) | Sí | Identidad interna |
| `name` | Sí | `VARCHAR(200)`, no nulo |
| `slug` | Sí | `VARCHAR(200)`, único, no nulo |
| `description` | Sí | `VARCHAR(360)`, no nulo — descripción corta |
| `long_description` | Sí | `VARCHAR(512)`, nulable — pertenece al detalle |
| `category_name` / `category_id` | Sí | `LEFT JOIN`, pueden venir vacíos si la categoría falta |
| `image_url` | Sí | Nombre de archivo, cadena vacía si no hay `main_img` |
| `images[]` | Sí | Galería; pertenece al detalle |
| `available` | Sí | Booleano |
| `quantity` | Sí | Entero; `scanCatalogProducts` fuerza `available=false` cuando `quantity <= 0` |
| `qrcode_filename` | Sí en `products` | No expuesto por la vista de catálogo; no debe entrar al listado |

**No existen** precio, descuento, promoción, valoración, dimensiones ni material. No inventarlos.

### Qué muestra la tarjeta Go hoy

Imagen, badge de categoría, badge "`{quantity}` pcs", badge Disponible/No disponible, nombre (`h3`, 2 líneas), descripción corta (2 líneas), botón "Añadir a Selección" y, en `xl`, un overlay con "ver detalle" y "añadir".

### Contrato propuesto para Next

Mostrar: imagen, nombre (`h3`), descripción corta, categoría, estado de disponibilidad. La tarjeta completa es un enlace al detalle.

Decidir en §8: mostrar `quantity` numérica. Es un dato real, pero exponer inventario exacto es una decisión comercial, no técnica. Propuesta por defecto: mostrar solo el estado binario (Disponible / No disponible) y dejar la cantidad para el detalle.

Reglas:

- **Imagen:** `next/image` sobre el proxy existente `/api/catalog/media/{filename}`. Contenedor con `aspect-square` (el mismo del Go y de la tarjeta de Home) para reservar espacio.
- **Alt:** hoy Go usa `alt={product.Name}`, y Home usa `Imagen del producto {name}`. Adoptar el patrón de Home por coherencia.
- **Sin imagen:** `image_url` vacío. Hoy Go genera `src="/static/uploads/"` → imagen rota. Next debe detectar el vacío y usar un marcador de posición del sistema de diseño, sin petición fallida.
- **CTA:** la tarjeta entera enlaza a `/catalogo/producto/{slug}` con `slug` codificado; no anidar botones dentro del enlace (el Go actual anida `<button>` dentro de la zona clicable, lo que rompe la semántica).
- **Carrito:** solo si §L lo aprueba; en tal caso como acción separada fuera del enlace, nunca anidada.
- **Touch target:** ≥ 44 px en cualquier control.
- **Focus:** anillo del sistema (`:focus-visible` global), sin `outline: none`.
- **Semántica:** `article` o `li` dentro de una lista; `h3` para el nombre bajo el `h2` de resultados.
- **`sizes`:** debe reflejar el grid real de §S, no `100vw`.

### Comparación con `ProductCard` de Home

`frontend/components/home/product-card.tsx` recibe `CatalogPreviewProduct` (`id`, `name`, `slug`, `description`, `imageUrl`) y no conoce categoría, disponibilidad ni cantidad. El listado necesita esos tres campos y estados adicionales.

**Recomendación:** mantener componentes separados. `CatalogProductCard` vive en `frontend/components/catalog/` y no modifica el de Home, igual que `EventDetailSection` se mantuvo separado de `EventSection` en Fase 4B2. Motivo: extender el de Home con props opcionales acoplaría dos páginas con contratos de datos distintos y arriesgaría regresiones en una Home ya cerrada.

---

## H. Categories

- **Fuente:** vista `catalog_categories` (`id`, `name`, `product_count`), leída por `FindCatalogCategories`.
- **Identidad estable:** el `id` UUID. La tabla `categories` **no** tiene columna `slug`. Hoy la URL usa el **nombre**, que es la identidad más frágil disponible (renombrar una categoría rompe todos los enlaces existentes).
- **Decisión requerida (§8):** conservar el nombre en la URL por compatibilidad con Home, aceptar además el UUID (ya soportado), y dejar la introducción de un `slug` de categoría como cambio de esquema separado y justificado. Este plan **no** propone la migración.
- **Imagen:** las categorías no tienen imagen en el esquema. No inventar una.
- **Orden:** `ORDER BY name` en la vista; con búsqueda de categorías, por `ts_rank`. Para el listado basta el alfabético.
- **Conteos:** `product_count` incluye productos no disponibles. Si la UI muestra el conteo, debe etiquetarse como total de la categoría, o se necesita un conteo filtrado, que implicaría cambiar la vista (fuera de alcance por ahora).
- **Categorías vacías:** aparecen con `(0)`. Decidir si se ocultan; hoy Go las muestra.
- **Seleccionada:** `aria-current="page"` y estilo activo del sistema.
- **Responsive:** hoy es una fila con `overflow-x-auto`. Aceptable, pero debe conservar foco visible y no producir desbordamiento del documento (§S).
- **Sin JavaScript:** con enlaces reales funciona; hoy no funciona en absoluto porque las categorías se cargan por `hx-trigger="load"`.

---

## I. Search

### Implementación actual

`FilterCatalogProducts` con `SearchMode = fulltext` genera:

```
search_vector @@ plainto_tsquery("spanish", @search_query)
```

y como columna de orden `ts_rank(search_vector, plainto_tsquery("spanish", @search_query)) as search_rank`.

**Hallazgo:** el nombre de la configuración va entre **comillas dobles**. En PostgreSQL las comillas dobles delimitan identificadores, no literales de texto, por lo que `"spanish"` se interpreta como una referencia a columna. Todo indica que cualquier petición con `buscar` no vacío falla en la base de datos y `GetCatalogProducts` responde **HTTP 500 con el texto plano `Failed to find catalog products`**. Las rutas equivalentes que sí funcionan usan comillas simples: `FindCatalogCategories` escribe `plainto_tsquery('spanish', $1)`, igual que los triggers de la migración de full-text.

No pude ejecutar PostgreSQL en esta copia, así que la comprobación es estática. **Debe verificarse contra una base real antes de cualquier migración**, porque cambia por completo el alcance: si se confirma, la búsqueda de productos del catálogo está caída hoy en producción y el endpoint nuevo no puede limitarse a "replicar el comportamiento actual".

### Propiedades reales de la búsqueda (una vez corregida la cita)

- **Normalización y acentos:** la configuración `spanish` de PostgreSQL aplica stemming y desacentuado sobre `name` (peso A), `description` y `long_description` (peso B) y el nombre de categoría (peso C), según `20250728173855_add_product_fulltext_search.sql`.
- **Mayúsculas:** irrelevantes; `to_tsvector` normaliza.
- **Término vacío:** no añade condición; devuelve todo el catálogo paginado.
- **Caracteres especiales:** `plainto_tsquery` los descarta en lugar de fallar; no hay riesgo de sintaxis inyectada por el término.
- **Ranking:** con búsqueda, `ORDER BY search_rank DESC, name ASC`; sin búsqueda, `ORDER BY name ASC`.
- **Seguridad:** el término viaja como argumento con nombre (`pgx.NamedArgs`), no concatenado. No hay inyección SQL por esta vía.
- **Rendimiento:** existe índice GIN sobre `search_vector` (migración de full-text). La paginación usa `LIMIT/OFFSET` con un `COUNT(*)` previo sobre la misma vista.

### Regla para Next

Usar la búsqueda del servidor. **No** implementar búsqueda en cliente sobre todo el catálogo: multiplicaría la carga útil, duplicaría reglas de ranking y perdería el stemming en español.

Mensaje sin resultados: copy nuevo, corto y sin promesas ("No encontramos productos para esa búsqueda"), con acción para limpiar filtros.

---

## J. Pagination

- **Tamaño de página real:** `db.DefaultCatalogPageSize = 16`. Ajustable por petición vía `por_pagina`, límite duro 100 (`GetCatalogProducts` y `FilterCatalogProducts`).
- **Parámetro:** `pagina`, base 1.
- **Metadatos ya calculados** en `CatalogProductFilterResult`: `Total`, `Page`, `Limit`, `TotalPages`, `HasNext`, `HasPrevious`. No hace falta calcularlos en Next.
- **UI actual:** `ProductGrid` renderiza **todas** las páginas como botones, sin ventana. Con muchos productos, la barra crece sin límite y desborda horizontalmente. Además arrastra `categoria` pero **pierde `buscar`** al construir los enlaces, así que paginar cancela la búsqueda.
- **Propuesta Next:** ventana de páginas (primera, anterior, un rango alrededor de la actual, siguiente, última) y **preservación de todos los parámetros vigentes** en cada enlace.
- **Página inválida o fuera de rango:** normalizar a entero ≥ 1; si `pagina > TotalPages` y hay resultados, mostrar estado vacío con enlace a la página 1. HTTP 200 en ambos casos.
- **Scroll y foco:** al ser navegación real, el navegador reinicia la posición. Si se añade navegación cliente más adelante, deberá mover el foco al encabezado de resultados.
- **URLs accesibles:** cada página es un enlace `<a href>` real, no un botón.
- **`rel="prev"`/`rel="next"`:** opcional; los buscadores modernos los ignoran mayoritariamente. Decidir en §8 junto con la política de robots.

---

## K. Data contracts

### Qué falta hoy

`GET /api/catalog/listings` devuelve `{categories:[{name, products:[{id,name,slug,description,image_filename}]}]}`, tope de 4 productos por categoría, **sin** paginación, búsqueda, disponibilidad, cantidad ni identificador de categoría. Es insuficiente para el listado y **debe conservarse tal cual para Home**: cambiarlo rompería `frontend/lib/api/catalog.ts` y la Home ya cerrada.

`GET /api/products`, `/api/products/list` y `/api/products/{slug}` existen y **no requieren autenticación**, pero exponen el modelo administrativo `db.Product` completo (`mainImgId`, `galleryIds`, `qrcodeFilename`, `categoryId`). **No deben reutilizarse** para el frontend público. Ver §4 de seguridad.

### Endpoints mínimos propuestos (nuevos, solo lectura, públicos)

Un único endpoint de listado cubre búsqueda, categoría y paginación, porque la capa DB ya resuelve las tres en una consulta:

```
GET /api/catalog/products?buscar=&categoria=&pagina=&por_pagina=
```

Respuesta conceptual:

```json
{
  "items": [],
  "pagination": { "page": 1, "page_size": 16, "total_items": 0, "total_pages": 0 },
  "filters": { "buscar": "", "categoria": null }
}
```

Y un endpoint separado de categorías, porque su cardinalidad y su cacheabilidad son distintas y se necesitan aunque la lista de productos venga vacía:

```
GET /api/catalog/categories
```

```json
{ "categories": [ { "id": "", "name": "", "product_count": 0 } ] }
```

### Campos del listado

| Campo | Fuente DB | Tipo | Nulable | Exposición segura | Uso en UI |
|---|---|---|---|---|---|
| `id` | `catalog_products.id` | string (UUID) | No | Sí — ya se expone en `/api/catalog/listings` | `key` de React; posible carrito |
| `name` | `products.name` | string | No | Sí | Título de tarjeta |
| `slug` | `products.slug` | string | No | Sí | Enlace al detalle |
| `description` | `products.description` | string | No | Sí | Descripción corta |
| `category_name` | `categories.name` vía vista | string | Sí (LEFT JOIN) | Sí | Badge |
| `category_id` | `products.category` | string (UUID) | Sí | Sí | Enlace de filtro estable |
| `image_filename` | `catalog_products.image_url` | string \| null | Sí (`''` → `null`) | Sí — solo el nombre, igual que en listings | `src` del proxy |
| `available` | `products.available` + regla `quantity<=0` | bool | No | Sí | Estado |
| `quantity` | `products.quantity` | int | No | **Decisión pendiente** (§8) | Solo si se aprueba mostrar inventario |

**No exponer:** `long_description` (pertenece al detalle), `images[]` (detalle), `qrcode_filename`, `main_img_id`, `gallery_ids`, `search_vector`, `search_rank`.

La regla de negocio `quantity <= 0 → available = false` ya vive en `scanCatalogProducts`; el endpoint debe heredarla, no reimplementarla en Next.

### Consumo desde Next

`frontend/lib/api/catalog.ts` ya establece el patrón correcto: `server-only`, `getGoAPIBaseURL()`, `cache: "no-store"`, `AbortSignal.timeout(5000)`, validación estructural del JSON y degradación a `{status: "unavailable"}` sin propagar la excepción. El fetcher del listado debe seguir ese patrón exacto y **no** compartir el módulo de Home para no acoplar dos contratos.

---

## L. Cart boundary

### Cómo funciona hoy

1. **Creación de `cart_id`:** `publicMiddleware` (`internal/routes/routes.go`) comprueba la cookie en **cualquier** ruta pública; si falta, escribe `cart_id` = UUIDv7 con `Path=/`, 30 días, sin `HttpOnly`, sin `Secure`, sin `SameSite`.
2. **Lectura:** `db.GetCartIDFromRequest` en `GetCart`, `AddToCart`, `UpdateCartQuantity`, `RemoveFromCart`, `ClearCart`, y en el formulario de cotización (`internal/routes/contact_requests.go`).
3. **Añadir:** `PUT /carrito` con `product_id` y `source` como campos de formulario. Si el producto ya está, incrementa en 1; si no, resuelve el producto con `FindCatalogProductDetail` y crea el ítem con `MaxQty = product.Quantity`.
4. **Límite de stock:** `Cart.UpdateItemQty` aplica `min(quantity, item.MaxQty)`; `quantity <= 0` elimina el ítem. `MaxQty` se congela en el momento de añadir y no se revalida después.
5. **Actualizar:** `PATCH /carrito/items` con `id` y `action` ∈ {`increase`, `decrease`, `set`}; `set` usa `strconv.Atoi` ignorando el error (un valor no numérico se convierte en 0 → elimina el ítem).
6. **Eliminar:** `DELETE /carrito/items/{id}` (un ítem) y `DELETE /carrito/items` (todo).
7. **Cotización:** `/solicitar-cotizacion` lee el mismo `cart_id`, precarga los ítems y guarda `quote.CartID`.
8. **Respuestas:** siempre fragmentos HTML multi-target (`cartToggle`, `cartSidebar`, `toaster-toast`) más la cabecera `X-Includes-Toast`. Inútiles para un cliente React.

### Qué ocurre si Next sirve `/catalogo` y Go conserva el carrito

Si ambos procesos quedan detrás del mismo origen, la cookie `cart_id` (Path=/) sigue viajando a Go y el carrito y la cotización siguen funcionando. Si quedan en **orígenes distintos**, la cookie deja de ser de primera parte para Go y el puente carrito → cotización se rompe. La topología aún no está decidida (§W), así que esto es una dependencia real, no teórica.

### Opciones

1. **Acciones de carrito en Go, llamadas same-origin desde Next.** Sin duplicar lógica. Requiere origen compartido y una respuesta JSON, porque los fragmentos HTML actuales no sirven a React. Implica tocar Go.
2. **Route Handlers en Next que reenvían a Go.** Añade reenvío de cookies (`Set-Cookie` de vuelta, `credentials`), una superficie más que mantener y riesgo de divergencia; no elimina la necesidad de una respuesta JSON.
3. **Dejar "añadir a la selección" fuera de la primera migración.** El listado enlaza al detalle y a `/solicitar-cotizacion`; el carrito sigue viviendo íntegro en las páginas Go que aún no se migran.

### Recomendación: opción 3 para la primera versión, con opción 1 como paso siguiente

Justificación:

- **Cookies y dominio:** sin topología confirmada no se puede garantizar que `cart_id` siga siendo de primera parte. Construir sobre esa incógnita es lo que más riesgo introduce.
- **Credenciales:** los fetchers actuales de Next usan `credentials: "omit"` deliberadamente. Añadir carrito obliga a reenviar cookies en algunas rutas, un cambio de política de seguridad que merece su propia decisión aprobada.
- **CSRF:** hoy no existe token ni verificación de origen en las rutas de carrito, y son mutables sin autenticación. Ver §4. Migrar el carrito antes de resolver esto extendería el problema a una superficie nueva.
- **Progressive enhancement:** un botón de carrito sin JavaScript exigiría un `POST` con formulario y redirección; Go hoy solo responde fragmentos HTMX. Trabajo real, no cosmético.
- **Riesgo de duplicación:** la opción 2 crea una segunda implementación del contrato del carrito con reglas de stock ya sutiles (`MaxQty` congelado, `set` con `Atoi` silencioso).
- **Alcance:** el objetivo de esta fase es el listado. Diferir el carrito mantiene la migración reversible y verificable.

Consecuencia asumida y visible: la primera versión Next de `/catalogo` **no** tendrá "Añadir a Selección". Debe aprobarse explícitamente (§8) porque es una pérdida funcional frente a la página Go durante la ventana de coexistencia.

---

## M. Product detail boundary

- **En el listado:** imagen principal, nombre, descripción corta, categoría, disponibilidad, enlace.
- **Para `06-product-page.md`:** `long_description`, galería `images[]`, cantidad, productos relacionados, añadir al carrito, QR.
- **Ruta futura:** `/catalogo/producto/{slug}`. Ya la generan `frontend/components/home/product-card.tsx` y las tarjetas Go.
- **Compatibilidad `/productos/{id}`:** ruta viva registrada para QR impresos. `GetProductDetail` acepta UUID **o** slug (`FindCatalogProductDetail` decide con `uuid.Parse`). No puede eliminarse ni romperse: hay códigos QR físicos ya impresos.
- **Modal actual:** `ProductModal` se abre por HTMX sobre `/catalogo` con `hx-push-url`, y la misma URL sirve una página completa cuando no es petición HX. Es una implementación doble con problemas de accesibilidad conocidos (§R).
- **Recomendación:** **diferir** el modal. La primera versión Next navega a una página de detalle. Un modal enrutado (interceptación de rutas) es una decisión propia de `06-product-page.md` y solo tiene sentido si allí se aprueba, con foco atrapado y restaurado.
- **Relacionados:** `FindRelatedProducts` con la vista materializada `product_similarities` y dos fallbacks. Pertenece al detalle, no al listado.
- **QR:** `qrcode_filename` en `products` y `categories`, archivos en `web/static/qrcodes/`. Fuera del listado.

---

## N. Wizard boundary

- **Apertura:** desde `/catalogo`, mediante `WizardBanner` y el modal `#wizard-modal`, cargado por HTMX desde `GET /wizard/selection`. Los controles globales `window.showWizardModal` / `closeWizardModal` se definen en `CatalogScript()`.
- **Datos:** `db.FilterWizards` con `Enabled: 1`; los pasos filtran productos por categorías mediante `FilterCatalogProductsForWizard` (solo disponibles y con `MinQuantity: 1`).
- **Relación con categorías:** cada paso se asocia a categorías, es decir, comparte fuente de datos con el filtro del catálogo.
- **Terminal:** `POST /wizard/{wizard_id}/complete` → `CompleteWizardAndAddToCart`, que escribe en el carrito.
- **Dependencias:** HTMX, JavaScript propio, GSAP, PostgreSQL (wizards, pasos, productos, carrito) y toda la maquinaria de carrito.
- **Riesgos de migrar ahora:** depende del carrito (diferido en §L), tiene su propia máquina de estados multi-paso, y su superficie administrativa es amplia (`/panel/asistentes/*`).
- **Recomendación: diferir.** No pertenece a la primera migración del listado. Merece su propio plan una vez resuelto el carrito. Mientras tanto, el wizard sigue disponible en las rutas Go no migradas.

---

## O. Component architecture

### Existentes reutilizables

| Componente | Tipo | Uso en catálogo | Motivo |
|---|---|---|---|
| `frontend/components/site/site-header.tsx` | Client existente | Layout | Heredado del grupo `(site)` |
| `frontend/components/site/site-footer.tsx` | Server | Layout | Heredado |
| `frontend/components/site/contact-strip.tsx` | Server | Layout | Heredado |
| `frontend/components/site/whatsapp-fab.tsx` | Server | Layout | Heredado, responsive de Fase 3C |
| `frontend/components/shared/section-heading.tsx` | Server | Encabezados `h2` | Nivel fijo `h2`, compatible |

### Nuevos propuestos (no se crean en esta fase)

| Archivo previsto | Tipo | Responsabilidad | Props | Datos | Estado | Dependencias |
|---|---|---|---|---|---|---|
| `components/catalog/catalog-page-hero.tsx` | Server | Hero con el `h1` único | copy, asset | Estático | — | `next/image` |
| `components/catalog/catalog-filters.tsx` | Server | Envolver búsqueda + categorías en un `form` GET | `buscar`, `categoria`, `categories` | Categorías | URL | — |
| `components/catalog/catalog-search-form.tsx` | Server | Campo `buscar` + submit | `defaultValue`, `hiddenParams` | — | URL | — |
| `components/catalog/category-filter.tsx` | Server | Lista de categorías como enlaces | `categories`, `active` | Categorías | URL | `next/link` |
| `components/catalog/catalog-results.tsx` | Server | Grid + recuento + orquestación de estados | `items`, `pagination`, `filters` | Listado | — | Card, EmptyState |
| `components/catalog/catalog-product-card.tsx` | Server | Tarjeta de producto | campos de §G | Producto | — | `next/image`, `next/link` |
| `components/catalog/catalog-pagination.tsx` | Server | Ventana de páginas preservando parámetros | `pagination`, `searchParams` | Paginación | URL | `next/link` |
| `components/catalog/catalog-empty-state.tsx` | Server | Vacío global, categoría vacía, búsqueda sin resultados | `variant`, `resetHref` | — | — | — |
| `components/catalog/catalog-error-state.tsx` | Server | Backend no disponible | `retryHref` | — | — | — |
| `lib/api/catalog-products.ts` | Módulo servidor | Fetcher del nuevo endpoint | — | — | — | `server-only`, `lib/env` |
| `lib/types.ts` | Modificación | Tipos del listado | — | — | — | — |

`CatalogSkeleton` **no se justifica** si la página es SSR con datos ya resueltos: no hay ventana de carga en cliente que rellenar. Solo tendría sentido junto con un `loading.tsx` (§Q).

### Separación explícita respecto de Home

- No modificar `frontend/components/home/product-card.tsx` ni `catalog-preview.tsx`.
- No modificar `frontend/lib/api/catalog.ts` ni el contrato de `/api/catalog/listings`.
- Precedente aplicado: en Fase 4B2 se creó `EventDetailSection` en vez de extender `EventSection`, y Home quedó byte-idéntica en texto.

### Client Components

Ninguno previsto. Todo el estado vive en la URL y toda la interacción son enlaces y un formulario GET. Si más adelante se aprueba filtrado incremental o carrito, cada uno requerirá su propia justificación de hidratación.

---

## P. Server and client boundary

- **Servidor:** página, hero, controles, tarjetas, paginación, estados. Todo el contenido llega en el HTML inicial.
- **Hidratación:** solo la que ya aporta el layout compartido (header y menú móvil).
- **Formulario GET:** búsqueda y, si se elige `<select>`, la categoría. Sin JavaScript sigue funcionando.
- **Estado en URL:** `buscar`, `categoria`, `pagina`, `por_pagina`. Nada más.
- **Nada en `useState`:** ni término de búsqueda, ni categoría activa, ni página; duplicarlos en estado de cliente rompe compartir la URL y back/forward.
- **Evitar fetch duplicado:** una única llamada al endpoint de listado por render en el Server Component de página; las categorías en su propia llamada. No repetir el mismo fetch en componentes hijos.
- **Evitar Client Component global:** los controles no deben elevar `"use client"` a la página. Un enlace y un `form` no necesitan JavaScript.
- **Sin JavaScript:** productos, filtros, paginación, estados y CTA visibles y utilizables; solo el menú móvil del layout requiere JS, como ya está documentado.

---

## Q. Loading and errors

- **`loading.tsx`:** solo se justifica si la consulta al backend es lenta de forma perceptible. Como el endpoint es paginado (16 ítems) y `no-store`, la recomendación es **no** añadirlo en la primera versión y reevaluar con medidas reales.
- **Skeleton:** ligado a lo anterior; sin `loading.tsx` no aporta.
- **Vacío global:** el catálogo no tiene productos. Mensaje neutro + enlace a `/solicitar-cotizacion`.
- **Búsqueda sin resultados:** mensaje con el término citado + "Limpiar búsqueda".
- **Categoría vacía:** mensaje + enlace a "Todas las categorías".
- **API no disponible:** el fetcher degrada a `{status: "unavailable"}` (patrón ya usado en Home). La página responde **HTTP 200** con estado de error visible, no 500: el hero y la navegación siguen siendo útiles. Mismo criterio que `CatalogPreview` en Home.
- **Imagen faltante:** marcador de posición, sin petición fallida (§G).
- **Producto inválido:** pertenece al detalle (`06-product-page.md`).
- **Reintento:** enlace a la misma URL; nada de reintentos automáticos.
- **Errores técnicos:** nunca visibles. Hoy Go escribe `Failed to find catalog products` en texto plano al usuario; eso no se migra.
- **Códigos HTTP:** 200 para vacío, búsqueda sin resultados, página fuera de rango y backend caído. Sin 500 por contenido ausente.

---

## R. Accessibility

### Requisitos para la versión Next

- Un solo `h1` (hero). `h2` para controles y resultados; `h3` para el nombre de cada producto.
- Landmarks: `main` del layout, `role="search"` en el formulario de búsqueda, `nav` con `aria-label` en la paginación.
- Etiqueta real asociada al campo de búsqueda (`<label>`), no solo `placeholder`.
- Recuento de resultados anunciado. Con navegación completa de página, el recuento en el DOM basta; `aria-live` **solo** si se añade actualización parcial.
- Paginación: `aria-current="page"` en la página actual, enlaces reales, nombres accesibles ("Página 3", no "3" a secas).
- Categoría activa: `aria-current="page"`.
- Foco visible en todos los controles (regla global `:focus-visible`).
- Objetivos táctiles ≥ 44 px.
- `alt` descriptivo por producto; marcador de posición sin texto alternativo redundante.
- Contraste AA en tarjetas, badges y estados. Nota heredada de Fase 4B3: los tokens `type-eyebrow` y el par `accent`/`accent-foreground` miden 3.5:1 y siguen pendientes de decisión de sistema de diseño (§8).
- `prefers-reduced-motion` respetado desde el primer render; ningún contenido comienza en `opacity: 0`.
- Contenido completo sin JavaScript.
- Tras filtrar o paginar hay navegación real, así que el foco lo gestiona el navegador. Si más adelante se añade actualización parcial, el foco debe moverse al encabezado de resultados.

### Defectos actuales que no deben migrarse

- Categorías y productos se cargan con `hx-trigger="load"`: sin JS la página queda vacía.
- Tarjetas y grid comienzan con `opacity-0` y dependen de GSAP para hacerse visibles; sin alternativa para movimiento reducido.
- Filtros de categoría son `<button>` que cambian la URL, en vez de enlaces.
- `<button>` anidado dentro de zonas clicables en la tarjeta.
- El modal de producto no declara `role="dialog"` ni `aria-modal`, no atrapa el foco, no cierra con Escape y no restaura el foco; el cierre manipula el historial con `history.pushState` a mano.
- El campo de búsqueda no tiene `<label>`.
- La barra de paginación renderiza todas las páginas, sin ventana ni nombres accesibles.
- El botón "subir" es un icono sin nombre accesible.
- El layout Go declara `lang="en"` con contenido en español.

---

## S. Responsive

Reglas comunes: contenedor y espaciado de `01-design-system.md`; todo medio con `max-width: 100%`; contenedores con `min-width: 0` donde haya texto largo; ningún texto, URL o badge puede ampliar el viewport.

| Ancho | Grid de productos | Controles | Categorías | Tarjeta | Paginación | Header | FAB |
|---|---|---|---|---|---|---|---|
| 360 px | 1 columna | Apilados, ancho completo | Fila con desplazamiento horizontal contenido | Imagen `aspect-square`, descripción 2 líneas | Ventana compacta (primera/anterior/actual/siguiente/última) | Teléfono oculto | Visible |
| 768 px | 2 columnas | Búsqueda y filtro en fila si caben | Fila desplazable | Igual | Ventana compacta | Teléfono oculto | **Oculto** |
| 1024 px | 3 columnas | En fila | Fila completa | Igual | Ventana completa | Teléfono visible | Visible |
| 1280 px | 4 columnas | En fila | Fila completa | Igual | Ventana completa | Teléfono visible | Visible |
| 1440 px | 4 columnas, contenedor al máximo del sistema | En fila | Fila completa | Igual | Ventana completa | Teléfono visible | Visible |

- **Tablet 768–1023:** dos columnas; no forzar tres, que comprimiría nombre y descripción.
- **1024–1279:** usar el contenedor ancho real (`max-w-7xl`), nunca `max-w-2xl`. Este es exactamente el defecto que la página Go arrastra y que Fase 4B2 evitó en Servicios; el Go de catálogo repite el patrón (`max-w-2xl md:max-w-4xl xl:max-w-7xl`).
- **Reglas heredadas de Fase 3C, intocables:** teléfono del header desde `lg`; FAB visible bajo `md`, oculto entre `md` y `lg`, visible desde `lg`.
- **Sin scroll horizontal:** `document.documentElement.scrollWidth <= clientWidth` en los cinco anchos. La fila de categorías debe desplazarse dentro de su propio contenedor.
- **Espacio inferior:** reservar para que el FAB no cubra la paginación ni el CTA final.

---

## T. Performance

- **SSR** con datos ya resueltos; sin cascada de peticiones en cliente.
- **Consultas paginadas:** `LIMIT/OFFSET` + `COUNT(*)`, ya implementado. Tope duro de 100 por página.
- **Índices existentes:** `idx_products_category`, `idx_images_products_product_id`, GIN sobre `search_vector`. No se proponen índices nuevos sin medición.
- **N+1:** la vista `catalog_products` agrega la galería con un subselect por fila. En el listado, `images[]` **no se expone**, así que el endpoint nuevo debería seleccionar columnas explícitas en vez de `SELECT *` sobre la vista, para no pagar la agregación de galería por producto. Verificar con `EXPLAIN` antes de darlo por bueno.
- **Imágenes:** `next/image` sobre `/api/catalog/media/{filename}`. El proxy actual valida el nombre con `^[\p{L}\p{N}._:-]+$`, rechaza `/`, `\`, `.`, `..` y NUL, exige `Content-Type: image/*`, usa `redirect: "error"` y responde `Cache-Control: no-store` con `X-Content-Type-Options: nosniff`. **Sirve sin cambios** para el listado.
- **`no-store` en el proxy:** correcto en cuanto a seguridad, pero significa que cada visita revalida cada miniatura. Con 16 productos por página conviene medir; cualquier cambio de política de caché de media es una decisión aparte (§8).
- **`priority`:** ninguna imagen de producto debería usarlo. La candidata LCP es el asset del hero, y solo esa.
- **Lazy loading:** por defecto de `next/image` para todas las tarjetas.
- **`sizes`:** reflejar el grid de §S (aproximadamente `(min-width:1280px) 22vw, (min-width:1024px) 30vw, (min-width:768px) 45vw, calc(100vw - 3rem)`), nunca `100vw`.
- **Caché y revalidación:** partir de `no-store` como hacen los fetchers actuales; el catálogo cambia desde el panel y no hay invalidación. Introducir `revalidate` es una decisión pendiente (§8).
- **Tamaño de respuesta:** 16 productos con campos recortados. No enviar `long_description` ni `images[]`.
- **CLS:** contenedores con `aspect-square`, altura mínima estable para descripciones recortadas, geometría fija de botones.
- **JavaScript:** cero bundle de página adicional.

---

## U. SEO

- **Title:** `Catálogo` (el template del layout produce "Catálogo · Villa Chenacolo").
- **Description:** debe describir solo lo que existe: mobiliario y piezas decorativas disponibles para eventos en Villa Chenacolo. Sin precios, sin cifras de inventario, sin superlativos no respaldados.
- **Un solo `h1`**, en el hero.
- **URLs indexables:** `/catalogo` y `/catalogo?categoria=...` son contenido legítimo.
- **Paginación:** indexable; `pagina=1` debería resolver al mismo contenido que `/catalogo`.
- **Búsquedas internas:** `?buscar=` genera combinaciones infinitas. Propuesta: `robots: noindex, follow` cuando `buscar` esté presente. Requiere aprobación (§8).
- **Categorías:** indexables.
- **Canonical:** pendiente hasta que exista URL pública. Mismo bloqueo declarado en `04-services-page.md`.
- **Metadata dinámica:** solo si está respaldada, por ejemplo `title` con el nombre real de la categoría filtrada. Nada inventado.
- **Sin dominio inventado**, sin autor, sin fechas, sin Twitter, sin datos estructurados de producto (requerirían precio y disponibilidad en formato comercial, que este sistema no tiene).

---

## V. Motion

### Versión inicial

- Sin Framer Motion, sin GSAP, sin ScrollTrigger.
- Contenido visible desde el HTML inicial; ningún elemento parte de `opacity: 0`.
- Solo transiciones CSS de hover/focus ya admitidas por el sistema (elevación de tarjeta, escala de imagen), anuladas por el bloque `prefers-reduced-motion` de `globals.css`.

### Mejora futura, bloqueada por Fase 1B

Tras aprobar e instalar Framer Motion, podría evaluarse una aparición escalonada de tarjetas equivalente a la del Go actual, siempre como mejora progresiva: su ausencia, fallo o reducción nunca cambia visibilidad, orden ni acceso a los controles.

---

## W. Routing and cutover

- **Propietario actual:** Go, `GET /catalogo` con `publicMiddleware`.
- **Propietario futuro:** Next, `frontend/app/(site)/catalogo/page.tsx`.
- **Dependencias:**
  - `/static/uploads/*` sigue siendo de Go; Next accede por su propio proxy `/api/catalog/media/{filename}`, que ya funciona.
  - Carrito: `PUT/PATCH/DELETE /carrito*` siguen en Go; con el alcance recomendado (§L) el listado Next no los usa.
  - Detalle: `/catalogo/producto/{id}` y `/productos/{id}` siguen en Go hasta `06-product-page.md`. **Migrar el listado sin el detalle deja rutas hermanas en procesos distintos**; la regla de entrada debe reflejarlo con precisión.
  - APIs: se añadirían `GET /api/catalog/products` y `GET /api/catalog/categories` en Go; `/api/catalog/listings` permanece intacto para Home.
- **Regla conceptual** (por precedencia, la más específica primero):
  1. `/panel/*`, `/carrito/*`, `/wizard/*`, `/static/*` → Go
  2. `/catalogo/producto/*` y `/productos/*` → Go (hasta `06-product-page.md`)
  3. `/catalogo` (exacto) → Next
  4. `/servicios` (exacto) → Next
  5. `/` (exacto) → Next, según la decisión pendiente de Fase 4B3
  6. `/api/*` → Go, salvo los path exactos que Next conserva (`/api/_health`, `/api/catalog/media/*`, y `/api/contact-requests` según decisión pendiente)
  7. Resto → Go
- **Rollback:** el handler Go `RenderCatalog` y `pages/catalog.templ` permanecen intactos; revertir = retirar la regla de `/catalogo`. Sin cambios de datos ni migraciones que deshacer.
- **Smoke tests posteriores:** ver §Y.

**El cutover no puede ejecutarse de forma segura hasta conocer la topología real de despliegue.**

---

## X. Implementation phases

### Fase 5B1 — Verificación de la búsqueda y contratos read-only en Go

**Objetivo:** confirmar el hallazgo de `plainto_tsquery("spanish", ...)` contra una base real y definir los endpoints públicos del listado.

**Crear:** ninguno todavía; documento de resultado de la verificación.
**Modificar:** ninguno.
**Cambios Go:** ninguno sin aprobación. Si se confirma el fallo, la corrección (comillas simples) es de una línea, pero afecta a la ruta Go viva y necesita autorización explícita.
**Riesgos:** dar por buena una búsqueda que hoy devuelve 500; construir el endpoint sobre una consulta rota.
**Pruebas:** ejecutar `/catalog/products?buscar=mesa` contra PostgreSQL real; `go test -mod=vendor . ./cmd/... ./internal/...`.
**Terminación:** se sabe con certeza si la búsqueda funciona y qué debe corregirse.

### Fase 5B2 — Endpoints públicos de catálogo en Go

**Objetivo:** exponer `GET /api/catalog/products` y `GET /api/catalog/categories`.
**Crear:** handlers en `internal/routes/public_api.go` (o un archivo hermano) y sus tests.
**Modificar:** `internal/routes/public_api.go` (registro).
**Cambios Go:** solo aditivos; `/api/catalog/listings` intacto.
**Cambios Next:** ninguno.
**Riesgos:** exponer campos administrativos; romper `listings`; filtrar errores internos.
**Pruebas:** tests de handler con loader falso (patrón ya existente en `public_api_test.go`): búsqueda, categoría por nombre y por UUID, paginación, parámetros inválidos, vacío, error de DB, `Content-Type`, métodos no permitidos.
**Terminación:** endpoints estables, sin campos administrativos, con tests verdes.

### Fase 5B3 — Tipos y fetchers en Next

**Objetivo:** consumir los endpoints desde el servidor.
**Crear:** `frontend/lib/api/catalog-products.ts`.
**Modificar:** `frontend/lib/types.ts` (tipos nuevos, sin tocar los existentes).
**Riesgos:** acoplar con el módulo de Home; propagar excepciones a la página.
**Pruebas:** `bun run lint`, `bun run build`; respuesta malformada degrada a `unavailable`.
**Terminación:** fetchers tipados y tolerantes a fallo, sin cambios en Home.

### Fase 5B4 — Página SSR y filtros GET

**Objetivo:** `/catalogo` en Next con hero, controles y resultados.
**Crear:** `frontend/app/(site)/catalogo/page.tsx`, `catalog-page-hero.tsx`, `catalog-filters.tsx`, `catalog-search-form.tsx`, `category-filter.tsx`.
**Modificar:** ninguno.
**Riesgos:** colisión conceptual con la ruta Go; más de un `h1`; perder compatibilidad con `?categoria={nombre}`.
**Pruebas:** HTML SSR, un `h1`, formulario funcional sin JS, `?categoria` de Home resuelve.
**Terminación:** la página compila y filtra con datos reales.

### Fase 5B5 — Tarjetas y paginación

**Objetivo:** grid de productos y navegación de páginas.
**Crear:** `catalog-product-card.tsx`, `catalog-results.tsx`, `catalog-pagination.tsx`.
**Riesgos:** `sizes` incorrecto; CLS; perder parámetros al paginar; imagen rota sin `image_filename`.
**Pruebas:** los cinco anchos, sin desbordamiento, imágenes 200, paginación conserva `buscar` y `categoria`.
**Terminación:** listado navegable con URLs compartibles.

### Fase 5B6 — Estados y accesibilidad

**Objetivo:** cerrar vacíos, errores y experiencia asistiva.
**Crear:** `catalog-empty-state.tsx`, `catalog-error-state.tsx`.
**Modificar:** solo componentes de catálogo si la validación lo exige.
**Riesgos:** exponer errores técnicos; anuncios `aria-live` redundantes; foco perdido.
**Pruebas:** backend apagado, categoría vacía, búsqueda sin resultados, página fuera de rango, recorrido por teclado.
**Terminación:** todos los estados legibles, sin JavaScript y con lector de pantalla.

### Fase 5B7 — Integración controlada con carrito (condicional)

**Objetivo:** solo si §8 aprueba adelantar el carrito. Requiere respuesta JSON en Go, política de cookies y decisión de CSRF.
**Riesgos:** cookies entre orígenes, CSRF, duplicación de reglas de stock.
**Terminación:** no se inicia sin topología confirmada y aprobación explícita.

### Fase 5B8 — QA y regresiones

**Objetivo:** validar el sistema completo.
**Pruebas:** §Y íntegra, con backend disponible y apagado, y comparación de Home y Servicios antes/después.
**Terminación:** cero regresiones, cero desbordamiento, sin fugas de `GO_API_BASE_URL`.

### Fase 5B9 — Preparación del cutover

**Objetivo:** definir la regla de propiedad exacta de `/catalogo` sin ejecutarla.
**Riesgos:** enviar `/catalogo/producto/*` a Next por accidente; perder el carrito; romper `/solicitar-cotizacion`.
**Terminación:** regla escrita, rollback verificado, smoke tests listos.

### Fase 5B10 — Motion futura

Bloqueada por Fase 1B. Sin imports condicionales ni marcadores de posición antes de esa aprobación.

---

## Y. Testing plan

### Validación obligatoria de repositorio

Desde la raíz:

```powershell
go test -mod=vendor . ./cmd/... ./internal/...
```

Desde `frontend/`:

```powershell
bun install --frozen-lockfile
bun run lint
bun run build
```

### Pruebas de endpoints

- Búsqueda: término con acentos, con mayúsculas, con caracteres especiales, vacío.
- Categoría: por UUID, por nombre exacto, por nombre inexistente, con nombre que requiera codificación.
- Paginación: primera, intermedia, última, fuera de rango, `por_pagina` en 1, 100 y 101.
- Parámetros inválidos: `pagina=abc`, `pagina=-3`, `por_pagina=0`, parámetros desconocidos.
- Resultados vacíos: catálogo vacío, categoría vacía, búsqueda sin coincidencias.
- Error de base de datos: respuesta JSON de error genérica, sin mensaje interno ni consulta.
- JSON: `Content-Type: application/json; charset=utf-8`, `Cache-Control: no-store`.
- Métodos HTTP: `POST`/`PUT`/`DELETE` sobre endpoints de solo lectura deben rechazarse.
- Cookies: los endpoints de listado **no** deben crear ni exigir `cart_id`.
- Sin filtración: ni rutas internas, ni nombres de tabla, ni `GO_API_BASE_URL`.

### Pruebas visuales y de robustez

- Anchos 360, 768, 1024, 1280 y 1440: `scrollWidth <= clientWidth`, columnas según §S, header sin desbordamiento, FAB según Fase 3C.
- Teclado: recorrido completo, foco visible, orden lógico, paginación y categorías alcanzables.
- Sin JavaScript: productos, filtros, paginación, estados y CTA visibles y usables.
- `prefers-reduced-motion`: sin regresiones ni contenido oculto.
- Assets: todas las miniaturas responden 200; ninguna 404; productos sin imagen muestran marcador.
- Consola: sin errores ni avisos de hidratación.
- Layout compartido: header, footer, ContactStrip, FAB y menú móvil intactos.
- Regresiones: Home byte-idéntica en texto renderizado; `/servicios` sin cambios; `/api/_health`, `/api/catalog/listings`, `/api/catalog/media/*` y `/api/contact-requests` sin cambios de comportamiento.
- Backend apagado: `/catalogo` responde 200 con estado de error visible; Home y Servicios conservan sus fallbacks.

### Pruebas de carrito

Solo si §8 aprueba la Fase 5B7. Con el alcance recomendado, la única comprobación es negativa: el listado Next **no** emite peticiones de carrito ni crea la cookie `cart_id`.

---

## Z. Acceptance criteria

- [ ] `/catalogo` en Next muestra productos reales de la base, sin datos simulados.
- [ ] Búsqueda por texto funciona en el servidor y es compartible por URL.
- [ ] Filtro por categoría funciona con nombre (compatible con Home) y con UUID.
- [ ] Paginación funciona y conserva `buscar` y `categoria` en cada enlace.
- [ ] Todo el estado vive en la URL; back/forward funcionan sin JavaScript.
- [ ] La página es SSR y su contenido existe con JavaScript deshabilitado.
- [ ] No se exponen campos administrativos (`qrcode_filename`, `main_img_id`, `gallery_ids`, `long_description`, `images[]`, `search_vector`).
- [ ] `/api/catalog/listings` y la Home permanecen sin cambios.
- [ ] Responsive verificado en 360, 768, 1024, 1280 y 1440 px, sin scroll horizontal.
- [ ] Un solo `h1`, jerarquía coherente, paginación y categorías con `aria-current`, foco visible, objetivos ≥ 44 px.
- [ ] Solo el asset del hero usa `priority`; las miniaturas cargan de forma diferida con `sizes` correcto.
- [ ] Estados de vacío, sin resultados, categoría vacía, página fuera de rango y backend caído devuelven 200 con mensaje claro y sin detalle técnico.
- [ ] Sin regresiones en Home, Servicios, layout compartido ni APIs existentes.
- [ ] El backend Go sigue siendo la única fuente de verdad de productos y categorías.
- [ ] El carrito queda preservado en Go y su exclusión de la primera versión está aprobada por escrito.
- [ ] `go test`, `bun install --frozen-lockfile`, `bun run lint` y `bun run build` pasan.
- [ ] La ruta está lista para asumir tráfico mediante una regla aislada, con el handler Go conservado como rollback.
- [ ] `/catalogo/producto/*` y `/productos/*` siguen respondiendo desde Go.

---

## Problemas detectados

### Seguridad

| # | Hallazgo | Evidencia | Clasificación |
|---|---|---|---|
| 1 | `GET /api/products`, `/api/products/list`, `/api/products/{slug}` son públicos y devuelven el modelo administrativo completo (`mainImgId`, `galleryIds`, `qrcodeFilename`) | `internal/routes/products.go` | Debe corregirse antes del cutover (no reutilizar en el frontend público; restringir es decisión del backend) |
| 2 | Rutas de carrito mutables sin autenticación ni token CSRF ni verificación de origen (`PUT /carrito`, `PATCH`/`DELETE /carrito/items*`) | `internal/routes/cart.go` | Debe corregirse antes de migrar el carrito; no bloquea el listado si se difiere |
| 3 | `cart_id` aceptado tal cual desde la cookie del cliente: cualquiera puede fijar el UUID de otro y leer o modificar ese carrito con `GET /carrito` | `db.GetCartIDFromRequest`, `db.GetOrCreateCart` | Debe corregirse antes de migrar el carrito |
| 4 | Cookie `cart_id` sin `HttpOnly`, sin `Secure`, sin `SameSite` explícito | `publicMiddleware` en `internal/routes/routes.go` | Debe corregirse antes del cutover |
| 5 | `UpdateCartQuantity` con `action=set` usa `strconv.Atoi` ignorando el error: un valor no numérico se vuelve 0 y elimina el ítem en silencio | `internal/routes/cart.go` | Puede corregirse después |
| 6 | `MaxQty` se congela al añadir; si el stock baja luego, el carrito conserva el límite antiguo | `internal/db/cart.go` | Puede corregirse después |
| 7 | Errores internos devueltos como texto plano al usuario (`Failed to find catalog products`) con `Content-Type` implícito | `internal/routes/catalog.go` | Debe corregirse antes del cutover (no migrar el patrón) |
| 8 | `GetCatalogCategories` en su rama de error escribe 500 **y luego** renderiza HTML: cabecera de error con cuerpo aparentemente válido | `internal/routes/catalog.go` | Puede corregirse después |
| 9 | Nombres de categoría insertados en URLs sin codificar (`fmt.Sprintf("/catalog/products?categoria=%s", category.Name)`) | `internal/templates/components/catalog.templ` | Puede corregirse después; Next debe codificar siempre |
| 10 | Path traversal en imágenes: el proxy de Next ya valida el nombre, rechaza separadores y exige `image/*` | `frontend/app/api/catalog/media/[filename]/route.ts` | Sin hallazgo; mantener tal cual |
| 11 | Inyección SQL: todas las consultas de catálogo usan argumentos posicionales o con nombre | `internal/db/catalog.go` | Sin hallazgo |
| 12 | XSS: templ escapa por defecto; no hay `templ.Raw` en las plantillas del catálogo | `internal/templates/components/catalog.templ` | Sin hallazgo |
| 13 | CORS: el frontend Next llama a Go desde el servidor; no hay dependencia de CORS y no debe introducirse | `frontend/lib/api/*` | Sin hallazgo |
| 14 | `plainto_tsquery("spanish", ...)` con comillas dobles: probable error de ejecución en toda búsqueda de productos | `internal/db/catalog.go` (`buildCatalogProductQueryConditions`, `buildCatalogProductSearchRankSelect`) | **Bloqueante para migración** — verificar contra base real antes de diseñar el endpoint |
| 15 | `FindCatalogListings` usa `ROW_NUMBER() OVER (PARTITION BY p.category)` sin `ORDER BY` dentro de la ventana: los 4 productos por categoría de Home no son deterministas | `internal/db/catalog.go` | Fuera del alcance del catálogo público (afecta a Home; documentar, no tocar) |
| 16 | La creación de `cart_id` ocurre en **todas** las rutas públicas, incluidas páginas editoriales que no tienen carrito | `publicMiddleware` | Fuera del alcance del catálogo público |

### Accesibilidad

Recogidos en §R. Los que impiden migrar tal cual: contenido inexistente sin JavaScript, tarjetas que arrancan invisibles, filtros como `<button>` en vez de enlaces, modal sin `role="dialog"`, sin foco atrapado, sin Escape y sin restauración de foco, campo de búsqueda sin `<label>`, paginación sin ventana ni nombres accesibles.

### Duplicación y relación con Home

- `db.FindCatalogListings` (Home) y `db.FilterCatalogProducts` (catálogo) consultan lo mismo por caminos distintos: la primera contra las tablas base, la segunda contra la vista `catalog_products`. No unificarlas en esta fase; documentar la divergencia.
- `publicAPICatalogProduct` (JSON de Home) es un subconjunto de lo que necesita el listado. Los tipos `CatalogPreviewProduct` y `CatalogPreviewCategory` de `frontend/lib/types.ts` **no** deben ampliarse: el listado define los suyos.
- `ProductCard` de Home y la tarjeta del catálogo comparten aspecto pero no contrato de datos. Se mantienen separados (§G).
- El proxy de media es común y **se reutiliza sin cambios**.

---

## Decisiones pendientes

Antes de implementar hay que aprobar:

1. **Búsqueda:** confirmar contra base real si `plainto_tsquery("spanish", ...)` falla y autorizar (o no) la corrección de una línea en `internal/db/catalog.go`. Condiciona el alcance de toda la fase.
2. **Alcance del carrito en la primera versión:** se recomienda diferirlo (§L); implica que el listado Next no tendrá "Añadir a Selección" durante la coexistencia.
3. **Modal de detalle:** se recomienda diferirlo y navegar a página; confirmar antes de `06-product-page.md`.
4. **Wizard:** se recomienda diferirlo.
5. **Tamaño de página:** conservar 16 o fijar otro valor; decidir si `por_pagina` sigue siendo público.
6. **Ordenamiento:** hoy no hay parámetro expuesto. Decidir si se añade `orden` (la capa DB ya lo soporta) o se mantiene `name ASC` / relevancia.
7. **Identidad de categoría:** conservar el nombre en la URL, o introducir un `slug` de categoría (cambio de esquema, plan aparte).
8. **Conteo por categoría:** mostrar el total actual (incluye no disponibles) o pedir un conteo filtrado a la vista.
9. **Mostrar `quantity` en la tarjeta:** decisión comercial sobre exponer inventario exacto.
10. **Productos sin stock:** ocultarlos, mostrarlos atenuados o mostrarlos normales con badge. Hoy Go los muestra con badge "No disponible" y botón deshabilitado.
11. **Contratos JSON:** aprobar la forma exacta de `GET /api/catalog/products` y `GET /api/catalog/categories`, incluidos los nombres de parámetros.
12. **Caché:** `no-store` frente a `revalidate` para el listado y para el proxy de media.
13. **Robots:** `noindex` para URLs con `buscar`; política para combinaciones de filtros.
14. **Topología de despliegue:** sin ella no hay cutover (§W) ni decisión válida sobre el carrito.
15. **Propietario de `/api/contact-requests`:** duplicado entre Go y Next, pendiente desde Fase 4B3.
16. **URL pública:** desbloquea canonical y Open Graph específicos.
17. **Contraste del sistema de diseño:** `type-eyebrow` y el par `accent`/`accent-foreground` miden 3.5:1; pendiente desde Fase 4B3 y aplicable a badges y CTA del catálogo.
18. **Framer Motion (Fase 1B):** sigue bloqueada y separada.

---

## Resultado de Fase 5A

Este documento es la fuente de verdad para la futura migración del listado del catálogo. En Fase 5A no se implementa la página, no se crean componentes ni endpoints, no se modifica Go ni Next, no se cambia el propietario de `/catalogo` y no se inicia `06-product-page.md`.
