import Link from "next/link";
import { Button } from "@/components/ui/button";
import { RevealOnView } from "@/lib/motion/reveal-on-view";
import { closingCta } from "@/lib/copy/home";
import { contact } from "@/lib/copy/contact";

/**
 * Closing CTA — one last nudge for the bottom-of-page reader.
 */
export function ClosingCta() {
  return (
    <section className="py-section">
      <div className="container-page">
        <RevealOnView>
          <div className="mx-auto max-w-2xl rounded-2xl border border-border bg-card px-6 py-14 text-center shadow-card md:px-12">
            <p className="text-eyebrow uppercase font-medium tracking-[0.18em] text-accent">
              {closingCta.eyebrow}
            </p>
            <h2 className="mt-3 font-display text-display-md">
              {closingCta.title}
            </h2>
            <div className="mt-8 flex flex-col items-center gap-4">
              <Button
                asChild
                size="lg"
                className="bg-accent text-accent-foreground hover:bg-accent/90"
              >
                <Link href={closingCta.cta.href}>{closingCta.cta.label}</Link>
              </Button>
              <a
                href={contact.whatsapp.href}
                target="_blank"
                rel="noopener noreferrer"
                className="text-body-sm font-medium text-accent transition-colors hover:text-accent/80"
              >
                {closingCta.secondary.label}
              </a>
            </div>
          </div>
        </RevealOnView>
      </div>
    </section>
  );
}
