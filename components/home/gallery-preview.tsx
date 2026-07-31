import Image from "next/image";
import Link from "next/link";
import { ArrowRight, Plus } from "lucide-react";
import { SectionHeading } from "@/components/shared/section-heading";
import { RevealOnView } from "@/lib/motion/reveal-on-view";
import { StaggerGroup, StaggerItem } from "@/lib/motion/stagger";
import { galleryPreview } from "@/lib/copy/home";
import { cn } from "@/lib/utils";

/**
 * Gallery preview — a 6-image mosaic with a "view full gallery" link.
 * The layout is deliberately asymmetric: one tall tile on the left, one
 * wide tile on the right, four squares in between.
 */
export function GalleryPreview() {
  const imgs = galleryPreview.images;

  return (
    <section id="seccion-galeria" className="py-section">
      <div className="container-page">
        <RevealOnView>
          <SectionHeading
            eyebrow={galleryPreview.eyebrow}
            title={galleryPreview.title}
            lede={galleryPreview.lede}
            rule
          />
        </RevealOnView>

        <StaggerGroup
          className="mt-14 grid grid-cols-2 gap-3 md:grid-cols-4 md:grid-rows-2"
          staggerStep={0.05}
          y={14}
        >
          {imgs.map((img, i) => (
            <StaggerItem
              key={img.src}
              className={cn(
                i === 0 && "md:row-span-2",
                i === 5 && "md:col-span-2",
              )}
            >
              <div className="group relative h-full min-h-40 overflow-hidden rounded-lg">
                <Image
                  src={img.src}
                  alt={img.alt}
                  fill
                  sizes="(min-width: 768px) 25vw, 50vw"
                  className="object-cover transition-transform duration-500 ease-[cubic-bezier(0.22,1,0.36,1)] group-hover:scale-105"
                />
                <div className="absolute inset-0 flex items-center justify-center bg-primary/0 transition-colors duration-300 group-hover:bg-primary/30">
                  <Plus
                    className="h-8 w-8 text-primary-foreground opacity-0 transition-opacity duration-300 group-hover:opacity-100"
                    aria-hidden
                  />
                </div>
              </div>
            </StaggerItem>
          ))}
        </StaggerGroup>

        <RevealOnView className="mt-8 flex justify-end" y={10}>
          <Link
            href={galleryPreview.cta.href}
            className="inline-flex items-center gap-2 text-body font-semibold text-accent transition-colors hover:text-accent/80"
          >
            {galleryPreview.cta.label}
            <ArrowRight className="h-4 w-4" aria-hidden />
          </Link>
        </RevealOnView>
      </div>
    </section>
  );
}
