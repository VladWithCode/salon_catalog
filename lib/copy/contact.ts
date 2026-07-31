/**
 * Brand & contact details shared by the header (phone link), footer (contact
 * column), and WhatsApp floating action button.
 */

/** Phone number used for both `tel:` and WhatsApp deep links, digits only. */
export const phoneDigits = "526182593026";

export const brand = {
  name: "Villa Chenacolo",
  tagline: "Sala de Acontecimientos Especiales",
  description:
    "Creamos momentos inolvidables con servicios de planificación de eventos excepcionales y de alto nivel.",
} as const;

/** Build a WhatsApp deep link with a pre-filled message. */
export function whatsappHref(message: string): string {
  return `https://wa.me/${phoneDigits}?text=${encodeURIComponent(message)}`;
}

export const contact = {
  address: "Entronque, 34234 Ignacio López Rayón, Dgo.",
  phoneDisplay: "618 259 3026",
  phoneHref: `tel:+${phoneDigits}`,
  email: "villachenacolo@gmail.com",
  hours: [
    { day: "Lun – Vie", hours: "9:00 – 18:00" },
    { day: "Sáb – Dom", hours: "9:00 – 18:00" },
    { day: "Dom",       hours: "Con cita" },
  ],
  whatsapp: {
    label: "WhatsApp",
    href: whatsappHref("Hola! Me gustaría más información sobre su salón."),
  },
} as const;
