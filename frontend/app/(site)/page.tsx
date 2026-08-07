import type { Metadata } from "next";

import { AboutStrip } from "@/components/home/about-strip";
import { CatalogPreview } from "@/components/home/catalog-preview";
import { ClosingCta } from "@/components/home/closing-cta";
import { ContactSection } from "@/components/home/contact-section";
import { EventSection } from "@/components/home/event-section";
import { GalleryPreview } from "@/components/home/gallery-preview";
import { HomeHero } from "@/components/home/home-hero";
import { OfferIntro } from "@/components/home/offer-intro";
import { getCatalogListings } from "@/lib/api/catalog";
import { eventKinds } from "@/lib/copy/home";

export const metadata: Metadata = {
  title: {
    absolute: "Villa Chenacolo · Salón de eventos de alto nivel en Durango",
  },
};

export default async function HomePage() {
  const catalog = await getCatalogListings();

  return (
    <article className="overflow-x-clip">
      <HomeHero />
      <OfferIntro />
      {eventKinds.map((event) => (
        <EventSection key={event.id} event={event} />
      ))}
      <CatalogPreview status={catalog.status} categories={catalog.categories} />
      <GalleryPreview />
      <AboutStrip />
      <ContactSection />
      <ClosingCta />
    </article>
  );
}
