import Link from "next/link";

import { FooterServices } from "@/components/site/footer-services";
import { Logo } from "@/components/site/logo";
import { SocialIcons } from "@/components/site/social-icons";
import { contact } from "@/lib/copy/contact";
import {
  footerExplore,
  footerHeadings,
  footerLegal,
  footerServices,
} from "@/lib/copy/footer";
import type { SocialLink } from "@/lib/types";

type SiteFooterProps = Readonly<{
  socials: readonly SocialLink[];
}>;

export function SiteFooter({ socials }: SiteFooterProps) {
  const year = new Date().getFullYear();

  return (
    <footer className="bg-primary text-primary-foreground">
      <div aria-hidden="true" className="h-px bg-accent/30" />
      <div className="mx-auto max-w-7xl px-6 py-20 pb-28 md:px-10 md:py-24">
        <div className="grid grid-cols-1 gap-12 sm:grid-cols-2 md:grid-cols-12">
          <section className="space-y-6 sm:col-span-2 md:col-span-4">
            <Logo variant="dark" />
            <p className="type-small max-w-sm text-primary-foreground/70">
              {contact.tagline}
            </p>
            <SocialIcons socials={socials} />
          </section>

          <div className="md:col-span-2">
            <FooterServices
              title={footerHeadings.services}
              ariaLabel="Servicios"
              links={footerServices}
            />
          </div>

          <div className="md:col-span-2">
            <FooterServices
              title={footerHeadings.explore}
              ariaLabel="Explorar el sitio"
              links={footerExplore}
            />
          </div>

          <section className="space-y-5 md:col-span-4">
            <h2 className="type-eyebrow text-primary-foreground">
              {footerHeadings.contact}
            </h2>
            <address className="space-y-2 text-sm not-italic text-primary-foreground/70">
              <p>{contact.address}</p>
              <p>
                <a
                  href={contact.phoneHref}
                  className="inline-flex min-h-11 items-center hover:text-accent"
                >
                  {contact.phone}
                </a>
              </p>
              <p>
                <a
                  href={`mailto:${contact.email}`}
                  className="inline-flex min-h-11 items-center break-all hover:text-accent"
                >
                  {contact.email}
                </a>
              </p>
            </address>
            <div>
              <h3 className="text-sm font-medium text-primary-foreground">
                {footerHeadings.hours}
              </h3>
              <dl className="mt-3 grid max-w-sm grid-cols-[auto_1fr] gap-x-5 gap-y-2 text-sm text-primary-foreground/70">
                {contact.hours.map((item) => (
                  <div key={`${item.day}-${item.hours}`} className="contents">
                    <dt>{item.day}</dt>
                    <dd>{item.hours}</dd>
                  </div>
                ))}
              </dl>
            </div>
          </section>
        </div>

        <div className="mt-16 flex flex-col gap-6 border-t border-primary-foreground/15 pt-8 text-sm text-primary-foreground/60 md:flex-row md:items-center md:justify-between">
          <p>
            © {year} {contact.brand}. Todos los derechos reservados.
          </p>
          <nav aria-label="Páginas legales">
            <ul className="flex flex-wrap gap-x-6 gap-y-2">
              {footerLegal.map((link) => (
                <li key={link.href}>
                  <Link
                    href={link.href}
                    prefetch={false}
                    className="inline-flex min-h-11 items-center hover:text-accent"
                  >
                    {link.label}
                  </Link>
                </li>
              ))}
            </ul>
          </nav>
        </div>
      </div>
    </footer>
  );
}
