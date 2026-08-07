import Link from "next/link";

import { CatalogPagination } from "@/components/catalog/catalog-pagination";
import { CatalogProductCard } from "@/components/catalog/catalog-product-card";
import {
  buildCatalogHref,
  buildCatalogPageHref,
} from "@/lib/catalog/query";
import type { CatalogProductsResult } from "@/lib/types";

type CatalogResultsProps = Readonly<{
  result: CatalogProductsResult;
  hasFilters: boolean;
  resetHref: string;
}>;

type EmptyCatalogProps = Readonly<{
  hasFilters: boolean;
  query: string;
  category: string;
  pageSize: number;
}>;

function EmptyCatalog({
  hasFilters,
  query,
  category,
  pageSize,
}: EmptyCatalogProps) {
  let message: React.ReactNode;
  let actionHref: string;
  let actionLabel: string;

  if (query !== "") {
    message = <>No encontramos productos para “{query}”.</>;
    actionHref = buildCatalogHref({ query: "", category, pageSize });
    actionLabel = "Limpiar búsqueda";
  } else if (category !== "") {
    message = <>No encontramos productos en esta categoría.</>;
    actionHref = buildCatalogHref({ query, category: "", pageSize });
    actionLabel = "Ver todas las categorías";
  } else if (hasFilters) {
    message = <>No encontramos productos con los filtros seleccionados.</>;
    actionHref = "/catalogo";
    actionLabel = "Ver todo el catálogo";
  } else {
    message = <>El catálogo aún no tiene productos disponibles.</>;
    actionHref = "/solicitar-cotizacion";
    actionLabel = "Solicitar cotización";
  }

  return (
    <div role="status" className="space-y-4 border-y border-border py-8">
      <p className="type-body text-muted-foreground">{message}</p>
      <Link
        href={actionHref}
        prefetch={false}
        className="type-small inline-flex min-h-11 items-center font-medium text-accent-strong underline decoration-accent-strong/40 underline-offset-4 hover:decoration-accent-strong"
      >
        {actionLabel}
      </Link>
    </div>
  );
}

export function CatalogResults({
  result,
  hasFilters,
  resetHref,
}: CatalogResultsProps) {
  return (
    <section aria-labelledby="catalog-results-title" className="space-y-6">
      <div className="space-y-2">
        <p className="type-eyebrow">Catálogo</p>
        <h2 id="catalog-results-title" className="type-h2 font-medium">
          Resultados
        </h2>
      </div>

      {result.status === "unavailable" ? (
        <p
          role="alert"
          className="type-body border-y border-border py-8 text-destructive"
        >
          No pudimos cargar el catálogo en este momento. Inténtalo nuevamente
          más tarde.
        </p>
      ) : null}

      {result.status === "invalid_parameters" ? (
        <div role="alert" className="space-y-4 border-y border-border py-8">
          <p className="type-body font-medium">
            Revisa los parámetros del catálogo.
          </p>
          <ul className="type-small list-disc space-y-2 pl-5 text-destructive">
            {result.fields.pagina ? (
              <li>Página: {result.fields.pagina}</li>
            ) : null}
            {result.fields.por_pagina ? (
              <li>Productos por página: {result.fields.por_pagina}</li>
            ) : null}
          </ul>
          <Link
            href={resetHref}
            prefetch={false}
            className="type-small inline-flex min-h-11 items-center font-medium text-accent-strong underline decoration-accent-strong/40 underline-offset-4 hover:decoration-accent-strong"
          >
            Volver a una página válida
          </Link>
        </div>
      ) : null}

      {result.status === "success" ? (
        <>
          <div className="type-small text-muted-foreground">
            <p>
              {result.pagination.totalItems}{" "}
              {result.pagination.totalItems === 1 ? "producto" : "productos"}
            </p>
            {result.pagination.totalItems > 0 ? (
              <p>
                Página {result.pagination.page} de {result.pagination.totalPages}
              </p>
            ) : null}
          </div>

          {result.items.length === 0 &&
          result.pagination.totalItems > 0 &&
          result.pagination.page > result.pagination.totalPages ? (
            <div
              role="status"
              className="space-y-4 border-y border-border py-8"
            >
              <p className="type-body text-muted-foreground">
                No encontramos productos en esta página.
              </p>
              <Link
                href={buildCatalogPageHref(
                  {
                    query: result.filters.query,
                    category: result.filters.category ?? "",
                    pageSize: result.pagination.pageSize,
                  },
                  1,
                )}
                prefetch={false}
                className="type-small inline-flex min-h-11 items-center font-medium text-accent-strong underline decoration-accent-strong/40 underline-offset-4 hover:decoration-accent-strong"
              >
                Volver a la primera página
              </Link>
            </div>
          ) : null}

          {result.items.length === 0 && result.pagination.totalItems === 0 ? (
            <EmptyCatalog
              hasFilters={hasFilters}
              query={result.filters.query}
              category={result.filters.category ?? ""}
              pageSize={result.pagination.pageSize}
            />
          ) : null}

          {result.items.length > 0 ? (
            <ul className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              {result.items.map((product) => (
                <li key={product.id} className="min-w-0">
                  <CatalogProductCard product={product} />
                </li>
              ))}
            </ul>
          ) : null}

          {result.items.length > 0 ? (
            <CatalogPagination
              pagination={result.pagination}
              query={result.filters.query}
              category={result.filters.category ?? ""}
              pageSize={result.pagination.pageSize}
            />
          ) : null}
        </>
      ) : null}
    </section>
  );
}
