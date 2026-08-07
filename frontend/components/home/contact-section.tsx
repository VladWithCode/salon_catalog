import { ArrowRight, Mail, MapPin, Phone } from "lucide-react";
import Image from "next/image";
import Link from "next/link";

import { ContactForm } from "@/components/home/contact-form";
import { SectionHeading } from "@/components/shared/section-heading";
import { contact } from "@/lib/copy/contact";
import { homeCopy } from "@/lib/copy/home";

export function ContactSection() {
  const copy = homeCopy.contact;

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

        <div className="grid gap-6 lg:grid-cols-2 lg:items-stretch">
          <ContactForm />

          <aside className="rounded-xl border border-primary-foreground/20 bg-primary/90 p-6 shadow-elevated backdrop-blur-sm sm:p-8">
            <h3 className="type-h3 font-medium">{copy.info.title}</h3>
            <p className="type-body mt-3 text-primary-foreground/75">
              {copy.info.description}
            </p>

            <dl className="mt-8 space-y-6">
              <div className="grid grid-cols-[2.75rem_1fr] gap-x-3">
                <MapPin aria-hidden="true" className="mt-1 size-5 justify-self-center text-accent" />
                <div>
                  <dt className="type-small font-semibold">{copy.info.labels.address}</dt>
                  <dd className="type-body mt-1 text-primary-foreground/75">
                    {contact.address}
                  </dd>
                </div>
              </div>
              <div className="grid grid-cols-[2.75rem_1fr] gap-x-3">
                <Mail aria-hidden="true" className="mt-1 size-5 justify-self-center text-accent" />
                <div>
                  <dt className="type-small font-semibold">{copy.info.labels.email}</dt>
                  <dd className="type-body mt-1 break-all text-primary-foreground/75">
                    <a href={`mailto:${contact.email}`} className="hover:text-accent">
                      {contact.email}
                    </a>
                  </dd>
                </div>
              </div>
              <div className="grid grid-cols-[2.75rem_1fr] gap-x-3">
                <span
                  aria-hidden="true"
                  className="mt-1 justify-self-center font-display text-xl text-accent"
                >
                  f
                </span>
                <div>
                  <dt className="type-small font-semibold">{copy.info.labels.facebook}</dt>
                  <dd className="type-body mt-1 text-primary-foreground/75">
                    {copy.info.facebook}
                  </dd>
                </div>
              </div>
              <div className="grid grid-cols-[2.75rem_1fr] gap-x-3">
                <Phone aria-hidden="true" className="mt-1 size-5 justify-self-center text-accent" />
                <div>
                  <dt className="type-small font-semibold">{copy.info.labels.phone}</dt>
                  <dd className="type-body mt-1 text-primary-foreground/75">
                    <a href={contact.phoneHref} className="hover:text-accent">
                      {contact.phone}
                    </a>
                  </dd>
                </div>
              </div>
            </dl>

            <div className="mt-9 border-t border-primary-foreground/20 pt-7">
              <p className="type-small text-primary-foreground/70">
                {copy.info.quoteLead}
              </p>
              <Link
                href={copy.info.quoteCta.href}
                prefetch={false}
                className="type-button mt-3 inline-flex min-h-12 items-center gap-2 rounded-md bg-accent px-6 font-semibold uppercase text-accent-foreground hover:bg-secondary"
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
