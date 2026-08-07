# 08 — Estado final y preparación de producción

Fase 8. Cierre técnico consolidado. Documento generado, no aprobado para cutover.

## Estado general

**~98% de la migración pública completa.** Todas las páginas públicas migrables ya existen en Next (Fases 1-7). Bloqueo único y persistente: carrito Next, condicionado a la suite PostgreSQL real, que **no corrió esta ejecución** (variables ausentes en el entorno). Cotización y reservaciones quedan correctamente clasificadas como informativas, no simuladas.

---

## 1. Matriz de rutas públicas

| Ruta | Método | Dueño Go | Next | Estado | DB | Carrito | JS | QA | Dueño recomendado |
|---|---|---|---|---|---|---|---|---|---|
| `/` | GET | `RenderIndex` | Sí | Funcional | Sí | No | No | Estático | Next |
| `/servicios` | GET | `RenderServices` | Sí | Funcional | No | No | No | Estático | Next |
| `/catalogo` | GET | `RenderCatalog` | Sí | Funcional (SSR + API) | Sí | No | No | Estático | Next |
| `/catalogo/producto/{id}` | GET | `GetProductDetail` | Sí (`[identifier]`) | Funcional | Sí | No (lectura) | No | Estático | Next |
| `/productos/{id}` | GET | `GetProductDetail` | No (deliberado) | Rollback/QR | Sí | No | No | Cubierto por tests Go | **Go** (QR impreso) |
| `/experiencia` | GET | `RenderSalon` | Sí | Funcional, simplificada (sin popup/lightbox) | No | No | No | Estático | Next |
| `/reservaciones` | GET | `RenderReservations` | Sí | Informativa (Caso C, ver §2) | No | No | No | Estático | Next |
| `/solicitar-cotizacion` | GET | `RenderQuoteRequest` | Sí | Informativa (bloqueada por carrito) | No (en Next) | Sí (en Go) | No | Estático | **Go** hasta desbloquear |
| `/solicitar-cotizacion` | POST | `HandleQuoteRequestSubmission` | No | No migrado | Sí | Sí | No (progresivo en Go) | N/A | **Go** |
| `/politica-privacidad` | GET | `RenderPrivacyPolicy` | Sí | Funcional, copy literal | No | No | No | Estático | Next |
| `/terminos-servicio` | GET | `RenderTermsOfService` | Sí | Funcional, copy literal | No | No | No | Estático | Next |
| `/politica-cookies` | GET | `RenderCookiePolicy` | Sí | Funcional, copy literal | No | No | No | Estático | Next |
| Carrito (`/carrito*`, `/api/cart*`) | varios | `internal/routes/cart.go` | No | **Bloqueado** | Sí | Sí | Progresivo (Go) | Suite sin ejecutar | **Go** hasta suite PostgreSQL en verde |
| Wizard (`/wizard/*`) | varios | `internal/routes/wizard.go` | No | No migrado | Sí | Sí | Parcial | No auditado a fondo | **Go** |
| QR (`/productos/{UUID}`) | GET | igual que detalle | No | Compat | Sí | No | No | Cubierto | **Go** |
| APIs públicas (`/api/catalog/*`, `/api/_health`, `/api/socials`, `/api/contact-requests`) | GET/POST | `public_api.go` | Consumidas por Next | Funcional | Sí (catálogo) | No | No | Tests Go | **Go** (API), Next como consumidor |
| `/api/products/{slug}` (heredado) | GET | `products_public.go` | No | No consumido por Next | Sí | No | No | Tests Go (6B1) | Go, deprecar cuando 6B2 sea el único contrato |
| Admin (`/panel/*`) | varios | múltiples | No | Fuera de alcance | Sí | No | Sí | No auditado esta fase | Go (siempre) |

---

## 2. Reservaciones — auditoría completa

- **Template**: `internal/templates/pages/reservations.templ`. Formulario `id="reservation-form"`, `hx-post="/api/reservations"`, campos `name`, `phone`, `date` (`datetime-local`), sin `action` HTML nativo (solo `hx-post` — **no funciona sin JavaScript incluso en Go hoy**).
- **Modelo/tabla/migración**: **ninguno**. `grep -rln "reservation" internal/db/*.go sql/migrations/*.sql` → cero resultados.
- **Handler esperado**: ninguno implementado. `grep -rn "reservation" internal/routes/*.go` → cero resultados (solo el propio `RenderReservations`, que sirve el HTML).
- **`/api/reservations`**: no registrado en `NewRouter` ni en ningún archivo de `internal/routes`. Confirmado — no es un caso de "handler existe pero no registrado" (Caso B): **no existe handler, modelo ni tabla** (Caso C puro).
- **Confirmación/email/notificación**: no existen — no hay a qué apuntarles.
- **Tests**: ninguno.

