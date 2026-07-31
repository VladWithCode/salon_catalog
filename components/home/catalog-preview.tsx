import Image from "next/image";
import Link from "next/link";
import { ArrowRight } from "lucide-react";
import {
  EmphasizedTitle,
  SectionHeading,
} from "@/components/shared/section-heading";
import { RevealOnView } from "@/lib/motion/reveal-on-view";
import { StaggerGroup, StaggerItem } from "@/lib/motion/stagger";
import { catalogPreview } from "@/lib/copy/home";
import type { CatalogListings, CatalogProd } from "@/lib/types";

type CatalogPreviewProps = {
  listings: CatalogListings;
};

const FALLBACK_IMG = "/assets/chenacolo_24.jpeg";

function productImage(p: CatalogProd): string {
  if (p.image_url) return `/assets/uploads/${p.image_url}`;
  return FALLBACK_IMG;
}

/**
 * Catalog preview — one horizontal strip per category with up to 4 products
 * and a "Ver todos" card. Falls back to a single CTA card when there are no
 * listings (e.g. API unavailable).
 */
export function CatalogPreview({ listings }: CatalogPreviewProps) {
  const categories = Object.keys(listings).sort((a, b) => a.localeCompare(b, "es"));

  return (
    <section id="seccion-catalogo" className="relative overflow-hidden">
      {/* Tinted photo background */}
      <div className="absolute inset-0 -z-10">
        <Image
          src={FALLBACK_IMG}
          alt=""
          fill
          sizes="100vw"
          className="object-cover"
        />
        <div className="absolute inset-0 bg-gradient-to-tl from-primary/70 via-primary/55 to-accent/35" />
      </div>

      <div className="container-page py-section text-primary-foreground">
        <RevealOnView>
          <SectionHeading
            dark
            eyebrow={catalogPreview.eyebrow}
            title={
              <EmphasizedTitle
                title={catalogPreview.title}
                word={catalogPreview.italicWord}
              />
            }
            lede={catalogPreview.lede}
            rule
          />
        </RevealOnView>

        {categories.length === 0 ? (
          <RevealOnView className="mt-14 flex justify-center">
            <Link
              href="/catalogo"
              className="group relative block aspect-square w-full max-w-sm overflow-hidden rounded-xl shadow-card"
            >
              <Image
                src={FALLBACK_IMG}
                alt="Catálogo de Villa Chenacolo"
                fill
                sizes="(min-width: 640px) 24rem, 100vw"
                className="object-cover transition-transform duration-500 group-hover:scale-105"
              />
              <div className="absolute inset-0 flex items-center justify-center bg-card/90">
                <span className="font-display text-display-sm uppercase text-foreground">
                  Ver catálogo
                </span>
              </div>
            </Link>
          </RevealOnView>
        ) : (
          <div className="mt-14 space-y-12">
            {categories.map((category) => (
              <div key={category}>
                <h3 className="font-display text-display-sm text-primary-foreground/95">
                  {category}
                </h3>
                <StaggerGroup
                  className="mt-5 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5"
                  staggerStep={0.06}
                  y={14}
                >
                  {listings[category].slice(0, 4).map((p) => (
                    <StaggerItem key={p.id}>
                      <Link
                        href={`/catalogo/producto/${p.slug}`}
                        className="group relative block aspect-square overflow-hidden rounded-lg shadow-card"
                      >
                        <Image
                          src={productImage(p)}
                          alt={`Imagen de ${p.name}`}
                          fill
                          sizes="(min-width: 1024px) 20vw, (min-width: 640px) 33vw, 50vw"
                          className="object-cover transition-transform duration-500 group-hover:scale-105"
                        />
                        <div className="absolute inset-x-0 bottom-0 bg-card/95 p-3 backdrop-blur-sm">
                          <p className="line-clamp-2 text-body-sm font-semibold text-foreground">
                            {p.name}
                          </p>
                          <p className="mt-0.5 text-xs font-medium text-accent opacity-0 transition-opacity duration-300 group-hover:opacity-100">
                            Ver detalles
                          </p>
                        </div>
                      </Link>
                    </StaggerItem>
                  ))}
                  <StaggerItem>
                    <Link
                      href={`/catalogo?categoria=${encodeURIComponent(category)}`}
                      className="group flex aspect-square flex-col items-center justify-center gap-2 rounded-lg bg-card/95 p-4 text-center shadow-card backdrop-blur-sm transition-colors hover:bg-card"
                    >
                      <span className="text-body-sm font-semibold uppercase tracking-wide text-foreground">
                        Ver todos
                      </span>
                      <ArrowRight
                        className="h-5 w-5 text-accent transition-transform duration-300 group-hover:translate-x-1"
                        aria-hidden
                      />
                    </Link>
                  </StaggerItem>
                </StaggerGroup>
              </div>
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
