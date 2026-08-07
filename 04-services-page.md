# 04 — Página Servicios

## Estado del documento

- **Fase:** 4A, auditoría y plan.
- **Estado:** aprobado para planificación; la implementación permanece pendiente.
- **Ruta objetivo:** `/servicios`.
- **Alcance de este documento:** conservar la oferta comprobable de la página Go actual y definir su futura implementación en Next.js sin sustituir todavía la ruta existente.
- **Regla de precedencia:** el código activo confirma el comportamiento real; `01-design-system.md` gobierna el lenguaje visual; `02-shared-layout.md` y los componentes actuales de `frontend/components/site/` gobiernan el layout compartido; este documento decide únicamente lo específico de Servicios.

---

## A. Goal

### Propósito

Presentar, de forma clara y verificable, los seis tipos de evento que hoy aparecen en la página Go de Servicios y conducir al visitante a una conversación contextual por WhatsApp o a la ruta existente de solicitud de cotización.

### Visitante principal

Una persona que ya considera celebrar un evento en Villa Chenacolo y necesita responder tres preguntas antes de contactar:

1. si su tipo de evento está contemplado;
2. qué espacios y apoyos están respaldados para ese tipo de evento;
3. cómo pedir información o iniciar una cotización.

### Acción principal esperada

La acción primaria de cada bloque es **“Me interesa saber más”**, con un mensaje de WhatsApp contextual al tipo de evento y una URL construida a partir de `frontend/lib/copy/contact.ts`. La acción global al inicio y al cierre es **“Solicitar cotización”**, con destino `/solicitar-cotizacion`.

### Relación con la Home

La Home introduce la oferta y muestra los mismos seis tipos de evento dentro de una narrativa general. Servicios debe profundizar en esa oferta, conservar sus anclas públicas y ofrecer una lectura enfocada. No debe duplicar la estructura completa de Home, su video, su galería, su catálogo ni su formulario de contacto.

### Relación con solicitar cotización

Servicios no incorpora ni reimplementa el flujo de cotización. Solo enlaza a `/solicitar-cotizacion`, que continúa siendo una ruta Go hasta que exista un plan específico para migrarla.

---

## B. Source of truth

### Fuentes revisadas y autoridad

| Fuente | Evidencia que aporta | Autoridad para este plan |
|---|---|---|
| `internal/routes/routes.go` | Registro, handler y middleware de `/servicios` | Comportamiento Go actual |
| `internal/templates/pages/services.templ` | Estructura, copy, assets, anclas y CTA actuales | Oferta comprobable de Servicios |
| `internal/templates/base_layout.templ` | Dependencias globales y composición de layout Go | Auditoría del estado actual |
| `internal/templates/components/cta_section.templ` | CTA final actual | Copy y destino respaldados |
| `internal/templates/components/header.templ` | Header Go y menú móvil | Auditoría del estado actual |
| `internal/templates/components/desktop_nav.templ` | Navegación de escritorio Go | Auditoría del estado actual |
| `internal/templates/components/footer.templ` | Footer Go y redes dinámicas | Auditoría del estado actual |
| `internal/templates/components/toaster.templ` | Toaster global | Dependencia heredada, no necesidad de Servicios |
| `web/style/styles.css` | CSS y fuentes del frontend Go | Auditoría visual y responsive actual |
| `web/static/assets/` | Archivos usados por la página Go | Assets de contenido respaldados |
| `frontend/public/assets/` | Copia estática disponible para Next | Fuente física futura de imágenes |
| `frontend/lib/copy/home.ts` | Copy, tipos, mensajes y presentación de los seis eventos en Home | Implementación Next actual que no debe sufrir regresiones |
| `frontend/lib/copy/contact.ts` | Teléfono, WhatsApp y datos de contacto canónicos | Única fuente futura para contacto |
| `frontend/lib/copy/nav.ts` | Enlace existente a `/servicios` | Navegación Next actual |
| `frontend/components/home/event-section.tsx` | Contrato y markup de los eventos de Home | Candidato a adaptar, no reutilización automática |
| `frontend/components/shared/section-heading.tsx` | Encabezados editoriales `h2` | Reutilizable donde la semántica coincida |
| `frontend/components/shared/check-list.tsx` | Listas con icono y semántica de lista | Reutilizable |
| `frontend/components/site/site-header.tsx` | Header Next estabilizado en Fase 3C | Layout compartido futuro |
| `frontend/components/site/site-footer.tsx` | Footer Next y redes | Layout compartido futuro |
| `01-design-system.md` | Tokens, tipografía, superficies, foco e imágenes | Sistema visual obligatorio |
| `02-shared-layout.md` | Intención del layout compartido | Referencia, subordinada al código estabilizado de Fase 3C |

### Ruta, handler y middleware actuales

- Registro: `GET /servicios` en `internal/routes/routes.go`.
- Handler: `RenderServices` en `internal/routes/routes.go`.
- Render: `pages.Services().Render(r.Context(), w)`.
- Template principal: `internal/templates/pages/services.templ`.
- Layout: `templates.BaseLayout("Servicios | Villa Chenacolo", nil, false)`.
- Middleware: `publicMiddleware(RenderServices)`.

El handler no consulta la base de datos. Sin embargo, `publicMiddleware` sí ejecuta `db.GetSocialLinks()`, agrega las redes al contexto y crea una cookie `cart_id` cuando falta. Por tanto, la oferta de Servicios es estática, pero la ruta Go tiene hoy una dependencia indirecta de PostgreSQL para el footer y un efecto lateral de carrito que no pertenece al contenido de la página.

### Componentes templ utilizados

La página usa de forma directa `components.CTASection()` y, a través de `BaseLayout`, Header, navegación de escritorio, Footer, Toaster, iconos y utilidades compartidas. También hereda hojas de estilo, HTMX, extensiones HTMX, GSAP, ScrollTrigger y el script general de eventos.

### Secciones visibles y orden Go actual

1. `seccion-titulo` — hero.
2. `seccion-bodas` — Bodas.
3. `seccion-quinceaneras` — Quinceañeras y celebraciones.
4. `seccion-bautizos` — Bautizos y comuniones.
5. `seccion-corporativos` — Eventos corporativos.
6. `seccion-graduaciones` — Graduaciones.
7. `seccion-privadas` — Fiestas privadas.
8. `seccion-cta` — CTA final compartido.

### Assets de Servicios

