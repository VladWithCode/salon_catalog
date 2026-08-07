import Link from "next/link";

import type { FooterLink } from "@/lib/copy/footer";

type FooterServicesProps = Readonly<{
  title: string;
  ariaLabel: string;
  links: readonly FooterLink[];
}>;

export function FooterServices({
  title,
  ariaLabel,
  links,
}: FooterServicesProps) {
  return (
    <nav aria-label={ariaLabel}>
      <h2 className="type-eyebrow text-primary-foreground">{title}</h2>
      <ul className="mt-5 space-y-2">
        {links.map((link) => (
          <li key={link.href}>
            <Link
              href={link.href}
              prefetch={false}
              className="inline-flex min-h-11 items-center text-sm text-primary-foreground/70 hover:text-accent-on-dark"
            >
              {link.label}
            </Link>
          </li>
        ))}
      </ul>
    </nav>
  );
}
