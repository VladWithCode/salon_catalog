import Link from "next/link";
import { Mail, MapPin, Phone } from "lucide-react";
import { Logo } from "./logo";
import { SocialPlatformIcon } from "./social-icons";
import { brand, contact } from "@/lib/copy/contact";
import { footerColumns } from "@/lib/copy/footer";
import type { SocialLink } from "@/lib/types";

type SiteFooterProps = {
  socials: SocialLink[];
};

/**
 * Site footer: brand column + services links + explore links + contact info,
 * plus a legal strip with copyright and policy links.
 */
export function SiteFooter({ socials }: SiteFooterProps) {
  const year = new Date().getFullYear();

  return (
    <footer className="bg-primary text-primary-foreground">
      <div className="h-px bg-accent/30" aria-hidden />
      <div className="container-page py-16 md:py-20">
        <div className="grid grid-cols-1 gap-12 md:grid-cols-12">
          {/* Brand */}
          <div className="space-y-5 md:col-span-4">
            <Logo variant="dark" />
            <p className="max-w-xs text-body-sm text-primary-foreground/70">
              {brand.description}
            </p>
            {socials.length > 0 && (
              <ul className="flex items-center gap-4" aria-label="Redes sociales">
                {socials.map((s) => (
                  <li key={s.id ?? s.name}>
                    <a
                      href={s.link}
                      target="_blank"
                      rel="noopener noreferrer"
                      aria-label={s.label ?? s.name}
                      className="text-primary-foreground/80 transition-colors hover:text-accent"
                    >
                      <SocialPlatformIcon platform={s.platform} className="h-5 w-5" />
                      <span className="sr-only">{s.label ?? s.name}</span>
                    </a>
                  </li>
                ))}
              </ul>
            )}
          </div>

          {/* Services */}
          <nav aria-label="Servicios" className="md:col-span-2">
            <h3 className="text-eyebrow uppercase font-medium text-primary-foreground/60">
              {footerColumns.services.title}
            </h3>
            <ul className="mt-4 space-y-2.5">
              {footerColumns.services.links.map((l) => (
                <li key={l.href}>
                  <Link
                    href={l.href}
                    className="text-body-sm text-primary-foreground/80 transition-colors hover:text-accent"
                  >
                    {l.label}
                  </Link>
                </li>
              ))}
            </ul>
          </nav>

          {/* Explore */}
          <nav aria-label="Explora el sitio" className="md:col-span-2">
            <h3 className="text-eyebrow uppercase font-medium text-primary-foreground/60">
              {footerColumns.explore.title}
            </h3>
            <ul className="mt-4 space-y-2.5">
              {footerColumns.explore.links.map((l) => (
                <li key={l.href}>
                  <Link
                    href={l.href}
                    className="text-body-sm text-primary-foreground/80 transition-colors hover:text-accent"
                  >
                    {l.label}
                  </Link>
                </li>
              ))}
            </ul>
          </nav>

          {/* Contact */}
          <div className="md:col-span-4">
            <h3 className="text-eyebrow uppercase font-medium text-primary-foreground/60">
              Información de contacto
            </h3>
            <address className="mt-4 space-y-3 text-body-sm not-italic text-primary-foreground/80">
              <p className="flex items-start gap-3">
                <MapPin className="mt-0.5 h-4 w-4 shrink-0 text-accent" aria-hidden />
                <span>{contact.address}</span>
              </p>
              <p className="flex items-center gap-3">
                <Phone className="h-4 w-4 shrink-0 text-accent" aria-hidden />
                <a href={contact.phoneHref} className="transition-colors hover:text-accent">
                  {contact.phoneDisplay}
                </a>
              </p>
              <p className="flex items-center gap-3">
                <Mail className="h-4 w-4 shrink-0 text-accent" aria-hidden />
                <a href={`mailto:${contact.email}`} className="transition-colors hover:text-accent">
                  {contact.email}
                </a>
              </p>
            </address>
            <dl className="mt-4 space-y-1 text-body-sm text-primary-foreground/60">
              {contact.hours.map((h) => (
                <div key={h.day} className="flex justify-between gap-4 max-w-[16rem]">
                  <dt>{h.day}</dt>
                  <dd>{h.hours}</dd>
                </div>
              ))}
            </dl>
          </div>
        </div>

        {/* Legal */}
        <div className="mt-14 flex flex-col items-center justify-between gap-4 border-t border-primary-foreground/15 pt-8 md:flex-row">
          <p className="text-body-sm text-primary-foreground/60">
            {footerColumns.legal.copyright(year)}
          </p>
          <ul className="flex flex-wrap items-center gap-x-6 gap-y-2">
            {footerColumns.legal.links.map((l) => (
              <li key={l.href}>
                <Link
                  href={l.href}
                  className="text-body-sm text-primary-foreground/60 transition-colors hover:text-accent"
                >
                  {l.label}
                </Link>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </footer>
  );
}