| Uso actual | Asset Go | Copia Next | Dimensiones | Contenido comprobado |
|---|---|---|---:|---|
| Hero | `web/static/assets/chenacolo_24.jpeg` | `frontend/public/assets/chenacolo_24.jpeg` | 852 × 1280 | Piezas ceremoniales doradas sobre una mesa; no es un bar |
| Bodas | `web/static/assets/chenacolo_11.jpeg` | `frontend/public/assets/chenacolo_11.jpeg` | 1280 × 852 | Nave de la capilla con lámpara de cristal |
| Quinceañeras | `web/static/assets/chenacolo_15.jpeg` | `frontend/public/assets/chenacolo_15.jpeg` | 852 × 1280 | Lámparas de cristal del salón |
| Bautizos | `web/static/assets/chenacolo_10.jpeg` | `frontend/public/assets/chenacolo_10.jpeg` | 852 × 1280 | Nave y acceso interior de la capilla |
| Corporativos | `web/static/assets/chenacolo_8.jpeg` | `frontend/public/assets/chenacolo_8.jpeg` | 1280 × 852 | Acceso, jardines y fachada de la villa |
| Graduaciones | `web/static/assets/chenacolo_13.jpeg` | `frontend/public/assets/chenacolo_13.jpeg` | 1280 × 852 | Lámpara de cristal enmarcada por ventanales |
| Privadas | `web/static/assets/chenacolo_4.jpeg` | `frontend/public/assets/chenacolo_4.jpeg` | 1280 × 852 | Fachada y fuente de la villa |
| Icono CTA | `web/static/assets/wsp.svg` | `frontend/public/assets/wsp.svg` | SVG | Símbolo de WhatsApp del frontend Go |

Los siete archivos fotográficos existen en ambos árboles y las copias comparadas tienen el mismo tamaño y hash. La futura página Next debe referenciarlos mediante `/assets/...`; no debe duplicarlos ni volver a sincronizarlos durante esta implementación.

### Enlaces y CTA actuales

- Cada servicio abre WhatsApp en una pestaña nueva.
- Bodas usa la etiqueta “Más información”; los otros cinco bloques usan “Me interesa saber más >”.
- Todos los enlaces Go codifican el número antiguo `6181556407` y omiten `rel="noopener noreferrer"`.
- El CTA final dice “¡Agenda ya!”, explica “Ven a tener la mejor experiencia de eventos en Chenacolo” y enlaza a `/solicitar-cotizacion` con “Solicita tu cotización”.
- La implementación futura no preservará el número antiguo ni concatenará URLs manualmente: tomará la base de `frontend/lib/copy/contact.ts` y conservará el contexto editorial del mensaje.

### Datos y dependencias

| Elemento | Estado actual | Decisión futura |
|---|---|---|
| Copy y oferta de los seis eventos | Estáticos en templ; duplicados parcialmente en Home | Centralizar datos base compartidos |
| Imágenes | Archivos estáticos | Usar `next/image` desde `frontend/public/assets/` |
| Redes del footer | Dinámicas vía `db.GetSocialLinks()` en middleware Go | Heredarlas del layout Next y su integración existente |
| Contacto y WhatsApp | Número antiguo codificado por CTA | Usar `frontend/lib/copy/contact.ts` |
| Catálogo, carrito, cotización, reservaciones | No forman parte del cuerpo de Servicios | No migrar ni consultar |
| HTMX | Cargado por el layout, sin atributos HTMX en el cuerpo | No requerido |
| GSAP/ScrollTrigger | Usado para revelar elementos marcados | No requerido en la implementación inicial |
| PostgreSQL | Dependencia indirecta del layout Go | No requerido por el contenido de Servicios |

---

## C. Page structure

### Orden vertical propuesto

1. Layout compartido: `SiteHeader`.
2. Hero de Servicios.
3. Bodas.
4. Quinceañeras y celebraciones.
5. Bautizos y comuniones.
6. Eventos corporativos.
7. Graduaciones.
8. Fiestas privadas.
9. CTA final de cotización.
10. Layout compartido: `ContactStrip`, `SiteFooter` y `WhatsAppFab`.

El orden de los seis eventos conserva la página Go. No se adopta el orden diferente de Home —donde Graduaciones y Fiestas privadas aparecen antes que Corporativos— porque Servicios ya tiene una secuencia pública respaldada.

### Especificación por sección

| ID | Objetivo | Eyebrow | Título | Imagen y alt | CTA y enlace | Variante | Desktop | Móvil | Semántica y reutilización |
|---|---|---|---|---|---|---|---|---|---|
| `seccion-titulo` | Presentar el alcance del espacio y llevar a cotización | Servicios | Un lugar que eleva cualquier celebración. | `chenacolo_24.jpeg`, decorativa con `alt=""` al funcionar como fondo | Solicitar cotización → `/solicitar-cotizacion` | Hero oscuro con overlay cálido | Imagen a ancho completo; texto contenido y alineado a la izquierda | Altura estable, texto legible sin invadir header | Un único `h1`; nuevo `ServicesHero` |
| `seccion-bodas` | Detallar la oferta para bodas | Bodas | Bodas | `chenacolo_11.jpeg`; “Nave central de la capilla preparada para una ceremonia de boda” | Me interesa saber más → WhatsApp contextual | Clara | Dos columnas; imagen izquierda, contenido derecha | Copy primero e imagen después en flujo; una columna | `h2`, `EventDetailSection`, `CheckList` |
| `seccion-quinceaneras` | Detallar la oferta para XV años y celebraciones | XV años | Quinceañeras y celebraciones | `chenacolo_15.jpeg`; “Lámparas de cristal iluminadas en el salón Villa Chenacolo” | Me interesa saber más → WhatsApp contextual | Oscura | Dos columnas; imagen derecha, contenido izquierda | Copy primero e imagen después | `h2`, `EventDetailSection`, `CheckList` |
| `seccion-bautizos` | Detallar la oferta para ceremonias familiares | Bautizos | Bautizos y comuniones | `chenacolo_10.jpeg`; “Nave interior de la capilla de Villa Chenacolo” | Me interesa saber más → WhatsApp contextual | Clara | Dos columnas; imagen izquierda, contenido derecha | Copy primero e imagen después | `h2`, `EventDetailSection`, `CheckList` |
| `seccion-corporativos` | Detallar el uso profesional del espacio | Eventos profesionales | Eventos corporativos | `chenacolo_8.jpeg`; “Acceso principal, jardines y fachada de Villa Chenacolo” | Me interesa saber más → WhatsApp contextual | Oscura | Dos columnas; imagen derecha, contenido izquierda | Copy primero e imagen después | `h2`, `EventDetailSection`, `CheckList` |
| `seccion-graduaciones` | Detallar la oferta para graduaciones | Graduaciones | Graduaciones | `chenacolo_13.jpeg`; “Lámpara de cristal reflejada en los ventanales del salón” | Me interesa saber más → WhatsApp contextual | Clara | Dos columnas; imagen izquierda, contenido derecha | Copy primero e imagen después | `h2`, `EventDetailSection`, `CheckList` |
| `seccion-privadas` | Detallar usos privados respaldados | Celebraciones privadas | Fiestas privadas | `chenacolo_4.jpeg`; “Fachada y fuente de Villa Chenacolo” | Me interesa saber más → WhatsApp contextual | Oscura | Dos columnas; imagen derecha, contenido izquierda | Copy primero e imagen después | `h2`, `EventDetailSection`, `CheckList` |
| `seccion-cta` | Convertir a solicitud formal | Tu evento | Hablemos de tu evento. | Sin imagen nueva | Solicitar cotización → `/solicitar-cotizacion`; alternativa WhatsApp canónica | Destacada | Contenido centrado como bloque de acción, sin centrar párrafos globalmente | Botones apilables y targets táctiles | `h2`, nuevo `ServicesCta`; `SectionHeading` solo si su contrato permite el alineado requerido |

