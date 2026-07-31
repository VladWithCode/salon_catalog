import Image from "next/image";
import Link from "next/link";
import { Mail, MapPin, Phone } from "lucide-react";
import { RevealOnView } from "@/lib/motion/reveal-on-view";
import { FacebookIcon } from "@/components/site/social-icons";
import { ContactForm } from "./contact-form";
import { contactSection } from "@/lib/copy/home";
import { contact } from "@/lib/copy/contact";

/**
 * Contact section — photo background with a warm overlay, the form on the
 * left, and an "Encuéntranos" info card on the right.
 */
export function ContactSection() {
  return (
    <section id="contacto" className="relative overflow-hidden">
      {/* Background */}
      <div className="absolute inset-0 -z-10">
        <Image
          src="/assets/chenacolo_4.jpeg"
          alt=""
          fill
          sizes="100vw"
          className="object-cover"
        />
        <div className="absolute inset-0 bg-primary/80" />
      </div>

      <div className="container-page py-section text-primary-foreground">
        <RevealOnView>
          <div className="mx-auto max-w-2xl text-center">
            <p className="text-eyebrow uppercase font-medium tracking-[0.18em] text-accent">
              {contactSection.eyebrow}
            </p>
            <h2 className="mt-3 font-display text-display-lg">
              {contactSection.title}
            </h2>
            <p className="mt-4 text-body-lg text-primary-foreground/70">
              {contactSection.lede}
            </p>
          </div>
        </RevealOnView>

        <div className="mt-12 grid items-start gap-8 lg:grid-cols-2">
          <RevealOnView delay={0.1}>
            <ContactForm />
          </RevealOnView>

          <RevealOnView delay={0.2}>
            <div className="rounded-lg bg-primary/60 p-6 backdrop-blur-sm md:p-8">
              <h3 className="font-display text-display-sm">
                {contactSection.info.title}
              </h3>
              <p className="mt-2 text-body-sm text-primary-foreground/70">
                {contactSection.info.subtitle}
              </p>

              <dl className="mt-6 space-y-5">
                <div className="flex items-start gap-4">
                  <MapPin className="mt-0.5 h-5 w-5 shrink-0 text-accent" aria-hidden />
                  <div>
                    <dt className="text-eyebrow uppercase font-medium text-primary-foreground/60">
                      Dirección
                    </dt>
                    <dd className="mt-1 text-body">{contact.address}</dd>
                  </div>
                </div>
                <div className="flex items-start gap-4">
                  <Mail className="mt-0.5 h-5 w-5 shrink-0 text-accent" aria-hidden />
                  <div>
                    <dt className="text-eyebrow uppercase font-medium text-primary-foreground/60">
                      Email
                    </dt>
                    <dd className="mt-1 text-body">
                      <a href={`mailto:${contact.email}`} className="transition-colors hover:text-accent">
                        {contact.email}
                      </a>
                    </dd>
                  </div>
                </div>
                <div className="flex items-start gap-4">
                  <FacebookIcon className="mt-0.5 h-5 w-5 shrink-0 text-accent" aria-hidden />
                  <div>
                    <dt className="text-eyebrow uppercase font-medium text-primary-foreground/60">
                      Facebook
                    </dt>
                    <dd className="mt-1 text-body">Villa Chenacolo</dd>
                  </div>
                </div>
                <div className="flex items-start gap-4">
                  <Phone className="mt-0.5 h-5 w-5 shrink-0 text-accent" aria-hidden />
                  <div>
                    <dt className="text-eyebrow uppercase font-medium text-primary-foreground/60">
                      Teléfono
                    </dt>
                    <dd className="mt-1 text-body">
                      <a href={contact.phoneHref} className="transition-colors hover:text-accent">
                        {contact.phoneDisplay}
                      </a>
                    </dd>
                  </div>
                </div>
              </dl>

              <div className="mt-8 border-t border-primary-foreground/15 pt-6 text-center">
                <p className="text-body-sm text-primary-foreground/60">
                  O también puedes
                </p>
                <Link
                  href={contactSection.info.quoteCta.href}
                  className="mt-2 inline-block font-semibold text-accent transition-colors hover:text-accent/80"
                >
                  {contactSection.info.quoteCta.label} →
                </Link>
              </div>
            </div>
          </RevealOnView>
        </div>
      </div>
    </section>
  );
}
