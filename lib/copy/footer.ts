/**
 * Footer column copy. Keep link labels short (under 20 chars) so they wrap
 * gracefully on mobile.
 */

export const footerColumns = {
  services: {
    title: "Servicios",
    links: [
      { label: "Bodas",                  href: "/#seccion-bodas" },
      { label: "Quinceañeras y Celebraciones", href: "/#seccion-quinceaneras" },
      { label: "Bautizos y Comuniones",  href: "/#seccion-bautizos" },
      { label: "Eventos corporativos",   href: "/#seccion-corporativos" },
      { label: "Graduaciones",           href: "/#seccion-graduaciones" },
      { label: "Fiestas privadas",       href: "/#seccion-privadas" },
    ],
  },
  explore: {
    title: "Explora el sitio",
    links: [
      { label: "Inicio",         href: "/" },
      { label: "Catálogo",       href: "/catalogo" },
      { label: "Acerca de",      href: "/#seccion-oferta" },
      { label: "Galería",        href: "/#seccion-galeria" },
      { label: "Contacto",       href: "/#contacto" },
    ],
  },
  legal: {
    copyright: (year: number) =>
      `© ${year} Villa Chenacolo. Todos los derechos reservados.`,
    links: [
      { label: "Política de Privacidad", href: "/politica-privacidad" },
      { label: "Términos de Servicio",   href: "/terminos-servicio" },
      { label: "Política de Cookies",    href: "/politica-cookies" },
    ],
  },
} as const;