### Copy del hero

El párrafo se conserva literalmente como contenido respaldado:

> En Villa Chenacolo no organizamos tu evento: te entregamos el escenario perfecto para que lo vivas como lo imaginaste, sin complicaciones. Nuestra renta de salón incluye un espacio de alto nivel, con cada detalle pensado para brindar comodidad, elegancia y funcionalidad. Tú eliges a tus proveedores, nosotros ponemos lo mejor para recibirlos.

### Regla de composición

- El DOM mantiene primero el contenido textual y después la imagen no interactiva en cada servicio.
- En móvil, el orden visual coincide con el DOM.
- En escritorio, la imagen puede alternar mediante CSS. Como lo único reordenado es una imagen no interactiva, la secuencia de foco conserva el orden lógico de los CTA.
- La alternancia propuesta es una decisión nueva de Servicios para facilitar el escaneo; no se deriva automáticamente de Home.
- Los fondos claros y oscuros conservan el ritmo de la página Go.

---

## D. Hero

- **Tipo:** hero editorial con imagen estática de fondo; no video.
- **Asset:** `/assets/chenacolo_24.jpeg`.
- **Tratamiento:** `next/image` con `fill`, encuadre controlado, overlay cocoa cálido y gradiente suficiente para contraste. La imagen es decorativa porque el texto no depende de comprenderla.
- **Eyebrow:** “Servicios”.
- **H1:** “Un lugar que eleva cualquier celebración.”
- **Copy:** párrafo literal respaldado en la sección anterior.
- **CTA primario:** “Solicitar cotización” → `/solicitar-cotizacion`.
- **CTA secundario:** ninguno inicialmente. La página Go no respalda una segunda acción distinta y cada sección ya ofrece WhatsApp.
- **Header:** el hero empieza después del header sticky y opaco de rutas internas; no usa offsets negativos ni posicionamiento que lo coloque debajo del header.
- **LCP:** la imagen del hero es la única fotografía con `priority`; usa `sizes="100vw"` y un contenedor con altura mínima estable.
- **Reduced motion:** no requiere movimiento para aparecer ni para ser usable.
- **Sin JavaScript:** título, copy, imagen y CTA se renderizan desde el servidor y permanecen visibles.

El asset es vertical y puede sufrir recorte amplio en desktop. El encuadre debe validarse en las cuatro anchuras antes de aceptar la página; si el recorte oculta el motivo, cualquier cambio de asset requiere una decisión editorial explícita, no una sustitución silenciosa.

---

## E. Servicios o tipos de evento

### E.1 Bodas

- **ID:** `seccion-bodas`.
- **Orden:** 1 de 6.
- **Nombre canónico:** Bodas.
- **Descripción:** “El ‘sí’ más importante merece un lugar como este. Capilla íntima y elegante, salón cerrado con aire acondicionado, acústica impecable y áreas exteriores que se integran con armonía.”
- **Puntos:**
  - Renta de salón y capilla.
  - Espacio con capacidad ampliable (hasta 200 personas más al abrirse).
  - Climatización total.
  - Iluminación y acústica premium.
  - Cocina completamente equipada para tu proveedor de banquete.
  - Estacionamiento amplio dentro y fuera de la villa.
  - Personal de asistencia, valet y seguridad.
- **Nota:** “El mobiliario disponible se renta por separado.” Se elimina solo el emoji decorativo y “Todo”, tal como ya ocurre en Home.
- **Asset:** `/assets/chenacolo_11.jpeg`.
- **CTA:** “Me interesa saber más”.
- **Mensaje:** “¡Hola! Me gustaría conocer más sobre la planeación de bodas de Salón Chenacolo.”
- **Composición:** clara, imagen izquierda en desktop.

### E.2 Quinceañeras y celebraciones

- **ID:** `seccion-quinceaneras`.
- **Orden:** 2 de 6.
- **Nombre canónico:** Quinceañeras y celebraciones. Home puede conservar “XV años y celebraciones” como override editorial.
- **Descripción:** “Diseña la fiesta de tus sueños en un espacio que se transforma contigo. Desde temáticas juveniles hasta cenas formales, el salón se adapta a tu visión.”
- **Puntos:**
  - Salón climatizado y cerrado para eventos más íntimos o fiestas grandes.
  - Espacios exteriores: fuente, puente, jardín y área techada al aire libre para tematizar.
  - Cocina equipada lista para tu proveedor de catering.
  - Distribución funcional para áreas de foto, comida y pista de baile.
  - Personal en baños, seguridad y valet.
- **Asset:** `/assets/chenacolo_15.jpeg`.
- **CTA:** “Me interesa saber más”.
- **Mensaje:** “¡Hola! Me gustaría conocer más sobre las quinceañeras de Salón Chenacolo.”
- **Composición:** oscura, imagen derecha en desktop.

### E.3 Bautizos y comuniones

- **ID:** `seccion-bautizos`.
- **Orden:** 3 de 6.
- **Nombre canónico:** Bautizos y comuniones.
- **Descripción respaldada:** “Celebraciones con alma y buen gusto. La capilla es perfecta para ceremonias familiares, seguida de un evento en salón o jardín.”
- **Override editorial ya presente en Home:** “Celebraciones con alma y buen gusto. La capilla es perfecta para ceremonias familiares, seguidas de un evento en el salón o el jardín.” La segunda redacción corrige la concordancia sin cambiar la oferta y se recomienda como copy compartido.
- **Puntos:**
  - Renta de capilla y salón (opcional por separado).
  - Espacios ideales para fotos y reuniones familiares.
  - Cocina equipada para preparar o calentar alimentos.
  - Baños impecables con sala privada en el área de mujeres.
  - Atención personalizada y servicio discreto.
- **Asset:** `/assets/chenacolo_10.jpeg`.
- **CTA:** “Me interesa saber más”.
- **Mensaje:** “¡Hola! Me gustaría conocer más sobre los bautizos de Salón Chenacolo.”
- **Composición:** clara, imagen izquierda en desktop.

### E.4 Eventos corporativos

- **ID:** `seccion-corporativos`.
- **Orden:** 4 de 6.
- **Nombre canónico:** Eventos corporativos. Home puede conservar “Eventos empresariales” como override editorial.
- **Descripción:** “Un entorno elegante para eventos profesionales. Ya sea para una cena empresarial, una entrega de premios o un lanzamiento de producto, Chenacolo proyecta profesionalismo y alto nivel.”
- **Puntos:**
  - Salón con capacidad para grandes montajes.
  - Estacionamiento para equipo e invitados.
  - Apoyo en logística de acceso y seguridad.
  - Zona exterior techada para actividades complementarias.
  - Cocina y baños disponibles para todo el personal externo.
- **Asset:** `/assets/chenacolo_8.jpeg`.
- **CTA:** “Me interesa saber más”.
- **Mensaje:** “¡Hola! Me gustaría conocer más sobre los eventos corporativos de Salón Chenacolo.”
- **Composición:** oscura, imagen derecha en desktop.

### E.5 Graduaciones

