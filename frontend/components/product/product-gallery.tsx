"use client";

import Image from "next/image";
import { AnimatePresence, motion, MotionConfig } from "motion/react";
import { useState } from "react";

const fallbackImageUrl = "/assets/chenacolo_24.jpeg";

/**
 * Same-origin path served by the existing, unmodified proxy
 * (frontend/app/api/catalog/media/[filename]/route.ts). filename already
 * passed Go's isSafeCatalogProductImageFilename check and this fetcher's
 * own runtime validation before it ever reaches this component — no
 * further safety check happens here, only URL construction.
 */
function mediaURL(filename: string): string {
  return `/api/catalog/media/${encodeURIComponent(filename)}`;
}

type ProductGalleryProps = Readonly<{
  imageFilename: string | null;
  images: readonly string[];
  productName: string;
}>;

/**
 * Small Client Component: the only interactive piece of the detail page.
 * Every image — main and gallery — is a real <img> (via next/image)
 * present in the initial HTML regardless of JavaScript; only the "which
 * one is currently large" state requires hydration. Without JavaScript the
 * main image is still visible and every thumbnail is still a visible,
 * loadable image, satisfying the no-JS requirement without a second
 * server-rendered gallery implementation.
 */
export function ProductGallery({
  imageFilename,
  images,
  productName,
}: ProductGalleryProps) {
  const gallery = images.filter((filename) => filename !== imageFilename);
  const allImages = imageFilename !== null ? [imageFilename, ...gallery] : gallery;
  const [activeIndex, setActiveIndex] = useState(0);

  if (allImages.length === 0) {
    return (
      <div className="relative aspect-square w-full overflow-hidden rounded-lg bg-muted">
        <Image
          src={fallbackImageUrl}
          alt={productName}
          fill
          sizes="(min-width: 1024px) 50vw, 100vw"
          className="object-cover"
        />
      </div>
    );
  }

  const boundedIndex = Math.min(activeIndex, allImages.length - 1);
  const activeFilename = allImages[boundedIndex];

  return (
    // reducedMotion="user" makes every animation in this subtree an
    // instant cut when the visitor has prefers-reduced-motion set — the
    // only place Motion is used on the site, kept intentionally small.
    <MotionConfig reducedMotion="user">
      <div className="space-y-4">
        <div className="relative aspect-square w-full overflow-hidden rounded-lg bg-muted">
          <AnimatePresence initial={false} mode="sync">
            <motion.div
              key={activeFilename}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.2, ease: "easeOut" }}
              className="absolute inset-0"
            >
              <Image
                src={mediaURL(activeFilename)}
                alt={productName}
                fill
                sizes="(min-width: 1024px) 50vw, 100vw"
                className="object-cover"
                priority
              />
            </motion.div>
          </AnimatePresence>
        </div>

        {allImages.length > 1 ? (
          <ul
            className="flex gap-3 overflow-x-auto pb-1"
            aria-label={`Galería de imágenes de ${productName}`}
          >
            {allImages.map((filename, index) => (
              <li key={filename} className="shrink-0">
                <button
                  type="button"
                  onClick={() => setActiveIndex(index)}
                  aria-pressed={index === boundedIndex}
                  className={`relative block h-16 w-16 min-h-11 min-w-11 overflow-hidden rounded-md border-2 transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent ${
                    index === boundedIndex
                      ? "border-accent"
                      : "border-border hover:border-accent/50"
                  }`}
                >
                  <Image
                    src={mediaURL(filename)}
                    alt={`Miniatura ${index + 1} de ${productName}`}
                    fill
                    sizes="64px"
                    className="object-cover"
                  />
                </button>
              </li>
            ))}
          </ul>
        ) : null}
      </div>
    </MotionConfig>
  );
}
