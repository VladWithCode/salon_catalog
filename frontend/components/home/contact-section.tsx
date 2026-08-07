import { ArrowRight, Mail, MapPin, Phone } from "lucide-react";
import Image from "next/image";
import Link from "next/link";

import { ContactForm } from "@/components/home/contact-form";
import { SectionHeading } from "@/components/shared/section-heading";
import { contact } from "@/lib/copy/contact";
import { homeCopy } from "@/lib/copy/home";

type ContactSectionProps = Readonly<{
  /**
   * Only ever set by the no-JavaScript PRG redirect
   * (app/api/contact-requests/route.ts). With JavaScript the form component
   * owns its own success/error state and this is undefined, so the two
   * paths never render two competing confirmations.
   */
  status?: string;
}>;

export function ContactSection({ status }: ContactSectionProps = {}) {
  const copy = homeCopy.contact;
  const sent = status === "enviado";
  const failed = status === "error";

  return (
    <section
      id="contacto"
      aria-labelledby="contact-title"
      className="relative overflow-hidden py-section scroll-mt-20 text-primary-foreground"
    >
      <Image
        src={copy.background.src}
        alt={copy.background.alt}
        fill
        sizes="100vw"
        className="object-cover"
      />
      <div className="absolute inset-0 bg-primary/80" />

      <div className="container-main relative z-10 max-w-7xl space-block">
        <SectionHeading
          eyebrow={copy.eyebrow}
          title={<span id="contact-title">{copy.title}</span>}
          lede={copy.lede}
          align="center"
          inverted
        />

        {sent ? (
          <p
            role="status"
            className="rounded-xl border border-primary-foreground/25 bg-primary/90 px-5 py-4 text-center type-body"
          >
            {copy.form.successMessage}
          </p>
        ) : null}
        {failed ? (
          <p
            role="alert"
            className="rounded-xl border border-destructive/50 bg-destructive/20 px-5 py-4 text-center type-body"
          >
            {copy.form.unavailableMessage}
          </p>
        ) : null}

        <div className="grid gap-6 lg:grid-cols-2 lg:items-stretch">
          <ContactForm />

          <aside className="rounded-xl border border-primary-foreground/20 bg-primary/90 p-6 shadow-elevated backdrop-blur-sm sm:p-8">
            <h3 className="type-h3 font-medium">{copy.info.title}</h3>
            <p className="type-body mt-3 text-primary-foreground/75">
              {copy.info.description}
            </p>

            {/* A <dl> may only contain <dt>/<dd>, or <div> wrappers whose
                direct children are <dt>/<dd>. The previous markup nested a
                second <div> around each pair and put the icon beside it,
                which Lighthouse flagged (definition-list / dlitem) and which
                cost the page its ≥95 accessibility target. The icon now
                lives inside its own <dt>, so the structure is valid while
                the two-column look is unchanged. */}
            <dl className="mt-8 space-y-6">
              <div className="grid grid-cols-[2.75rem_1fr] gap-x-3">
                <dt className="col-span-2 grid grid-cols-[2.75rem_1fr] gap-x-3 type-small font-semibold">
                  <MapPin aria-hidden="true" className="mt-1 size-5 justify-self-center text-accent" />
                  <span>{copy.info.labels.address}</span>
                </dt>
                <dd className="type-body col-start-2 mt-1 text-primary-foreground/75">
                  {contact.address}
                </dd>
              </div>
              <div className="grid grid-cols-[2.75rem_1fr] gap-x-3">
                <dt className="col-span-2 grid grid-cols-[2.75rem_1fr] gap-x-3 type-small font-semibold">
                  <Mail aria-hidden="true" className="mt-1 size-5 justify-self-center text-accent" />
                  <span>{copy.info.labels.email}</span>
                </dt>
                <dd className="type-body col-start-2 mt-1 break-all text-primary-foreground/75">
                  <a href={`mailto:${contact.email}`} className="hover:text-accent-on-dark">
                    {contact.email}
                  </a>
                </dd>
              </div>
              <div className="grid grid-cols-[2.75rem_1fr] gap-x-3">
                <dt className="col-span-2 grid grid-cols-[2.75rem_1fr] gap-x-3 type-small font-semibold">
                  <span
                    aria-hidden="true"
                    className="mt-1 justify-self-center font-display text-xl text-accent-on-dark"
                  >
                    f
                  </span>
                  <span>{copy.info.labels.facebook}</span>
                </dt>
                <dd className="type-body col-start-2 mt-1 text-primary-foreground/75">
                  {copy.info.facebook}
                </dd>
              </div>
              <div className="grid grid-cols-[2.75rem_1fr] gap-x-3">
                <dt className="col-span-2 grid grid-cols-[2.75rem_1fr] gap-x-3 type-small font-semibold">
                  <Phone aria-hidden="true" className="mt-1 size-5 justify-self-center text-accent" />
                  <span>{copy.info.labels.phone}</span>
                </dt>
                <dd className="type-body col-start-2 mt-1 text-primary-foreground/75">
                  <a href={contact.phoneHref} className="hover:text-accent-on-dark">
                    {contact.phone}
                  </a>
                </dd>
              </div>
            </dl>

            <div className="mt-9 border-t border-primary-foreground/20 pt-7">
              <p className="type-small text-primary-foreground/70">
                {copy.info.quoteLead}
              </p>
              <Link
                href={copy.info.quoteCta.href}
                prefetch={false}
                className="type-button mt-3 inline-flex min-h-12 items-center gap-2 rounded-md bg-accent-strong px-6 font-semibold uppercase text-primary-foreground hover:bg-secondary"
              >
                {copy.info.quoteCta.label}
                <ArrowRight aria-hidden="true" className="size-4" />
              </Link>
            </div>
          </aside>
        </div>
      </div>
    </section>
  );
}