- **ID:** `seccion-graduaciones`.
- **Orden:** 5 de 6.
- **Nombre canónico:** Graduaciones.
- **Descripción:** “Celebra logros en un lugar con estilo y amplitud. Un espacio seguro, bien distribuido y con todo lo necesario para una noche inolvidable.”
- **Puntos:**
  - Salón principal climatizado.
  - Zonas exteriores para fotos y áreas de espera.
  - Cocina lista para uso profesional.
  - Iluminación y acústica para ceremonia o fiesta.
  - Servicio continuo de limpieza en baños y apoyo logístico.
- **Asset:** `/assets/chenacolo_13.jpeg`.
- **CTA:** “Me interesa saber más”.
- **Mensaje:** “¡Hola! Me gustaría conocer más sobre las graduaciones de Salón Chenacolo.”
- **Composición:** clara, imagen izquierda en desktop.

### E.6 Fiestas privadas

- **ID:** `seccion-privadas`.
- **Orden:** 6 de 6.
- **Nombre canónico:** Fiestas privadas. Home puede conservar “Cumpleaños y fiestas privadas” como override editorial.
- **Descripción:** “De lo íntimo a lo espectacular. Celebra cumpleaños, aniversarios o reuniones familiares en un entorno que se siente privado, exclusivo y cuidado.”
- **Puntos:**
  - Espacios personalizables: salón, jardín y zona techada trasera.
  - Baños amplios y en constante mantenimiento.
  - Distribución arquitectónica que permite que el evento fluya sin interrupciones.
  - Privacidad gracias a la ubicación.
  - Personal discreto y atento en todo momento.
- **Asset:** `/assets/chenacolo_4.jpeg`.
- **CTA:** “Me interesa saber más”.
- **Mensaje:** “¡Hola! Me gustaría conocer más sobre las fiestas privadas de Salón Chenacolo.”
- **Composición:** oscura, imagen derecha en desktop.

### Decisión de reutilización

`CheckList` sí coincide con la necesidad: recibe una lista inmutable, ofrece semántica de lista y evita repetir iconos. `SectionHeading` sirve para encabezados `h2`, pero no para el hero porque su nivel está fijado. `EventSection` es visualmente cercano, pero hoy está acoplado a `EventSectionCopy` de Home y a decisiones de numeración, imagen, alineación y variante propias de esa página. Por ello no debe importarse directamente en Servicios. La recomendación es extraer un componente neutral compartido y conservar `EventSection` como adaptador de Home.

---

## F. Estrategia de copy compartido

### Duplicación encontrada

`frontend/lib/copy/home.ts` y `internal/templates/pages/services.templ` repiten:

- seis identidades de evento;
- descripciones casi iguales;
- listas de prestaciones;
- mensajes de WhatsApp;
- IDs de sección;
- una parte de los assets.

No coinciden en orden, títulos editoriales, numeración, imágenes ni todas las frases. Copiar nuevamente los objetos completos a un futuro archivo de Servicios crearía tres fuentes de verdad.

### Recomendación

Crear en una futura fase `frontend/lib/copy/events.ts` con el núcleo editorial compartido y permitir que cada página añada presentación y overrides explícitos.

#### Contenido de `events.ts`

- `EventId`, unión cerrada de las seis identidades.
- `EventCoreCopy`, tipo inmutable con descripción, puntos, nota opcional y mensaje de WhatsApp.
- `eventsById`, registro canónico sin orden de página.
- Helper que construya el enlace contextual desde `contact.whatsapp`; nunca contiene el número telefónico.

No deben entrar en el núcleo: número visible, eyebrow, título de campaña, orden, asset, variante, alineación o decisión de composición. Esas propiedades pertenecen a cada página.

#### Contenido que permanece en `home.ts`

- Todo el copy que no pertenece a los seis eventos.
- Orden Home: Bodas, XV años, Bautizos, Graduaciones, Privadas, Corporativos.
- Números `01` a `06`.
- Eyebrows y títulos editoriales de Home.
- Imágenes Home actuales.
- Alineaciones y variantes Home.
- Exportación `eventKinds` con el mismo contrato consumido actualmente, aunque se construya a partir de `eventsById`.

#### Contenido de `services.ts`

- Copy de hero y CTA final.
- Orden Go: Bodas, Quinceañeras, Bautizos, Corporativos, Graduaciones, Privadas.
- Títulos/eyebrows de Servicios.
- Imágenes Go de Servicios.
- Presentación, variante y alineación de cada bloque.
- Referencias por `EventId` al núcleo común.

### Prevención de regresiones en Home

La extracción debe ser mecánica y verificable: antes y después, `eventKinds` debe producir los mismos seis IDs, orden, mensajes, assets, variantes, títulos, descripciones y puntos que consume la Home. Solo se aceptan las pequeñas normalizaciones editoriales ya presentes en Home y enumeradas en este documento; no se reescribe copy durante la extracción.

### Pruebas requeridas

Crear `frontend/lib/copy/events.test.ts` con el test runner de Bun, sin dependencia nueva, para verificar:

- exactamente seis IDs únicos;
- registro completo sin IDs desconocidos;
- orden independiente de Home y Servicios;
- URLs WhatsApp derivadas de `contact.whatsapp`;
- mensajes codificados sin perder acentos;
- snapshot o igualdad estructural de `eventKinds` respecto de sus valores actuales;
- ninguna URL de contacto codificada en `events.ts`, `home.ts` o `services.ts`.

Esta refactorización no se realiza en Fase 4A.

---

## G. Componentes

### Existentes reutilizables

| Componente | Tipo | Responsabilidad en Servicios | Props/dependencias relevantes | Motivo |
|---|---|---|---|---|
| `frontend/components/shared/check-list.tsx` | Server Component | Mostrar prestaciones reales | Lista inmutable de strings; icono existente | Contrato coincide con los puntos de cada evento |
| `frontend/components/shared/section-heading.tsx` | Server Component | Encabezado del CTA final cuando el nivel `h2` coincida | Eyebrow, título y alineación existentes | Mantiene tipografía y jerarquía visual |
| `frontend/components/site/site-header.tsx` | Client Component ya existente | Navegación global y menú móvil | Copy de navegación y estados de menú | Se hereda del layout; no se duplica |
| `frontend/components/site/site-footer.tsx` | Server Component | Footer y redes | Datos sociales del layout | Se hereda del layout |
| `frontend/components/site/contact-strip.tsx` | Server Component | Franja de contacto | `contact.ts` | Se hereda del layout |
| `frontend/components/site/whatsapp-fab.tsx` | Server Component | Acceso flotante canónico | `contact.ts` | Se hereda con el responsive corregido en Fase 3C |

### Componentes por modificar en una fase futura

| Archivo | Cambio | Riesgo y control |
|---|---|---|
| `frontend/components/home/event-section.tsx` | Convertirlo en un adaptador del componente neutral compartido, preservando su API pública | Puede alterar Home; comparar DOM, copy, orden, responsive y anchors antes/después |

### Componentes nuevos propuestos

