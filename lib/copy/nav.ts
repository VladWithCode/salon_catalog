/**
 * Primary site navigation. The header renders these links.
 * The `href` values are real app routes; on the home page, hash anchors scroll
 * to the corresponding sections. The footer also reuses this list.
 */

export type NavLink = {
  label: string;
  href: `/${string}` | `#${string}`;
  description?: string;
  emphasis?: boolean;
};

export const navLinks: readonly NavLink[] = [
  { label: "Inicio",      href: "/" },
  { label: "Servicios",   href: "/servicios" },
  { label: "Catálogo",    href: "/catalogo" },
  { label: "Experiencia", href: "/experiencia" },
  { label: "Contacto",    href: "/#contacto" },
  {
    label: "Cotizar",
    href: "/solicitar-cotizacion",
    emphasis: true,
  },
] as const;
