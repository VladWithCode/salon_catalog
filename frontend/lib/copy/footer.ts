export type FooterLink = Readonly<{
  label: string;
  href: string;
}>;

export const footerServices = [
  { label: "Bodas", href: "/#seccion-bodas" },
  { label: "Quinceañeras", href: "/#seccion-quinceaneras" },
  { label: "Bautizos", href: "/#seccion-bautizos" },
  { label: "Eventos corporativos", href: "/#seccion-corporativos" },
  { label: "Graduaciones", href: "/#seccion-graduaciones" },
  { label: "Fiestas privadas", href: "/#seccion-privadas" },
] as const satisfies readonly FooterLink[];

export const footerExplore = [
  { label: "Inicio", href: "/" },
  { label: "Catálogo", href: "/catalogo" },
  { label: "Galería", href: "/experiencia" },
  { label: "Contacto", href: "/#contacto" },
] as const satisfies readonly FooterLink[];

export const footerLegal = [
  { label: "Política de Privacidad", href: "/politica-privacidad" },
  { label: "Términos de Servicio", href: "/terminos-servicio" },
  { label: "Política de Cookies", href: "/politica-cookies" },
] as const satisfies readonly FooterLink[];

export const footerHeadings = {
  services: "Servicios",
  explore: "Explorar",
  contact: "Contacto",
  hours: "Horarios",
} as const;