| Archivo | Tipo | Responsabilidad | Props principales | Dependencias |
|---|---|---|---|---|
| `frontend/components/shared/event-detail-section.tsx` | Server Component | Render neutral de copy, lista, imagen y CTA de un evento | `id`, `eyebrow`, `title`, `description`, `highlights`, `footnote`, `image`, `alignment`, `variant`, `cta` y número opcional | `next/image`, `CheckList`, tokens del sistema |
| `frontend/components/services/services-hero.tsx` | Server Component | Hero estático y LCP de Servicios | Copy del hero, asset y CTA | `next/image`, `Link`, utilidades del sistema |
| `frontend/components/services/services-cta.tsx` | Server Component | Cierre hacia cotización y WhatsApp | Título, copy, CTA principal y alternativa | `Link`, `contact.ts`, opcionalmente `SectionHeading` |

`event-detail-section.tsx` mantiene el texto antes de la imagen en el DOM; la presentación alterna únicamente en breakpoints de escritorio. Sus CTA tienen nombres accesibles contextualizados, por ejemplo `aria-label="Me interesa saber más sobre bodas por WhatsApp"`, aunque la etiqueta visible sea uniforme.

### Componentes que no deben reutilizarse

- `HomeHero`: depende del video, póster y narrativa exclusiva de Home.
- `OfferIntro`: presenta la oferta general, no el detalle de Servicios.
- `ContactSection`: incluye el formulario de Home y su integración; Servicios no debe duplicarlo.
- `ClosingCta`: si su contrato actual está acoplado a la Home, es preferible un cierre pequeño de Servicios antes que añadir condiciones. Puede reevaluarse comparando props durante implementación.
- `CatalogPreview` y `ProductCard`: no hay catálogo en Servicios.
- Componentes templ: pertenecen al frontend Go y no deben importarse ni traducirse literalmente.

No se propone ningún Client Component nuevo. La interactividad de header ya existe y WhatsApp/cotización son enlaces normales.

---

## H. Data contract

El contenido de Servicios es editorial y estático. Por tanto:

- no requiere endpoint nuevo;
- no requiere PostgreSQL;
- no requiere `fetch`;
- no requiere Route Handler de Next;
- no requiere proxy ni rewrite;
- no requiere CORS;
- no requiere cookies de carrito;
- no requiere estado de cliente.

El layout compartido puede seguir obteniendo redes mediante su integración actual con `/api/socials` y su fallback; eso no convierte los servicios en datos dinámicos. Los datos de teléfono, email y WhatsApp salen exclusivamente de `frontend/lib/copy/contact.ts`.

Contrato editorial propuesto:

```text
EventId
  -> EventCoreCopy
     - description
     - highlights[]
     - footnote?
     - whatsappMessage

EventId + presentación Home     -> EventSectionCopy existente
EventId + presentación Servicios -> ServiceSectionCopy futuro
```

No se crea una API para trasladar constantes del servidor de Next al navegador: la página y sus componentes son Server Components, y los enlaces resultantes se renderizan como HTML.

---

## I. Routing

- **Ruta pública:** `/servicios`.
- **Propietario actual:** Go, mediante `GET /servicios`.
- **Propietario futuro:** Next, mediante `frontend/app/(site)/servicios/page.tsx`.
- **Entrada desde navegación:** `frontend/lib/copy/nav.ts` ya apunta a `/servicios`.
- **Entrada desde Home:** la tarjeta de Servicios ya apunta a `/servicios`.
- **Anclas públicas:** se conservan `#seccion-bodas`, `#seccion-quinceaneras`, `#seccion-bautizos`, `#seccion-corporativos`, `#seccion-graduaciones` y `#seccion-privadas`.
- **Salida hacia cotización:** `/solicitar-cotizacion`, que sigue en Go.
- **Salida hacia WhatsApp:** enlace externo derivado de `contact.ts`.

### Cambio futuro de propietario

1. Construir y validar la página Next sin tocar el handler Go.
2. Probar `/servicios` en un entorno local controlado con el layout y las APIs ya existentes.
3. Configurar en la capa de entrada o despliegue que **solo** `/servicios` se dirija a Next.
4. Mantener el handler Go durante la ventana de validación como rollback inmediato.
5. Comprobar que el resto de rutas públicas, administrativas y APIs siguen dirigidas a Go.
6. Retirar o redirigir el handler Go únicamente en un plan posterior con aprobación explícita.

Este documento no elige ni implementa reverse proxy, rewrite o topología de producción. La ubicación de esa regla depende de la decisión de despliegue aún pendiente. No se permite un proxy global `/api/*`.

Los enlaces del Footer Next hacia secciones de Home continúan como están; no se redirigen automáticamente a Servicios porque constituyen navegación Home existente y cambiarla excede esta página.

---

## J. SEO y metadata

- **Title:** `Servicios`. Con el template actual resulta “Servicios · Villa Chenacolo”.
- **Description:** “Conoce las opciones de Villa Chenacolo para bodas, quinceañeras, bautizos, eventos corporativos, graduaciones y fiestas privadas.”
- **H1:** exactamente uno, en el hero.
- **H2:** un encabezado por tipo de evento y uno para el CTA final.
- **H3:** no es necesario en la estructura inicial.
- **Open Graph:** no se define una imagen específica inicialmente. El hero es vertical y no es un asset OG apropiado; se puede heredar metadata general sin inventar un recurso.
- **Canonical:** pendiente hasta disponer de una URL pública configurada.
- **No incluir:** dominio inventado, autor, fecha, año de experiencia, Twitter ni estadísticas.

El copy de metadata enumera únicamente tipos de evento presentes en la página real y evita afirmar organización integral de eventos.

---

## K. Accesibilidad

### Requisitos

- `<html lang="es">`, ya establecido por el layout Next.
- Un solo `h1`; cada servicio usa `h2`.
- DOM de cada bloque: encabezado, descripción, lista, nota, CTA e imagen no interactiva. El orden móvil coincide con el DOM.
- Los cambios visuales desktop no reordenan elementos interactivos entre sí.
- Alt text específico y comprobado; el hero usa alt vacío por ser fondo decorativo.
- Enlaces WhatsApp con etiqueta accesible contextual y `target="_blank" rel="noopener noreferrer"` si abren otra pestaña.
- CTA y controles con targets de al menos 44 × 44 CSS px.
- Focus visible conforme a `01-design-system.md`, sin `outline: none` sin reemplazo.
- Contraste de texto, listas, botones y overlays validado en variantes clara, oscura y destacada.
- Navegación completa por teclado; no se añade interacción personalizada.
- Skip link, menú móvil y focus trap heredados del layout Next deben permanecer intactos.
- `prefers-reduced-motion` respetado mediante CSS desde el primer render.
- Ningún contenido empieza en `opacity: 0`; JS y animaciones nunca desbloquean contenido esencial.
- La página completa permanece legible y accionable sin JavaScript, salvo la interactividad propia del menú móvil ya documentada por el layout.

### Defectos actuales que no deben migrarse

