export type NavLink = Readonly<{
  label: string;
  href: string;
  emphasis?: boolean;
}>;

export const navLinks: readonly NavLink[] = [
  { label: "Inicio", href: "/" },
  { label: "Servicios", href: "/servicios" },
  { label: "Catálogo", href: "/catalogo" },
  { label: "Experiencia", href: "/experiencia" },
  { label: "Contacto", href: "/#contacto" },
  {
    label: "Cotizar",
    href: "/solicitar-cotizacion",
    emphasis: true,
  },
];

export const primaryNavLinks = navLinks.filter((link) => !link.emphasis);
export const quoteNavLink = navLinks.find((link) => link.emphasis)!;
