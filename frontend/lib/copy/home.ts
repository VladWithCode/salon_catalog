import { contact } from "@/lib/copy/contact";
import { eventsByKey, type EventKey } from "@/lib/copy/events";
import type { ImageAsset } from "@/lib/types";

export type HomeLink = Readonly<{
  label: string;
  href: string;
}>;

export type OfferCardCopy = Readonly<{
  eyebrow: string;
  title: string;
  description: string;
  image: ImageAsset;
  link: HomeLink;
}>;

export type EventSectionCopy = Readonly<{
  id: string;
  number: string;
  eyebrow: string;
  title: string;
  description: string;
  highlights: readonly string[];
  footnote?: string;
  image: ImageAsset;
  alignment: "image-left" | "image-right";
  variant: "light" | "dark";
  cta: HomeLink;
}>;

export type GalleryImageCopy = ImageAsset &
  Readonly<{
    layout: "tall" | "wide" | "square";
  }>;

function whatsappLink(message: string): string {
  const url = new URL(contact.whatsapp);
  url.searchParams.set("text", message);
  return url.toString();
}

function sharedEventCopy(
  key: EventKey,
): Pick<EventSectionCopy, "description" | "highlights" | "footnote"> {
  const event = eventsByKey[key];
  const footnote = "footnote" in event ? event.footnote : undefined;

  return {
    description: event.description,
    highlights: event.highlights,
    ...(footnote === undefined ? {} : { footnote }),
  };
}

function eventWhatsappLink(key: EventKey): string {
  return whatsappLink(eventsByKey[key].whatsappMessage);
}