- El layout Go declara `lang="en"` para contenido en español.
- El hero y varias secciones empiezan invisibles y dependen de GSAP/ScrollTrigger.
- No existe alternativa reduced motion para esas animaciones.
- El alt del hero lo identifica como bar aunque muestra piezas ceremoniales; otros cuatro alts mencionan un árbol que no corresponde a la imagen.
- Los enlaces externos carecen de `rel="noopener noreferrer"`.
- Las etiquetas de CTA son repetitivas y poco contextuales fuera de su sección.
- El menú móvil Go no ofrece un focus trap completo, actualización coherente de estado ARIA ni restauración comprobada de foco.
- El frontend Go no aporta skip link.

---

## L. Responsive

### Reglas comunes

- Usar el contenedor principal y el espaciado vertical de `01-design-system.md`.
- No crear un contenedor estrecho de dos columnas en 1024 px. El Go actual activa grid en `lg` pero mantiene `max-w-2xl` hasta `xl`, lo que comprime ambas columnas entre 1024 y 1279 px.
- Todo medio usa `max-width: 100%`, dimensiones estables y contenedores que admiten encogimiento (`min-width: 0` donde corresponda).
- Ningún texto largo, CTA o URL puede ampliar el viewport.
- El header sigue mostrando el teléfono desde `lg`, no desde `md`.
- El FAB sigue visible debajo de `md`, oculto entre `md` y `lg`, y visible desde `lg`, tal como quedó estabilizado en Fase 3C.

### 360 px

- Hero de una columna, altura suficiente para texto y CTA sin quedar bajo el header.
- Padding lateral del contenedor del sistema; sin texto pegado a bordes.
- Secciones en una columna: copy y CTA antes de la imagen.
- Botón ancho disponible, con label envolviendo si fuera necesario.
- Imagen con proporción estable; no se fuerza una altura que recorte en exceso.
- Reservar espacio inferior para que el FAB no cubra el último CTA.

### 768 px

- Continúa una columna para bloques de evento si dos columnas reducen la legibilidad.
- Se amplían gaps y ancho de texto, sin superar la medida de lectura establecida.
- El teléfono del header permanece oculto y el FAB permanece oculto; no reintroducir el desborde corregido en Fase 3C.
- Menú móvil y focus trap continúan intactos.

### 1024 px

- Header de escritorio con teléfono visible.
- Secciones pasan a dos columnas usando el contenedor ancho real, no `max-w-2xl`.
- Gaps de grid se ajustan a tokens; ambas columnas admiten texto y lista sin compresión.
- Alternancia visual comienza aquí: Bodas izquierda; Quinceañeras derecha; Bautizos izquierda; Corporativos derecha; Graduaciones izquierda; Privadas derecha.
- FAB vuelve a mostrarse según el comportamiento estabilizado y no tapa CTA ni footer.

### 1440 px

- Contenedor principal alcanza su máximo del sistema, centrado.
- La medida de los párrafos permanece limitada; no crece hasta líneas excesivas.
- Imágenes conservan proporción y no se escalan más allá de su resolución útil sin necesidad.
- Gaps amplios, pero dentro de los tokens definidos; no aparecen vacíos arbitrarios.

### Regresiones responsive prohibidas

- `document.documentElement.scrollWidth` no supera `clientWidth` en ninguna anchura.
- No se modifica la combinación de visibilidad del teléfono/FAB de Fase 3C.
- ContactStrip y Footer mantienen su layout actual.
- El menú móvil conserva apertura, cierre por Escape, focus trap, restauración de foco y scroll lock.

---

## M. Performance

- **Cantidad inicial:** siete fotografías: una de hero y seis de eventos.
- **Priority:** únicamente `chenacolo_24.jpeg`, candidata LCP.
- **Resto:** carga diferida predeterminada de `next/image`; nunca `priority` masivo.
- **Sizes hero:** `100vw`.
- **Sizes secciones:** reflejar una columna hasta `lg` y aproximadamente media anchura de contenedor desde `lg`, no `100vw` incondicional.
- **Dimensiones:** usar los tamaños reales del inventario para calcular relación de aspecto y reservar espacio.
- **CLS:** altura mínima estable del hero, wrappers con `aspect-ratio`, fuentes de `next/font` y botones con geometría estable.
- **Recorte:** `object-fit` y `object-position` se validan por asset; no sustituyen dimensiones intrínsecas.
- **Server Components:** página, hero, bloques y CTA son Server Components por defecto.
- **JavaScript:** no se añade bundle de página; solo permanece el necesario para el layout compartido.
- **Sin video:** Servicios no duplica el video LCP/decorativo de Home.
- **Sin fetch de contenido:** el copy se resuelve en servidor desde módulos locales.

Se debe comprobar la salida de `next build` y la red del navegador para confirmar que no hay assets 404, imágenes duplicadas ni carga eager de las seis imágenes secundarias.

---

## N. Motion

### Implementación inicial

- Sin Framer Motion.
- Sin GSAP ni ScrollTrigger.
- Contenido visible desde el HTML inicial.
- Solo transiciones CSS no esenciales ya admitidas por el sistema, como hover/focus de enlaces y botones.
- `prefers-reduced-motion` de `globals.css` anula o reduce esas transiciones.

### Mejora futura, bloqueada por Fase 1B

Después de aprobar e instalar Framer Motion de forma segura, podrá evaluarse:

- `frontend/lib/motion/transitions.ts`;
- `frontend/components/shared/motion/reveal-on-view.tsx`;
- `frontend/components/shared/motion/stagger-group.tsx`;
- `frontend/components/shared/motion/fade-in.tsx`.

Servicios podría envolver encabezados o bloques como mejora progresiva. La ausencia, fallo o reducción de movimiento nunca cambia visibilidad, orden, semántica ni acceso a CTA. No se añade ningún import condicional ni placeholder de Framer en la implementación inicial.

---

## O. Out of scope

- No migrar el catálogo.
- No migrar ni reimplementar la cotización completa.
- No migrar reservaciones.
- No modificar el panel administrativo.
- No crear carrito ni portar la cookie `cart_id`.
- No crear endpoints innecesarios.
- No modificar PostgreSQL, esquemas, migraciones ni `internal/db/`.
- No agregar CORS, proxy global o rewrite global.
- No instalar ni implementar Framer Motion en esta fase.
- No cambiar otras rutas Go o Next.
- No eliminar el handler o template Go antes del cambio controlado de propietario.
- No incorporar formularios, catálogo, testimonios, precios, paquetes, estadísticas o capacidades nuevas.
- No agregar video, galería, redes sociales propias de la página ni datos del salón no presentes.
- No corregir en Fase 4A los defectos del frontend Go.
- No modificar Home durante la auditoría.

---

## P. Implementation plan

### Fase P1 — Copy compartido y tipos

**Objetivo:** establecer una fuente única para el núcleo de los seis eventos sin alterar la salida actual de Home.

**Crear:**

- `frontend/lib/copy/events.ts`
- `frontend/lib/copy/services.ts`
- `frontend/lib/copy/events.test.ts`

**Modificar:**

- `frontend/lib/copy/home.ts`

**Riesgos:** cambios accidentales de orden, copy, mensajes de WhatsApp, assets o props de Home; introducción de una URL telefónica duplicada; tipos demasiado acoplados a una sola página.

**Validaciones:** test unitario de invariantes; búsqueda de números codificados; `bun run lint`; `bun run build`; comparación estructural de `eventKinds` antes/después.

