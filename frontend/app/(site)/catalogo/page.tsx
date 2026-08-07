import type { Metadata } from "next";

import { CatalogFilters } from "@/components/catalog/catalog-filters";
import { CatalogPageHero } from "@/components/catalog/catalog-page-hero";
import { CatalogResults } from "@/components/catalog/catalog-results";
import {
  fetchCatalogCategories,
  fetchCatalogProducts,
} from "@/lib/api/catalog-browse";
import {
  buildCatalogHref,
  parseCatalogSearchParams,
  readCatalogURLFilters,
  type CatalogSearchParams,
} from "@/lib/catalog/query";
import type { CatalogProductsResult } from "@/lib/types";

export const metadata: Metadata = {
  title: "Catálogo",
  description:
    "Mobiliario y piezas decorativas disponibles para eventos en Villa Chenacolo.",
};

type CatalogPageProps = Readonly<{
  searchParams: Promise<CatalogSearchParams>;
}>;

export default async function CatalogPage({ searchParams }: CatalogPageProps) {
  const categoriesPromise = fetchCatalogCategories();
  const rawSearchParams = await searchParams;
  const parsedQuery = parseCatalogSearchParams(rawSearchParams);
  const urlFilters = readCatalogURLFilters(rawSearchParams);

  let categoriesResult;
  let productsResult: CatalogProductsResult;

  if (parsedQuery.status === "valid") {
    [categoriesResult, productsResult] = await Promise.all([
      categoriesPromise,
      fetchCatalogProducts(parsedQuery.request),
    ]);
  } else {
    categoriesResult = await categoriesPromise;
    productsResult = {
      status: "invalid_parameters",
      fields: parsedQuery.fields,
    };
  }

  const hasFilters = urlFilters.query !== "" || urlFilters.category !== "";
  const resetHref = buildCatalogHref({
    query: urlFilters.query,
    category: urlFilters.category,
    pageSize: 16,
  });

  return (
    <article className="overflow-x-clip">
      <CatalogPageHero />
      <div className="section-light py-section">
        <div className="container-main max-w-7xl space-y-section">
          <CatalogFilters
            categoriesResult={categoriesResult}
            query={urlFilters.query}
            category={urlFilters.category}
            pageSize={urlFilters.pageSize}
          />
          <CatalogResults
            result={productsResult}
            hasFilters={hasFilters}
            resetHref={resetHref}
          />
        </div>
      </div>
    </article>
  );
}
