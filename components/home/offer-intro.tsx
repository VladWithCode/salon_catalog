import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { ImageTile } from "@/components/shared/image-tile";
import {
  EmphasizedTitle,
  SectionHeading,
} from "@/components/shared/section-heading";
import { RevealOnView } from "@/lib/motion/reveal-on-view";
import { StaggerGroup, StaggerItem } from "@/lib/motion/stagger";
import { offerIntro } from "@/lib/copy/home";

/**
 * "Nuestra oferta" — intro copy + three feature cards that link to the
 * services, catalog, and experience pages.
 */
export function OfferIntro() {
  return (
    <section id="seccion-oferta" className="py-section">
      <div className="container-page">
        <RevealOnView>
          <SectionHeading
            eyebrow={offerIntro.eyebrow}
            title={
              <EmphasizedTitle
                title={offerIntro.title}
                word={offerIntro.italicWord}
              />
            }
            lede={offerIntro.lede}
            rule
          />
        </RevealOnView>

        <StaggerGroup
          className="mt-14 grid gap-6 md:grid-cols-3"
          staggerStep={0.1}
        >
          {offerIntro.cards.map((card) => (
            <StaggerItem key={card.title}>
              <Link
                href={card.href}
                className="group relative block overflow-hidden rounded-xl shadow-card transition-shadow duration-300 hover:shadow-elevated"
              >
                <ImageTile
                  src={card.image}
                  alt={card.title}
                  ratio="aspect-[5/4]"
                  sizes="(min-width: 768px) 33vw, 100vw"
                  className="rounded-xl"
                />
                <div className="absolute inset-0 flex flex-col justify-end bg-gradient-to-t from-primary/85 via-primary/35 to-transparent p-6 text-primary-foreground">
                  <h3 className="font-display text-display-sm">{card.title}</h3>
                  <p className="mt-2 max-w-[32ch] text-body-sm text-primary-foreground/80">
                    {card.lede}
                  </p>
                  <span className="mt-4 inline-flex items-center gap-1.5 text-body-sm font-semibold text-accent">
                    {card.cta}
                    <ArrowRight
                      className="h-4 w-4 transition-transform duration-300 group-hover:translate-x-1"
                      aria-hidden
                    />
                  </span>
                </div>
              </Link>
            </StaggerItem>
          ))}
        </StaggerGroup>
      </div>
    </section>
  );
}
