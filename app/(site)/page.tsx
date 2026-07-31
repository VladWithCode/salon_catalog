import type { Metadata } from "next";
import { HomeHero } from "@/components/home/home-hero";
import { OfferIntro } from "@/components/home/offer-intro";
import { EventSection } from "@/components/home/event-section";
import { CatalogPreview } from "@/components/home/catalog-preview";
import { GalleryPreview } from "@/components/home/gallery-preview";
import { AboutStrip } from "@/components/home/about-strip";
import { ContactSection } from "@/components/home/contact-section";
import { ClosingCta } from "@/components/home/closing-cta";
import { eventKinds } from "@/lib/copy/home";
import { getCatalogListings } from "@/lib/api/catalog";
import type { CatalogListings } from "@/lib/types";

export const metadata: Metadata = {
  title: "Villa Chenacolo · Salón de eventos de alto nivel en Durango",
  description:
    "Bodas, quinceañeras, bautizos, eventos corporativos, graduaciones y fiestas privadas. Capilla, salón climatizado, jardines y catálogo de mobiliario.",
};

async function loadListings(): Promise<CatalogListings> {
  try {
    return await getCatalogListings();
  } catch (err) {
    if (process.env.NODE_ENV !== "production") {
      console.warn("[home] catalog listings unavailable:", err);
    }
    return {};
  }
}

export default async function HomePage() {
  const listings = await loadListings();

  return (
    <article>
      <HomeHero />
      <OfferIntro />
      {eventKinds.map((kind) => (
        <EventSection key={kind.id} kind={kind} />
      ))}
      <CatalogPreview listings={listings} />
      <GalleryPreview />
      <AboutStrip />
      <ContactSection />
      <ClosingCta />
    </article>
  );
}
