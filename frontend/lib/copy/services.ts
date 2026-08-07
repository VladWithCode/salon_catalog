import { contact } from "@/lib/copy/contact";
import { eventsByKey, type EventKey } from "@/lib/copy/events";
import type { ImageAsset } from "@/lib/types";

export type ServiceLink = Readonly<{
  label: string;
  href: string;
}>;

export type ServiceSectionCopy = Readonly<{
  id: string;
  eventKey: EventKey;
  eyebrow: string;
  title: string;
  description: string;
  highlights: readonly string[];
  footnote?: string;
  image: ImageAsset;
  alignment: "image-left" | "image-right";
  variant: "light" | "dark";
  cta: ServiceLink;
}>;

function whatsappLink(message: string): string {
  const url = new URL(contact.whatsapp);
  url.searchParams.set("text", message);
  return url.toString();
}

function sharedEventCopy(
  key: EventKey,
): Pick<ServiceSectionCopy, "description" | "highlights" | "footnote"> {
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

const ctaLabel = "Me interesa saber más";

export const servicesCopy = {
  metadata: {
    title: "Servicios",
    description:
      "Conoce las opciones de Villa Chenacolo para bodas, quinceañeras, bautizos, eventos corporativos, graduaciones y fiestas privadas.",
  },
  hero: {
    eyebrow: "Servicios",
    title: "Un lugar que eleva cualquier celebración.",
    intro:
      "En Villa Chenacolo no organizamos tu evento: te entregamos el escenario perfecto para que lo vivas como lo imaginaste, sin complicaciones. Nuestra renta de salón incluye un espacio de alto nivel, con cada detalle pensado para brindar comodidad, elegancia y funcionalidad. Tú eliges a tus proveedores, nosotros ponemos lo mejor para recibirlos.",
    primaryCta: {
      label: "Solicitar cotización",
      href: "/solicitar-cotizacion",
    },
    background: {
      src: "/assets/chenacolo_24.jpeg",
      alt: "",
      width: 852,
      height: 1280,
    },
  },
  closing: {
    id: "seccion-cta",
    eyebrow: "Tu evento",
    title: "Hablemos de tu evento.",
    cta: {
      label: "Solicitar cotización",
      href: "/solicitar-cotizacion",
    },
    whatsappLabel: "O escríbenos por WhatsApp",
    whatsappHref: contact.whatsapp,
  },
} as const;

export const serviceSections: readonly ServiceSectionCopy[] = [
  {
    ...sharedEventCopy("bodas"),
    id: "seccion-bodas",
    eventKey: "bodas",
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
    cta: { label: ctaLabel, href: eventWhatsappLink("bodas") },
  },
  {
    ...sharedEventCopy("quinceaneras"),
    id: "seccion-quinceaneras",
    eventKey: "quinceaneras",
    eyebrow: "XV años",
    title: "Quinceañeras y celebraciones",
    image: {
      src: "/assets/chenacolo_15.jpeg",
      alt: "Lámparas de cristal iluminadas en el salón Villa Chenacolo",
      width: 852,
      height: 1280,
    },
    alignment: "image-right",
    variant: "dark",
    cta: { label: ctaLabel, href: eventWhatsappLink("quinceaneras") },
  },
  {
    ...sharedEventCopy("bautizos"),
    id: "seccion-bautizos",
    eventKey: "bautizos",
    eyebrow: "Bautizos",
    title: "Bautizos y comuniones",
    image: {
      src: "/assets/chenacolo_10.jpeg",
      alt: "Nave interior de la capilla de Villa Chenacolo",
      width: 852,
      height: 1280,
    },
    alignment: "image-left",
    variant: "light",
    cta: { label: ctaLabel, href: eventWhatsappLink("bautizos") },
  },
  {
    ...sharedEventCopy("corporativos"),
    id: "seccion-corporativos",
    eventKey: "corporativos",
    eyebrow: "Eventos profesionales",
    title: "Eventos corporativos",
    image: {
      src: "/assets/chenacolo_8.jpeg",
      alt: "Acceso principal, jardines y fachada de Villa Chenacolo",
      width: 1280,
      height: 852,
    },
    alignment: "image-right",
    variant: "dark",
    cta: { label: ctaLabel, href: eventWhatsappLink("corporativos") },
  },
  {
    ...sharedEventCopy("graduaciones"),
    id: "seccion-graduaciones",
    eventKey: "graduaciones",
    eyebrow: "Graduaciones",
    title: "Graduaciones",
    image: {
      src: "/assets/chenacolo_13.jpeg",
      alt: "Lámpara de cristal reflejada en los ventanales del salón",
      width: 1280,
      height: 852,
    },
    alignment: "image-left",
    variant: "light",
    cta: { label: ctaLabel, href: eventWhatsappLink("graduaciones") },
  },
  {
    ...sharedEventCopy("privadas"),
    id: "seccion-privadas",
    eventKey: "privadas",
    eyebrow: "Celebraciones privadas",
    title: "Fiestas privadas",
    image: {
      src: "/assets/chenacolo_4.jpeg",
      alt: "Fachada y fuente de Villa Chenacolo",
      width: 1280,
      height: 852,
    },
    alignment: "image-right",
    variant: "dark",
    cta: { label: ctaLabel, href: eventWhatsappLink("privadas") },
  },
];
