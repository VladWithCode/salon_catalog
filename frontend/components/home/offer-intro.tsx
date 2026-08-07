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
              {/* The bottom stop carries the card's text, so it is taken to
                  near-opaque cocoa: at /85 the effective background still
                  depended on how light the photo underneath was, which put
                  the eyebrow's contrast below 4.5:1 on the brightest images.
                  At /95 the copy sits on a deterministic surface. */}
              <div className="absolute inset-0 bg-gradient-to-b from-primary/35 via-primary/55 to-primary/95 transition-colors group-hover:from-primary/45 group-hover:to-primary/95" />
              {/* on-dark: this copy sits on the photo's cocoa overlay, so the
                  eyebrow needs the lifted gold, not the light-surface one. */}
              <div className="on-dark absolute inset-x-0 bottom-0 z-10 p-4 sm:p-5 lg:p-6">
                <p className="type-eyebrow">{card.eyebrow}</p>
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
