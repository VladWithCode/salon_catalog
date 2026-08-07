import { ArrowUpRight } from "lucide-react";

import { ImageTile } from "@/components/shared/image-tile";
import { SectionHeading } from "@/components/shared/section-heading";
import { homeCopy } from "@/lib/copy/home";

export function OfferIntro() {
  const { offer } = homeCopy;

  return (
    <section id="nuestra-oferta" className="section-light py-section">
      <div className="container-main max-w-7xl space-block">
        <SectionHeading
          eyebrow={offer.eyebrow}
          title={
            <>
              {offer.titleBefore}
              <em className="font-normal">{offer.titleEmphasis}</em>
              {offer.titleAfter}
            </>
          }
          lede={offer.lede}
          align="center"
        />

        <div className="grid gap-6 md:grid-cols-3">
          {offer.cards.map((card) => (
            <ImageTile
              key={card.link.href}
              image={card.image}
              href={card.link.href}
              sizes="(min-width: 768px) 33vw, 100vw"
              className="aspect-[5/4] text-primary-foreground"
            >
              <div className="absolute inset-0 bg-gradient-to-b from-primary/35 via-primary/45 to-primary/85 transition-colors group-hover:from-primary/45 group-hover:to-primary/90" />
              <div className="absolute inset-x-0 bottom-0 z-10 p-4 sm:p-5 lg:p-6">
                <p className="type-eyebrow text-accent">{card.eyebrow}</p>
                <h3 className="type-h3 mt-3 font-medium">{card.title}</h3>
                <span className="type-button mt-5 inline-flex min-h-11 items-center gap-2 font-semibold uppercase">
                  {card.link.label}
                  <ArrowUpRight aria-hidden="true" className="size-4" />
                </span>
              </div>
            </ImageTile>
          ))}
        </div>
      </div>
    </section>
  );
}
