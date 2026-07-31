/**
 * Home page copy. Centralized here so it's easy to scan, edit, and (later)
 * extract into an i18n catalog.
 *
 * `eventKinds` mirrors the original six sections on the home page. The image
 * paths are relative to `/assets/` (which is the migrated Go static folder).
 * The `whatsapp` URLs use the original pre-filled messages so we don't break
 * any existing conversations in flight.
 */

import { whatsappHref } from "./contact";

export const hero = {
  eyebrow: "Sala de Acontecimientos Especiales",
  title: "Donde los momentos importantes se vuelven recuerdos.",
  primaryCta: { label: "Solicitar cotización", href: "/solicitar-cotizacion" },
  secondaryCta: { label: "Ver la experiencia",   href: "/#seccion-galeria" },
} as const;

export const offerIntro = {
  eyebrow: "Nuestra oferta",
  title: "Un lugar que eleva cualquier celebración.",
  italicWord: "eleva",
  lede:
    "En Villa Chenacolo no organizamos tu evento: te entregamos el escenario perfecto para que lo vivas como lo imaginaste, sin complicaciones. Nuestra renta de salón incluye un espacio de alto nivel, con cada detalle pensado para brindar comodidad, elegancia y funcionalidad. Tú eliges a tus proveedores, nosotros ponemos lo mejor para recibirlos.",
  cards: [
    {
      title: "Servicios",
      lede:
        "Para bodas, bautizos, graduaciones, quinceañeras. Te ayudamos con todo para que tu experiencia en Villa Chenacolo sea inolvidable.",
      href: "/servicios",
      cta: "Conoce nuestros servicios",
      image: "/assets/chenacolo-st-2.jpg",
    },
    {
      title: "Catálogo",
      lede:
        "Elige entre una selección de mobiliario y piezas decorativas que combinan con el estilo del lugar y transforman cualquier montaje en una experiencia visual y funcional.",
      href: "/catalogo",
      cta: "Conoce nuestro catálogo",
      image: "/assets/chenacolo_18.jpeg",
    },
    {
      title: "Experiencia",
      lede:
        "Capilla, salón, jardines, cocina y mucho más. Todo conectado en un entorno privado, elegante y armonioso.",
      href: "/experiencia",
      cta: "Conoce nuestro salón",
      image: "/assets/chenacolo-st-5.jpg",
    },
  ],
} as const;

export type EventKindCopy = {
  id: "bodas" | "quinceaneras" | "bautizos" | "corporativos" | "graduaciones" | "privadas";
  num: "01" | "02" | "03" | "04" | "05" | "06";
  title: string;
  image: string;
  /** When true, render the section on a dark cocoa background. */
  dark: boolean;
  whatsapp: string;
  lede: string;
  /** Bullet list of inclusions/features. */
  items: readonly string[];
  /** Optional footnote (e.g. "el mobiliario se renta por separado"). */
  footnote?: string;
  /** Caption used for the `alt` attribute of the section image. */
  imageAlt: string;
};

