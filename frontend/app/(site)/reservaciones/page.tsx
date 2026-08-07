import type { Metadata } from "next";

import { contact } from "@/lib/copy/contact";

// internal/templates/pages/reservations.templ posts to hx-post="/api/reservations",
// a route that is not registered anywhere in internal/routes (grep confirms
// no "/api/reservations" handler exists) — the Go form is already
// non-functional today. Rather than reproduce a broken submission in Next,
// this page keeps the real hero copy and routes visitors to the contact
// channels already approved and used sitewide (WhatsApp, phone), matching
// section 9's instruction: "Si no existe un flujo real: No inventarlo.
// Mantener únicamente copy y CTA respaldados."
export const metadata: Metadata = {
  title: "Reservaciones",
  description: "Consulta fechas disponibles para tu evento en Villa Chenacolo.",
};

export default function ReservationsPage() {
  return (
    <article className="section-light py-section">
      <div className="container-main max-w-2xl space-y-8 text-center">
        <div className="space-y-4">
          <h1 className="type-h1 font-medium">Consulta fechas disponibles</h1>
          <p className="type-lead text-muted-foreground">
            Escríbenos con la fecha de tu evento y te responderemos a la
            brevedad para confirmar disponibilidad.
          </p>
        </div>

        <div className="flex flex-wrap items-center justify-center gap-4">
          <a
            href={contact.whatsapp}
            target="_blank"
            rel="noopener noreferrer"
            className="type-button inline-flex min-h-11 items-center rounded-lg bg-accent-strong px-5 text-primary-foreground hover:bg-accent-strong/90 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent"
          >
            Escribir por WhatsApp
          </a>
          <a
            href={contact.phoneHref}
            className="type-button inline-flex min-h-11 items-center rounded-lg border border-border px-5 text-foreground hover:border-accent focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent"
          >
            Llamar {contact.phone}
          </a>
        </div>
      </div>
    </article>
  );
}
