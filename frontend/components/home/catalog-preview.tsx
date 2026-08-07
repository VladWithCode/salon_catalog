import { ArrowRight } from "lucide-react";
import Image from "next/image";
import Link from "next/link";

import { ProductCard } from "@/components/home/product-card";
import { SectionHeading } from "@/components/shared/section-heading";
import { homeCopy } from "@/lib/copy/home";
import type {
  CatalogListingsStatus,
  CatalogPreviewCategory,
} from "@/lib/types";

type CatalogPreviewProps = Readonly<{
  status: CatalogListingsStatus;
  categories: readonly CatalogPreviewCategory[];
}>;

export function CatalogPreview({ status, categories }: CatalogPreviewProps) {
  const { catalog } = homeCopy;
  const categoriesWithProducts = categories.filter(
    (category) => category.products.length > 0,
  );

  return (
    <section
      id="seccion-catalogo"
      aria-labelledby="catalog-preview-title"
      className="relative overflow-hidden py-section text-primary-foreground"
    >
      <Image
        src={catalog.background.src}
        alt={catalog.background.alt}
        fill
        sizes="100vw"
        className="object-cover"
      />
      <div className="absolute inset-0 bg-gradient-to-tr from-primary/85 via-primary/70 to-accent/45" />

      <div className="container-main relative z-10 max-w-7xl space-block">
        <SectionHeading
          eyebrow={catalog.eyebrow}
          title={
            <span id="catalog-preview-title">
              {catalog.titleBefore}
              <em className="font-normal">{catalog.titleEmphasis}</em>
              {catalog.titleAfter}
            </span>
          }
          lede={catalog.lede}
          inverted
        />

        {status === "unavailable" ? (
          <div
            role="status"
            className="rounded-xl border border-primary-foreground/25 bg-primary/75 p-8 shadow-elevated backdrop-blur-sm sm:p-10"
          >
            <p className="type-lead max-w-2xl text-primary-foreground/85">
              No pudimos cargar el catálogo en este momento. Puedes consultar el
              catálogo completo.
            </p>
          </div>
        ) : categoriesWithProducts.length > 0 ? (
          <div className="space-block">
            {categoriesWithProducts.map((category, categoryIndex) => (
              <section
                key={category.name}
                aria-labelledby={`catalog-category-${categoryIndex}`}
              >
                <div className="mb-5 flex items-center justify-between gap-4">
                  <h3
                    id={`catalog-category-${categoryIndex}`}
                    className="type-h3"
                  >
                    {category.name}
                  </h3>
                  <Link
                    href={`/catalogo?categoria=${encodeURIComponent(category.name)}`}
                    prefetch={false}
                    className="type-small inline-flex min-h-11 shrink-0 items-center gap-2 rounded-sm font-semibold hover:text-accent focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent"
                  >
                    Ver todos
                    <ArrowRight aria-hidden="true" className="size-4" />
                  </Link>
                </div>
                <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
                  {category.products.slice(0, 4).map((product) => (
                    <ProductCard key={product.id} product={product} />
                  ))}
                </div>
              </section>
            ))}
          </div>
        ) : (
          <div
            role="status"
            className="rounded-xl border border-primary-foreground/25 bg-primary/75 p-8 shadow-elevated backdrop-blur-sm sm:p-10"
          >
            <p className="type-lead max-w-2xl text-primary-foreground/85">
              El catálogo aún no tiene productos disponibles para mostrar.
            </p>
          </div>
        )}

        <div>
          <Link
            href={catalog.cta.href}
            prefetch={false}
            className="type-button inline-flex min-h-12 items-center gap-2 rounded-md bg-accent px-6 font-semibold uppercase text-accent-foreground hover:bg-secondary focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent"
          >
            {catalog.cta.label}
            <ArrowRight aria-hidden="true" className="size-4" />
          </Link>
        </div>
      </div>
    </section>
  );
}