export const homeCopy = {
  hero: {
    eyebrow: "Sala de acontecimientos especiales",
    title: "Donde los momentos importantes se vuelven recuerdos.",
    intro:
      "En Villa Chenacolo no organizamos tu evento: te entregamos el escenario perfecto para que lo vivas como lo imaginaste, sin complicaciones.",
    primaryCta: {
      label: "Solicitar cotización",
      href: "/solicitar-cotizacion",
    },
    secondaryCta: {
      label: "Ver la experiencia",
      href: "#experiencia",
    },
    poster: {
      src: "/assets/chenacolo_3.jpeg",
      alt: "Lámparas de cristal iluminadas en el salón Villa Chenacolo",
      width: 852,
      height: 1280,
    },
    video: {
      webm: "/assets/chenacolo_vid.webm",
      mp4: "/assets/chenacolo_vid.mp4",
    },
    wordmark: {
      src: "/assets/logo_name_white.png",
      alt: "Villa Chenacolo, sala de acontecimientos especiales",
      width: 512,
      height: 141,
    },
  },
  offer: {
    eyebrow: "Nuestra oferta",
    titleBefore: "Un lugar que ",
    titleEmphasis: "eleva",
    titleAfter: " cualquier celebración.",
    lede:
      "En Villa Chenacolo no organizamos tu evento: te entregamos el escenario perfecto para que lo vivas como lo imaginaste, sin complicaciones. Nuestra renta de salón incluye un espacio de alto nivel, con cada detalle pensado para brindar comodidad, elegancia y funcionalidad. Tú eliges a tus proveedores, nosotros ponemos lo mejor para recibirlos.",
    cards: [
      {
        eyebrow: "Servicios",
        title: "¿Cómo podemos ayudarte?",
        description:
          "Para bodas, bautizos, graduaciones y quinceañeras. Te ayudamos con todo para que tu experiencia en Villa Chenacolo sea inolvidable.",
        image: {
          src: "/assets/chenacolo-st-2.jpg",
          alt: "Acceso principal, jardines y fachada de Villa Chenacolo",
          width: 1920,
          height: 1280,
        },
        link: { label: "Conoce más", href: "/servicios" },
      },
      {
        eyebrow: "Catálogo",
        title: "Mobiliario y piezas decorativas",
        description:
          "Elige entre una selección de mobiliario y piezas decorativas que combinan con el estilo del lugar y transforman cualquier montaje en una experiencia visual y funcional.",
        image: {
          src: "/assets/chenacolo_18.jpeg",
          alt: "Piezas ceremoniales doradas disponibles en Villa Chenacolo",
          width: 1280,
          height: 852,
        },
        link: { label: "Conoce más", href: "/catalogo" },
      },
      {
        eyebrow: "Experiencia",
        title: "Recorre el salón",
        description:
          "Capilla, salón, jardines, cocina y mucho más. Todo conectado en un entorno privado, elegante y armonioso.",
        image: {
          src: "/assets/chenacolo-st-5.jpg",
          alt: "Capilla de Villa Chenacolo rodeada por jardines y palmeras",
          width: 1920,
          height: 1280,
        },
        link: { label: "Conoce más", href: "/experiencia" },
      },
    ] satisfies readonly OfferCardCopy[],
  },
  catalog: {
    eyebrow: "Catálogo",
    titleBefore: "Cada detalle, ",
    titleEmphasis: "pensado",
    titleAfter: " para tu evento.",
    lede:
      "Tenemos un gran repertorio de mesas, copas y otros utensilios para brindarte la experiencia perfecta.",
    pendingTitle: "Catálogo pendiente de integración",
    pendingDescription:
      "La selección de productos reales se mostrará aquí cuando se conecte el catálogo en la siguiente etapa.",
    cta: { label: "Ver catálogo", href: "/catalogo" },
    background: {
      src: "/assets/chenacolo_24.jpeg",
      alt: "",
      width: 852,
      height: 1280,
    },
  },
  gallery: {
    eyebrow: "Galería",
    title: "Queremos presumirte.",
    lede:
      "Queremos presumirte el estilo y la elegancia de nuestro salón. Entra a nuestra galería y llévate una sorpresa.",
    cta: { label: "Ver la galería completa", href: "/experiencia" },
    images: [
      {
        src: "/assets/chenacolo_25.jpeg",
        alt: "Detalle del crucifijo en la capilla de Villa Chenacolo",
        width: 852,
        height: 1280,
        layout: "tall",
      },
      {
        src: "/assets/chenacolo_30.jpeg",
        alt: "Fachada lateral de la capilla y fuente del jardín",
        width: 1024,
        height: 1280,
        layout: "square",
      },
      {
        src: "/assets/chenacolo_31.jpeg",
        alt: "Campanario de la capilla visto desde el puente del jardín",
        width: 1024,
        height: 1280,
        layout: "square",
      },
      {
        src: "/assets/chenacolo_23.jpeg",
        alt: "Medallones decorativos en el interior de la capilla",
        width: 1280,
        height: 852,
        layout: "square",
      },
      {
        src: "/assets/chenacolo_18.jpeg",
        alt: "Detalle de piezas ceremoniales doradas",
        width: 1280,
        height: 852,
        layout: "square",
      },
      {
        src: "/assets/chenacolo_11.jpeg",
        alt: "Nave de la capilla iluminada por una lámpara de cristal",
        width: 1280,
        height: 852,
        layout: "wide",
      },
    ] satisfies readonly GalleryImageCopy[],
  },
  about: {
    eyebrow: "Quiénes somos",
    paragraphs: [
      "Ofrecemos un espacio de alto nivel para celebrar momentos únicos con libertad, privacidad y estilo.",
      "Cada detalle arquitectónico y cada servicio están pensados para que tu evento fluya a la perfección.",
    ],
    pattern: {
      src: "/assets/bg_pattern.png",
      alt: "",
      width: 1668,
      height: 724,
    },
  },
  contact: {
    eyebrow: "Contacto",
    title: "Hablemos de tu evento",
    lede:
      "Si deseas conocer disponibilidad, precios o agendar una visita, déjanos tus datos. Te contactaremos lo antes posible.",
    background: {
      src: "/assets/chenacolo_4.jpeg",
      alt: "",
      width: 1280,
      height: 852,
    },
    form: {
      title: "Queremos ayudarte a crear algo inolvidable",
      description: "Uno de nuestros asesores te dará seguimiento directo.",
      fields: {
        name: {
          label: "Nombre completo",
          helper: "Escribe tu nombre y apellidos.",
        },
        phone: {
          label: "Teléfono",
          helper: "Incluye lada para que podamos identificar el número.",
        },
      },
      button: {
        initial: "Solicitar contacto",
        submitting: "Enviando...",
      },
      successMessage:
        "Recibimos tu solicitud. Uno de nuestros asesores se pondrá en contacto contigo.",
      unavailableMessage:
        "No pudimos enviar tu solicitud en este momento. Inténtalo nuevamente más tarde.",
      validation: {
        nameRequired: "El nombre es obligatorio.",
        nameTooShort: "El nombre debe tener al menos 2 caracteres.",
        nameTooLong: "El nombre no puede exceder 120 caracteres.",
        phoneRequired: "El teléfono es obligatorio.",
        phoneInvalid:
          "Ingresa un número de teléfono válido de entre 10 y 15 dígitos.",
      },
      noScriptMessage:
        "El envío de este formulario requiere JavaScript. Puedes comunicarte con nosotros por teléfono o correo electrónico.",
    },
    info: {
      title: "Encuéntranos",
      description: "Estos son todos los medios por los que puedes contactarnos.",
      labels: {
        address: "Dirección",
        email: "Email",
        facebook: "Facebook",
        phone: "Teléfono",
      },
      facebook: "Villa Chenacolo",
      quoteLead: "O también puedes solicitar una cotización.",
      quoteCta: {
        label: "Solicitar cotización",
        href: "/solicitar-cotizacion",
      },
    },
  },
  closing: {
    eyebrow: "Estás listo",
    title: "Hablemos de tu evento.",
    cta: { label: "Solicitar cotización", href: "/solicitar-cotizacion" },
    whatsappLabel: "O escríbenos por WhatsApp",
  },
} as const;

