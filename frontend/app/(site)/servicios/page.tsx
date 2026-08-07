import type { Metadata } from "next";

import { EventDetailSection } from "@/components/services/event-detail-section";
import { ServicesCta } from "@/components/services/services-cta";
import { ServicesHero } from "@/components/services/services-hero";
import { serviceSections, servicesCopy } from "@/lib/copy/services";

export const metadata: Metadata = {
  title: servicesCopy.metadata.title,
  description: servicesCopy.metadata.description,
};

export default function ServicesPage() {
  return (
    <article className="overflow-x-clip">
      <ServicesHero />
      {serviceSections.map((section) => (
        <EventDetailSection
          key={section.id}
          id={section.id}
          eventKey={section.eventKey}
          eyebrow={section.eyebrow}
          title={section.title}
          description={section.description}
          highlights={section.highlights}
          footnote={section.footnote}
          image={section.image.src}
          imageWidth={section.image.width}
          imageHeight={section.image.height}
          alt={section.image.alt}
          alignment={section.alignment}
          variant={section.variant}
          ctaLabel={section.cta.label}
          href={section.cta.href}
        />
      ))}
      <ServicesCta />
    </article>
  );
}