export const eventKinds: readonly EventKindCopy[] = [
  {
    id: "bodas",
    num: "01",
    title: "Bodas",
    image: "/assets/chenacolo_11.jpeg",
    imageAlt: "Recibidor del salón de eventos Villa Chenacolo",
    dark: false,
    whatsapp: whatsappHref(
      "Hola! Me gustaría conocer más sobre la planeación de bodas en Salón Chenacolo.",
    ),
    lede:
      "El ‘sí’ más importante merece un lugar como este. Capilla íntima y elegante, salón cerrado con aire acondicionado, acústica impecable y áreas exteriores que se integran con armonía.",
    items: [
      "Renta de salón y capilla",
      "Espacio con capacidad ampliable (hasta 200 personas más al abrirse)",
      "Climatización total",
      "Iluminación y acústica premium",
      "Cocina completamente equipada para tu proveedor de banquete",
      "Estacionamiento amplio dentro y fuera de la villa",
      "Personal de asistencia, valet y seguridad",
    ],
    footnote: "Todo el mobiliario disponible se renta por separado.",
  },
  {
    id: "quinceaneras",
    num: "02",
    title: "Quinceañeras y Celebraciones",
    image: "/assets/chenacolo-st-7.jpg",
    imageAlt: "Jardín del salón de eventos al atardecer",
    dark: true,
    whatsapp: whatsappHref(
      "Hola! Me gustaría conocer más sobre las quinceañeras en Salón Chenacolo.",
    ),
    lede:
      "Diseña la fiesta de tus sueños en un espacio que se transforma contigo. Desde temáticas juveniles hasta cenas formales, el salón se adapta a tu visión.",
    items: [
      "Salón climatizado y cerrado para eventos íntimos o fiestas grandes",
      "Espacios exteriores: fuente, puente, jardín y área techada al aire libre",
      "Cocina equipada lista para tu proveedor de catering",
      "Distribución funcional para áreas de foto, comida y pista de baile",
      "Personal en baños, seguridad y valet",
    ],
  },
  {
    id: "bautizos",
    num: "03",
    title: "Bautizos y Comuniones",
    image: "/assets/chenacolo-st-9.jpg",
    imageAlt: "Capilla del salón lista para ceremonia íntima",
    dark: false,
    whatsapp: whatsappHref(
      "Hola! Me gustaría conocer más sobre los bautizos en Salón Chenacolo.",
    ),
    lede:
      "Celebraciones con alma y buen gusto. La capilla es perfecta para ceremonias familiares, seguida de un evento en salón o jardín.",
    items: [
      "Renta de capilla y salón (opcional por separado)",
      "Espacios ideales para fotos y reuniones familiares",
      "Cocina equipada para preparar o calentar alimentos",
      "Baños impecables con sala privada en el área de mujeres",
      "Atención personalizada y servicio discreto",
    ],
  },
  {
    id: "corporativos",
    num: "04",
    title: "Eventos corporativos",
    image: "/assets/chenacolo-st-4.png",
    imageAlt: "Salón principal preparado para montaje corporativo",
    dark: true,
    whatsapp: whatsappHref(
      "Hola! Me gustaría conocer más sobre los eventos corporativos en Salón Chenacolo.",
    ),
    lede:
      "Un entorno elegante para eventos profesionales. Ya sea para una cena empresarial, una entrega de premios o un lanzamiento de producto, Chenacolo proyecta profesionalismo y alto nivel.",
    items: [
      "Salón con capacidad para grandes montajes",
      "Estacionamiento para equipo e invitados",
      "Apoyo en logística de acceso y seguridad",
      "Zona exterior techada para actividades complementarias",
      "Cocina y baños disponibles para todo el staff externo",
    ],
  },
  {
    id: "graduaciones",
    num: "05",
    title: "Graduaciones",
    image: "/assets/chenacolo-st-5.jpg",
    imageAlt: "Área social del salón iluminada para graduación",
    dark: false,
    whatsapp: whatsappHref(
      "Hola! Me gustaría conocer más sobre las graduaciones en Salón Chenacolo.",
    ),
    lede:
      "Celebra logros en un lugar con estilo y amplitud. Un espacio seguro, bien distribuido y con todo lo necesario para una noche inolvidable.",
    items: [
      "Salón principal climatizado",
      "Zonas exteriores para fotos y áreas de espera",
      "Cocina lista para uso profesional",
      "Iluminación y acústica perfectas para ceremonia o fiesta",
      "Servicio continuo de limpieza en baños y apoyo logístico",
    ],
  },
  {
    id: "privadas",
    num: "06",
    title: "Fiestas privadas",
    image: "/assets/chenacolo-st-3.jpg",
    imageAlt: "Salón ambientado para fiesta privada nocturna",
    dark: true,
    whatsapp: whatsappHref(
      "Hola! Me gustaría conocer más sobre las fiestas privadas en Salón Chenacolo.",
    ),
    lede:
      "De lo íntimo a lo espectacular. Celebra cumpleaños, aniversarios o reuniones familiares en un entorno que se siente privado, exclusivo y cuidado.",
    items: [
      "Espacios personalizables (salón, jardín, zona techada trasera)",
      "Baños amplios y en constante mantenimiento",
      "Distribución arquitectónica que hace fluir el evento sin interrupciones",
      "Privacidad total gracias a la ubicación",
      "Personal discreto y atento en todo momento",
    ],
  },
] as const;

export const catalogPreview = {
  eyebrow: "Catálogo",
  title: "Cada detalle, pensado para tu evento.",
  italicWord: "pensado",
  lede:
    "Tenemos una gran repertorio de mesas, copas y otros utensilios para brindarte la experiencia perfecta.",
} as const;

export const galleryPreview = {
  eyebrow: "Galería",
  title: "Queremos presumirte.",
  lede:
    "Entra y descubre el estilo y la elegancia de nuestro salón. Te va a encantar.",
  cta: { label: "Ver la galería completa", href: "/experiencia" },
  images: [
    { src: "/assets/chenacolo_25.jpeg", alt: "Área de bar del salón" },
    { src: "/assets/chenacolo_30.jpeg", alt: "Mesa principal decorada" },
    { src: "/assets/chenacolo_31.jpeg", alt: "Salón iluminado al atardecer" },
    { src: "/assets/chenacolo_23.jpeg", alt: "Detalle del área lounge" },
    { src: "/assets/chenacolo_18.jpeg", alt: "Montaje del comedor" },
    { src: "/assets/chenacolo_11.jpeg", alt: "Recibidor del salón" },
  ],
} as const;

export const aboutStrip = {
  eyebrow: "Quiénes somos",
  paragraphs: [
    "Ofrecemos un espacio de alto nivel, para celebrar momentos únicos con libertad, privacidad y estilo.",
    "Cada detalle arquitectónico y cada servicio están pensados para que tu evento fluya a la perfección.",
  ],
} as const;

export const contactSection = {
  eyebrow: "Hablemos",
  title: "De tu evento.",
  lede:
    "Si deseas conocer la disponibilidad, precios o agendar una visita, déjanos tus datos. Te contactaremos lo antes posible.",
  form: {
    title: "Queremos ayudarte a crear algo inolvidable",
    subtitle: "Uno de nuestros asesores te dará seguimiento directo.",
    submit: "Solicitar contacto",
    success: "¡Se ha enviado tu solicitud!",
    errorGeneric: "No pudimos enviar tu solicitud. Inténtalo de nuevo.",
  },
  info: {
    title: "Encuéntranos",
    subtitle:
      "Estos son todos los medios por los que puedes contactarnos.",
    quoteCta: { label: "O solicita una cotización", href: "/solicitar-cotizacion" },
  },
} as const;

export const closingCta = {
  eyebrow: "Estás listo",
  title: "Hablemos de tu evento.",
  cta: { label: "Solicitar cotización", href: "/solicitar-cotizacion" },
  /** WhatsApp href comes from `lib/copy/contact.ts` so the label stays in sync. */
  secondary: { label: "o escríbenos por WhatsApp" },
} as const;
