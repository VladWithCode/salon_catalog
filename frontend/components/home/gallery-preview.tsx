import { ArrowRight } from "lucide-react";
import Link from "next/link";

import { ImageTile } from "@/components/shared/image-tile";
import { SectionHeading } from "@/components/shared/section-heading";
import { homeCopy } from "@/lib/copy/home";
import { cn } from "@/lib/utils";

const galleryLayouts = [
  "sm:col-span-2 sm:row-span-2",
  "sm:col-span-2",
  "sm:col-span-2",
  "sm:col-span-1",
  "sm:col-span-1",
  "sm:col-span-2",
] as const;

export function GalleryPreview() {
  const { gallery } = homeCopy;

  return (
    <section id="experiencia" className="section-light py-section scroll-mt-20">
      <div className="container-main max-w-7xl space-block">
        <SectionHeading
          eyebrow={gallery.eyebrow}
          title={gallery.title}
          lede={gallery.lede}
        />

        <div className="grid auto-rows-[12rem] grid-cols-2 gap-3 sm:auto-rows-[13rem] sm:grid-cols-6 md:auto-rows-[15rem]">
          {gallery.images.map((image, index) => (
            <ImageTile
              key={image.src}
              image={image}
              sizes="(min-width: 768px) 34vw, (min-width: 640px) 50vw, 50vw"
              className={cn(
                "min-h-0 rounded-lg shadow-none",
                galleryLayouts[index],
              )}
            >
              <span
                aria-hidden="true"
                className="absolute inset-0 flex items-center justify-center bg-primary/0 text-3xl text-primary-foreground transition-colors group-hover:bg-primary/30"
              >
                <span className="opacity-0 transition-opacity group-hover:opacity-100">+</span>
              </span>
            </ImageTile>
          ))}
        </div>

        <div className="flex justify-end">
          <Link
            href={gallery.cta.href}
            prefetch={false}
            className="type-button inline-flex min-h-12 items-center gap-2 font-semibold uppercase text-primary hover:text-accent-strong"
          >
            {gallery.cta.label}
            <ArrowRight aria-hidden="true" className="size-4" />
          </Link>
        </div>
      </div>
    </section>
  );
}