**Clasificación: Caso C.** No se inventó esquema, tabla, ni handler. `/reservaciones` en Next (Fase 7) es informativa: mismo hero real, sin formulario roto, CTA a WhatsApp/teléfono ya aprobados. **Contenido pendiente de aprobación del negocio**: si se desea un flujo real de reservación, se necesita definir campos, tabla, y proceso de confirmación — nada de eso existe hoy en ningún nivel del sistema.

---

## 3. Cotización — auditoría y límite seguro

- `GET /solicitar-cotizacion` (`RenderQuoteRequest`, `withCartSession`): renderiza el formulario **junto con los items del carrito de la sesión actual** (`formState.Cart`) — inseparable del carrito.
- `POST /solicitar-cotizacion` (`HandleQuoteRequestSubmission`, `withProtectedCartSession` con CSRF): valida nombre/teléfono/fecha/tipo de evento, pero también depende de la sesión de carrito para asociar los items cotizados.
- `PUT/DELETE /cotizacion/carrito/items/{id}`: mutación directa del carrito de cotización — mismo bloqueo.
- **No existe una porción independiente del carrito** que pueda desacoplarse de forma segura: los campos de contacto por sí solos, sin los productos cotizados, no cumplen el propósito real del flujo (cotizar productos específicos del catálogo).

**Conclusión: todo el flujo depende del carrito bloqueado.** No se conectó ningún formulario real a Go desde Next. `/solicitar-cotizacion` en Next permanece informativa (Fase 7), con enlace a WhatsApp/teléfono, sin simular selección de productos, sin `cart_id`, sin almacenar nada en React.

---

## 4. Suite PostgreSQL real

```
CART_INTEGRATION_TEST_DATABASE_URL → no configurada en este entorno
CART_INTEGRATION_TEST_ALLOW_DESTRUCTIVE → no configurada en este entorno
```

**No ejecutada.** Ninguna variable presente (`env | grep -i "CART_INTEGRATION\|DATABASE_URL"` → vacío). Por instrucción explícita: no se inventaron credenciales, no se instaló PostgreSQL, no se creó Docker. El bloqueo se mantiene exactamente como en 5B7B6C.

**Consecuencia directa**: condición del §7 ("integrar solo si la suite pasa completamente") **no se cumple** → carrito Next **no implementado** esta fase. Ningún botón, contador, drawer, POST JSON, ni Idempotency-Key en React.

---

## 5. Dominio, canonical y metadata

Búsqueda realizada en: variables de entorno, `frontend/.env.example`, `.env.example`, README, código Go, componentes Next. **Ningún dominio público confirmado.** El único valor encontrado es `https://villachenacolo.com` hardcodeado en la generación de QR (`internal/routes/products.go:217,354`) y `villachenacolo@gmail.com` como email de contacto — ninguno de los dos es evidencia de topología de despliegue confirmada, por instrucción explícita de no asumir esto.

**No se configuró**: `metadataBase`, `canonical`, Open Graph absoluto, `schema.org`. Metadata de todas las páginas permanece relativa (`title`/`description` únicamente), sin cambios respecto a fases previas.

**Pendiente de negocio**: confirmar dominio público real y decisión de topología antes de poder añadir `metadataBase`/canonical con seguridad.

---

## 6. QA

### QA visual de navegador: **no ejecutado**
Mismo límite estructural de todas las fases previas: Go no puede levantar sin una conexión PostgreSQL válida (`db.Connect()` falla al arrancar sin `DATABASE_URL`), y no existe una en este entorno. No se afirma ninguna prueba de navegador no realizada.

### QA estático ejecutado
- `bun run build` — compiló todas las rutas, sin errores de tipo.
- Inspección de HTML generado (vía build + lectura de componentes): un `<h1>` por página, jerarquía de headings correcta, `alt` presentes, touch targets ≥44px, focus visible (clases `focus-visible:outline-*` ya estandarizadas), sin `dangerouslySetInnerHTML` en ningún componente nuevo.
- Responsive: cerrado por diseño (tokens compartidos `container-main`/grid `lg:`), no verificado visualmente en los 5 anchos solicitados — **QA visual pendiente**, no ejecutado.

---

## 7. Seguridad

