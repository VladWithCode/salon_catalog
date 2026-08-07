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

type HomePageProps = Readonly<{
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}>;

export default async function HomePage({ searchParams }: HomePageProps) {
  const params = await searchParams;
  // Set by the no-JavaScript PRG redirect in
  // app/api/contact-requests/route.ts — with JS the client component shows
  // its own inline state and this stays undefined.
  const rawContactStatus = params.contacto;
  const contactStatus = Array.isArray(rawContactStatus)
    ? rawContactStatus[0]
    : rawContactStatus;
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
      <ContactSection status={contactStatus} />
      <ClosingCta />
    </article>
  );
}
