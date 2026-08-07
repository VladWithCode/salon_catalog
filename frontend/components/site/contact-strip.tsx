import { Mail, MapPin, Phone } from "lucide-react";

import { contact } from "@/lib/copy/contact";

export function ContactStrip() {
  return (
    <section
      aria-label="Contacto rápido"
      className="border-y border-border bg-secondary text-secondary-foreground"
    >
      <address className="mx-auto flex max-w-7xl flex-col gap-2 px-6 py-4 text-sm not-italic sm:flex-row sm:flex-wrap sm:items-center sm:justify-between md:px-10">
        <span className="flex min-h-11 items-center gap-2">
          <MapPin aria-hidden="true" className="size-4 shrink-0" />
          {contact.address}
        </span>
        <a
          href={contact.phoneHref}
          className="flex min-h-11 items-center gap-2 hover:text-accent-strong"
        >
          <Phone aria-hidden="true" className="size-4" />
          {contact.phone}
        </a>
        <a
          href={`mailto:${contact.email}`}
          className="flex min-h-11 items-center gap-2 hover:text-accent-strong"
        >
          <Mail aria-hidden="true" className="size-4" />
          {contact.email}
        </a>
      </address>
    </section>
  );
}