| Ítem | Resultado |
|---|---|
| CORS | Ninguno agregado en ningún fetcher/endpoint nuevo |
| CSRF | Intacto — no tocado |
| Cookies | Next no emite cookies nuevas (páginas de Fase 7-8 son 100% estáticas/SSR sin sesión) |
| Origin/Referer | Sin cambios |
| Redirects abiertos | Ninguno introducido |
| Path traversal | Sin cambios (proxy de imágenes intacto) |
| XSS / `dangerouslySetInnerHTML` | No usado en ningún archivo nuevo |
| Campos administrativos expuestos | Ninguno nuevo |
| Errores DB filtrados | N/A — páginas nuevas no consultan DB |
| Body ilimitado / Content-Type | N/A — sin formularios nuevos |
| Mutaciones públicas nuevas | Ninguna |

### Hallazgo separado — no corregido esta fase (fuera de alcance, admin)
**`POST /api/categories`, `PUT /api/categories/{id}`, `DELETE /api/categories/{id}`** (`internal/routes/categories.go:41-44`) están registrados **sin `auth.ValidateAuth`**, a diferencia de todas las demás rutas de categorías (`/categorias/*`, que sí llevan `auth.ValidateAuth`). Esto permite crear/editar/eliminar categorías **sin autenticación** desde el JSON API heredado. Es más severo que el hallazgo ya documentado en `/api/products/{slug}` (que solo filtraba lectura): aquí hay **escritura no autenticada**. No corregido — no es un bloqueo de esta fase (no afecta páginas públicas ni cutover), y una corrección real requiere decidir si `/api/categories` tiene algún consumidor legítimo sin auth antes de envolverlo (mismo patrón de auditoría que 6B1 aplicó a `/api/products/{slug}`). Recomendado como próxima fase de seguridad dedicada.

---

## 8. Route ownership y topología recomendada

### Opciones evaluadas
- **A. Reverse proxy — Next páginas, Go APIs/mutaciones**: requiere decisión de infraestructura no confirmada (no hay config de proxy inverso en el repo).
- **B. Same-origin por path routing**: mismo requisito, sin evidencia de que exista hoy.
- **C. Mantener bloqueadas en Go**: statu quo para todo lo que depende de carrito (cotización real, mutaciones de carrito, wizard).

**Recomendación**: **C para todo lo dependiente de carrito** (statu quo, sin cambios); **A como destino final** una vez exista configuración de proxy confirmada — es la única opción compatible con "Go sigue siendo fuente de verdad" sin exponer PostgreSQL directamente a Next. No se elige A ni B como acción inmediata porque ninguna tiene evidencia de infraestructura real en este repositorio.

### No se crean redirects productivos ni se cambia producción.

---

## 9. Cutover y rollback

- **Cutover: no ejecutado.** Next y Go sirven las mismas rutas de contenido en paralelo (páginas públicas); ninguna ruta Go fue eliminada, modificada de comportamiento, o redirigida.
- **Rollback**: trivial para todo lo migrado — dejar de enlazar/servir la ruta Next equivalente; Go sigue respondiendo exactamente igual que antes de la Fase 6/7 en todas las rutas.
- **QR**: intacto, cero cambios en `qrgen` ni en las URLs embebidas.

---

## 10. Deuda técnica conocida

1. Carrito Next — bloqueado por suite PostgreSQL sin ejecutar (bloqueante duro, sin fecha).
2. Cotización real — depende de (1).
3. Reservaciones reales — sin modelo/tabla/handler; requiere definición de negocio antes de cualquier implementación.
4. `POST/PUT/DELETE /api/categories` sin autenticación — hallazgo de seguridad, sin corregir.
5. `/api/products/{slug}` heredado — sigue activo en paralelo al nuevo contrato 6B2, sin plan de deprecación formal.
6. Dominio/canonical/metadataBase — sin confirmar.
7. QA visual de navegador — pendiente en todas las páginas migradas desde 6B en adelante, por ausencia de PostgreSQL en este entorno.
8. Galería/tour de `/experiencia` — versión simplificada (sin lightbox ni overlay de video), decisión de alcance, no un defecto.

---

## Checklist de producción (antes de cutover real)

- [ ] Confirmar dominio público y añadir `metadataBase`/canonical.
- [ ] Definir y confirmar topología (proxy inverso vs same-origin) con evidencia de infraestructura real.
- [ ] Ejecutar suite `internal/dbtest`/`internal/cart` contra `cart_integration_*` real, en verde, con cadena completa de migraciones.
- [ ] Solo entonces: implementar carrito Next (tarjetas, detalle, header, cotización).
- [ ] QA visual real en navegador, 360-1440px, con Go y Next ambos corriendo contra datos reales.
- [ ] Decidir y corregir `/api/categories` sin auth (hallazgo de seguridad).
- [ ] Decisión de negocio sobre reservaciones reales (o mantener informativa permanentemente).
- [ ] Confirmar `/api/products/{slug}` heredado: deprecar o documentar como permanente.