**Criterio de terminación:** los seis núcleos existen una sola vez, Home produce el mismo contrato visible y Servicios dispone de presentación propia en el orden Go.

### Fase P2 — Componentes de Servicios

**Objetivo:** extraer un bloque neutral de evento y crear hero/cierre sin Client Components nuevos.

**Crear:**

- `frontend/components/shared/event-detail-section.tsx`
- `frontend/components/services/services-hero.tsx`
- `frontend/components/services/services-cta.tsx`

**Modificar:**

- `frontend/components/home/event-section.tsx`

**Riesgos:** regresión visual o semántica de Home; orden DOM/visual inconsistente; alts o `sizes` incorrectos; CTA sin contexto accesible.

**Validaciones:** lint/build, render SSR de Home, inspección de markup, teclado, focus, alts, enlaces y comparativa responsive de Home.

**Criterio de terminación:** el adaptador conserva Home y los componentes nuevos renderizan toda la información sin JS cliente.

### Fase P3 — Página y metadata

**Objetivo:** componer `/servicios` en Next con hero, seis eventos y CTA final.

**Crear:**

- `frontend/app/(site)/servicios/page.tsx`

**Modificar:** ninguno previsto.

**Riesgos:** colisión conceptual con la ruta Go en el entorno de ejecución; IDs duplicados; más de un `h1`; metadata no respaldada.

**Validaciones:** build, HTML SSR, metadata, un H1, IDs únicos, enlaces internos y ausencia de fetch de contenido.

**Criterio de terminación:** la ruta Next compila y entrega la estructura completa con contenido real, sin asumir todavía tráfico de producción.

### Fase P4 — Integración con layout

**Objetivo:** confirmar que la página hereda correctamente Header, ContactStrip, Footer, redes y FAB del grupo `(site)`.

**Crear:** ninguno.

**Modificar:** ninguno previsto. Si aparece un defecto objetivo del layout, debe documentarse y solicitar aprobación para una corrección mínima separada.

**Riesgos:** duplicar layout dentro de la página; alterar las reglas responsive estabilizadas; dependencia no controlada de `/api/socials`.

**Validaciones:** backend disponible y apagado, navegación, menú móvil, footer/fallback de redes, FAB y ausencia de filtración de `GO_API_BASE_URL`.

**Criterio de terminación:** Servicios usa un solo layout y los fallos de redes no impiden renderizar el contenido editorial.

### Fase P5 — Responsive y accesibilidad

**Objetivo:** cerrar los estados 360/768/1024/1440 y la experiencia asistiva.

**Crear:** ninguno previsto.

**Modificar, solo si las validaciones lo requieren:**

- `frontend/components/shared/event-detail-section.tsx`
- `frontend/components/services/services-hero.tsx`
- `frontend/components/services/services-cta.tsx`
- `frontend/app/(site)/servicios/page.tsx`

**Riesgos:** scroll horizontal, recorte incorrecto, medida de texto excesiva, FAB sobre CTA, reintroducción del desborde del header.

**Validaciones:** las cuatro anchuras, scrollWidth/clientWidth, teclado, foco, reduced motion, JS deshabilitado, headings, anchors, contraste y targets táctiles.

**Criterio de terminación:** cero desborde, jerarquía semántica correcta y contenido/CTA utilizables en todos los estados.

### Fase P6 — QA y cambio de propietario de `/servicios`

**Objetivo:** validar regresiones del sistema y preparar una conmutación aislada de la ruta.

**Crear:** ninguno en el repositorio previsto.

**Modificar:** ninguna ruta Go durante QA. La configuración externa de entrada se define solo cuando la topología de despliegue esté decidida y aprobada.

**Riesgos:** enviar otras rutas a Next; perder rollback Go; romper `/solicitar-cotizacion`; confundir dos procesos que reclaman el mismo path.

**Validaciones:** suites Go/Next, APIs existentes, enlaces desde Home/nav, respuesta real de `/servicios`, rollback y comprobación de que panel/catálogo/carrito/cotización/reservaciones siguen en Go.

**Criterio de terminación:** existe una regla de propietario exacta para `/servicios`, observabilidad básica y rollback; la ruta está lista para asumir tráfico sin ampliar el alcance.

### Fase P7 — Motion futura pendiente

**Objetivo:** añadir mejora progresiva solamente después de aprobar Fase 1B.

**Crear:**

- `frontend/lib/motion/transitions.ts`
- `frontend/components/shared/motion/reveal-on-view.tsx`
- `frontend/components/shared/motion/stagger-group.tsx`
- `frontend/components/shared/motion/fade-in.tsx`

**Modificar:** `package.json`, `bun.lock` y, solo si se aprueba, los componentes de Servicios que adopten wrappers no esenciales.

**Riesgos:** contenido oculto, hidratación innecesaria, bundle mayor y movimiento pese a preferencia reducida.

**Validaciones:** movimiento habilitado/reducido, JS deshabilitado, rendimiento, hidratación y contenido siempre visible.

**Criterio de terminación:** la mejora es opcional, respeta reduced motion y su eliminación no cambia la funcionalidad. Esta fase permanece bloqueada hasta aprobación independiente.

---

## Q. Testing plan

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

Si se agrega `events.test.ts`, ejecutar además el script de test que se incorpore mediante una modificación aprobada de `package.json`, o directamente `bun test lib/copy/events.test.ts` sin instalar otra dependencia.

### Validación funcional

- `/servicios` responde HTTP 200 desde Next en el entorno de prueba.
- Header, nav, Home y footer enlazan sin 404.
- `/solicitar-cotizacion` continúa siendo propiedad de Go y abre correctamente.
- Los seis enlaces WhatsApp usan el número de `contact.ts`, incluyen el mensaje correspondiente y no exponen otra URL interna.
- Con Go disponible, redes y layout funcionan.
- Con Go apagado, el fallback de redes no impide leer Servicios.
- No aparecen consultas de catálogo, carrito, cotizaciones o reservaciones.
- No se crea cookie desde el cuerpo de Servicios.

### Validación visual y responsive

Revisar 360, 768, 1024 y 1440 px:

- `document.documentElement.scrollWidth <= document.documentElement.clientWidth`;
- cero scroll horizontal y cero desborde del header;
- un solo `h1`;
- IDs únicos;
- anchors correctos y no ocultos por el header;
- header y footer intactos;
- menú móvil, focus trap, Escape, restauración de foco y scroll lock intactos;
- teléfono visible desde `lg`;
- FAB sin cubrir contenido y oculto entre `md` y `lg`;
- ContactStrip intacto;
- imágenes con encuadre correcto y sin 404;
- copy y listas legibles, sin truncamiento;
- alternancia visual y orden móvil según este documento.

### Accesibilidad y robustez

- Recorrer todos los enlaces por teclado.
- Comprobar focus visible sobre cada CTA.
- Confirmar targets táctiles mínimos.
- Probar `prefers-reduced-motion: reduce`.
- Deshabilitar JavaScript y confirmar contenido, imágenes, CTA y jerarquía.
- Auditar alts, nombres accesibles y enlaces externos seguros.
- Confirmar ausencia de errores relevantes en consola e hidratación.
- Confirmar que no se filtra `GO_API_BASE_URL`.