export const eventKinds: readonly EventSectionCopy[] = [
  {
    ...sharedEventCopy("bodas"),
    id: "seccion-bodas",
    number: "01",
    eyebrow: "Bodas",
    title: "Bodas",
    image: {
      src: "/assets/chenacolo_11.jpeg",
      alt: "Nave central de la capilla preparada para una ceremonia de boda",
      width: 1280,
      height: 852,
    },
    alignment: "image-left",
    variant: "light",
    cta: {
      label: "Me interesa saber más",
      href: eventWhatsappLink("bodas"),
    },
  },
  {
    ...sharedEventCopy("quinceaneras"),
    id: "seccion-quinceaneras",
    number: "02",
    eyebrow: "XV años",
    title: "XV años y celebraciones",
    image: {
      src: "/assets/chenacolo-st-7.jpg",
      alt: "Jardín, fuente y puente frente al salón Villa Chenacolo",
      width: 6000,
      height: 4000,
    },
    alignment: "image-right",
    variant: "dark",
    cta: {
      label: "Me interesa saber más",
      href: eventWhatsappLink("quinceaneras"),
    },
  },
  {
    ...sharedEventCopy("bautizos"),
    id: "seccion-bautizos",
    number: "03",
    eyebrow: "Bautizos",
    title: "Bautizos y comuniones",
    image: {
      src: "/assets/chenacolo-st-9.jpg",
      alt: "Altar y nave interior de la capilla de Villa Chenacolo",
      width: 5889,
      height: 3926,
    },
    alignment: "image-left",
    variant: "light",
    cta: {
      label: "Me interesa saber más",
      href: eventWhatsappLink("bautizos"),
    },
  },
  {
    ...sharedEventCopy("graduaciones"),
    id: "seccion-graduaciones",
    number: "04",
    eyebrow: "Graduaciones",
    title: "Graduaciones",
    image: {
      src: "/assets/chenacolo-st-5.jpg",
      alt: "Capilla y jardines de Villa Chenacolo vistos desde el exterior",
      width: 1920,
      height: 1280,
    },
    alignment: "image-right",
    variant: "dark",
    cta: {
      label: "Me interesa saber más",
      href: eventWhatsappLink("graduaciones"),
    },
  },
  {
    ...sharedEventCopy("privadas"),
    id: "seccion-privadas",
    number: "05",
    eyebrow: "Cumpleaños",
    title: "Cumpleaños y fiestas privadas",
    image: {
      src: "/assets/chenacolo-st-3.jpg",
      alt: "Salón, lago y jardines de Villa Chenacolo al atardecer",
      width: 1920,
      height: 1280,
    },
    alignment: "image-left",
    variant: "light",
    cta: {
      label: "Me interesa saber más",
      href: eventWhatsappLink("privadas"),
    },
  },
  {
    ...sharedEventCopy("corporativos"),
    id: "seccion-corporativos",
    number: "06",
    eyebrow: "Eventos empresariales",
    title: "Eventos empresariales",
    image: {
      src: "/assets/chenacolo-st-4.png",
      alt: "Fachada de la capilla y acceso por el puente de Villa Chenacolo",
      width: 1920,
      height: 1280,
    },
    alignment: "image-right",
    variant: "dark",
    cta: {
      label: "Me interesa saber más",
      href: eventWhatsappLink("corporativos"),
    },
  },
];
