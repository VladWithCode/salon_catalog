import Link from "next/link";

import { CatalogSearchForm } from "@/components/catalog/catalog-search-form";
import { CategoryFilter } from "@/components/catalog/category-filter";
import { DEFAULT_CATALOG_PAGE_SIZE } from "@/lib/catalog/query";
import type { CatalogCategoriesResult } from "@/lib/types";

type CatalogFiltersProps = Readonly<{
  categoriesResult: CatalogCategoriesResult;
  query: string;
  category: string;
  pageSize: number;
}>;

export function CatalogFilters({
  categoriesResult,
  query,
  category,
  pageSize,
}: CatalogFiltersProps) {
  const hasActiveState =
    query !== "" || category !== "" || pageSize !== DEFAULT_CATALOG_PAGE_SIZE;

  return (
    <section aria-labelledby="catalog-filters-title" className="space-y-8">
      <div className="space-y-2">
        <p className="type-eyebrow">Explorar</p>
        <h2 id="catalog-filters-title" className="type-h3 font-medium">
          Filtros del catálogo
        </h2>
      </div>
      <CatalogSearchForm
        query={query}
        category={category}
        pageSize={pageSize}
      />
      <CategoryFilter
        categoriesResult={categoriesResult}
        query={query}
        category={category}
        pageSize={pageSize}
      />
      {hasActiveState ? (
        <Link
          href="/catalogo"
          prefetch={false}
          className="type-small inline-flex min-h-11 items-center font-medium text-accent underline decoration-accent/40 underline-offset-4 hover:decoration-accent"
        >
          Limpiar búsqueda y filtros
        </Link>
      ) : null}
    </section>
  );
}