### Rendimiento

- Identificar `chenacolo_24.jpeg` como candidata LCP.
- Confirmar que solo el hero usa `priority`.
- Confirmar carga diferida de las otras seis imágenes.
- Revisar `sizes`, proporciones, CLS y transferencias duplicadas.
- Confirmar que Servicios no añade Client Components ni librerías de movimiento.

---

## R. Acceptance criteria

- [ ] `/servicios` contiene hero, seis eventos y CTA final en el orden Go documentado.
- [ ] Todo el contenido comercial proviene de la página Go o de normalizaciones ya presentes en Home.
- [ ] No se agregaron paquetes, precios, testimonios, estadísticas, fechas, capacidades ni servicios inventados.
- [ ] Se usan exactamente los siete assets documentados, con alts correctos y sin 404.
- [ ] El copy base de los seis eventos está centralizado sin duplicación entre Home y Servicios.
- [ ] Home conserva orden, presentación, enlaces, copy visible y comportamiento actuales.
- [ ] Teléfono y WhatsApp salen únicamente de `frontend/lib/copy/contact.ts`.
- [ ] Cada enlace WhatsApp contiene el mensaje contextual correcto y es seguro como enlace externo.
- [ ] La página usa un solo `h1`, IDs únicos y jerarquía `h2` coherente.
- [ ] La página funciona con teclado, focus visible, reduced motion y JavaScript deshabilitado.
- [ ] No hay scroll horizontal a 360, 768, 1024 o 1440 px.
- [ ] No regresan el desborde del header ni las reglas de teléfono/FAB de Fase 3C.
- [ ] Header, menú móvil, ContactStrip, Footer y FAB permanecen intactos.
- [ ] Hero y seis imágenes reservan espacio; solo la candidata LCP usa `priority`.
- [ ] No se añade endpoint, consulta PostgreSQL, fetch de contenido, CORS, proxy o rewrite innecesario.
- [ ] No se modifica panel, catálogo, carrito, cotización ni reservaciones.
- [ ] Las pruebas Go pasan con `go test -mod=vendor . ./cmd/... ./internal/...`.
- [ ] `bun install --frozen-lockfile`, `bun run lint` y `bun run build` pasan.
- [ ] `/servicios` está lista para asumir tráfico mediante una regla aislada, con el handler Go conservado como rollback hasta una aprobación posterior.
- [ ] Fase 1B y Motion continúan pendientes y separadas.

---

## Auditoría consolidada y decisiones pendientes

### Funcionalidades que deben mantenerse

- Los seis tipos de evento, su oferta verificable, su orden de Servicios y sus anclas.
- La posibilidad de pedir información por WhatsApp con contexto.
- La ruta de solicitud de cotización.
- Header, navegación, footer y redes del layout compartido.
- Responsive de una a dos columnas, con fondos alternos.
- Imágenes estáticas reales asociadas a cada bloque.

### Funcionalidades que pueden simplificarse

- Retirar de Servicios HTMX, extensiones, toaster, GSAP y ScrollTrigger porque el cuerpo no los necesita.
- Sustituir seis URLs WhatsApp codificadas por un helper y `contact.ts`.
- Unificar etiquetas visibles de los CTA y aportar contexto accesible.
- Eliminar la dependencia de animación para revelar contenido.
- No reproducir en Next la cookie de carrito introducida por el middleware Go.

### Problemas y riesgos actuales

1. Número WhatsApp Go distinto al canónico de `contact.ts`.
2. Alt text incorrecto o genérico en cinco de siete imágenes.
3. Contenido inicialmente invisible y dependiente de JS, sin reduced motion.
4. `lang="en"` en una página española.
5. Enlaces `_blank` sin relación segura.
6. CTA repetitivos sin nombre contextual autónomo.
7. Grid activado en 1024 px dentro de un contenedor `max-w-2xl` hasta 1280 px.
8. Imágenes sin dimensiones declaradas, estrategia LCP o lazy loading explícita.
9. Dependencias globales HTMX/GSAP/toaster no utilizadas por el cuerpo.
10. Cookie de carrito y consulta social acopladas por middleware a una página editorial.
11. `overflow-x-hidden` del layout Go puede ocultar un desborde en lugar de resolverlo.
12. La imagen vertical del hero requiere validar recorte a 1440 px.

### Duplicación y contradicciones

- Home y Servicios comparten eventos, pero difieren en orden, assets y títulos; el núcleo debe compartirse y la presentación permanecer por página.
- Home llama “XV años”, “Eventos empresariales” y “Cumpleaños y fiestas privadas”; Go usa “Quinceañeras”, “Eventos corporativos” y “Fiestas privadas”. Son overrides editoriales, no nuevos servicios.
- Home ya mejora varias frases de listas y concordancia. Se recomiendan esas normalizaciones cuando no cambian la promesa.
- El hero de Servicios afirma expresamente que Villa Chenacolo **no organiza** el evento y que el cliente elige proveedores. `frontend/lib/copy/contact.ts` afirma “servicios de planificación de eventos excepcionales”. Ambas promesas entran en tensión. Hasta decisión comercial, Servicios debe usar el copy respaldado de renta de espacio y no ampliar la afirmación de planificación.
- Las descripciones de `00-project-setup.md` sobre rutas absolutas, CORS y variables públicas están superadas por la implementación: hoy se usa `GO_API_BASE_URL` privada, Route Handlers específicos y ningún proxy/CORS global.
- `01-design-system.md` describe componentes de Motion que siguen pendientes de Fase 1B; Servicios inicial no puede depender de ellos.
- `02-shared-layout.md` indicaba otros cortes para teléfono/FAB; prevalece el código estabilizado en Fase 3C: teléfono desde `lg` y FAB oculto entre `md` y `lg`.
- `03-home-page.md` contiene partes descriptivas que ya no reflejan exactamente el código final; para Home, `frontend/lib/copy/home.ts` y los componentes activos son la referencia de regresión.

### Decisiones pendientes antes del cambio de propietario

1. Confirmar que el copy canónico debe evitar “planificación de eventos” mientras el hero niega que se organice el evento.
2. Aprobar las normalizaciones editoriales de Home como núcleo compartido cuando difieren levemente del templ.
3. Confirmar la alternancia desktop propuesta para Servicios frente a la composición actual Go con imagen siempre a la izquierda.
4. Validar el recorte de `chenacolo_24.jpeg` o aprobar otro asset respaldado si no funciona como hero amplio.
5. Definir dónde vive la regla de enrutamiento que entregará solo `/servicios` a Next.
6. Definir URL pública antes de añadir canonical u Open Graph específicos.
7. Decidir cuándo retirar o redirigir la ruta Go después de una ventana de rollback.
8. Mantener Fase 1B separada hasta resolver la instalación segura de Framer Motion.

### Resultado de Fase 4A

Este documento es la fuente de verdad para la futura implementación de Servicios. En Fase 4A no se implementa la página, no se refactoriza Home, no se crean componentes o endpoints, no se cambia el propietario de `/servicios` y no se modifica el sistema Go.
