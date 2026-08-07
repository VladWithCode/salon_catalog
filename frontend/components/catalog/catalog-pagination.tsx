import Link from "next/link";

import { buildCatalogPageHref } from "@/lib/catalog/query";
import type { CatalogBrowsePagination } from "@/lib/types";

type CatalogPaginationProps = Readonly<{
  pagination: CatalogBrowsePagination;
  query: string;
  category: string;
  pageSize: number;
}>;

type PaginationItem =
  | Readonly<{ type: "page"; page: number }>
  | Readonly<{ type: "ellipsis"; key: string }>;

function getPaginationItems(
  currentPage: number,
  totalPages: number,
): readonly PaginationItem[] {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, index) => ({
      type: "page" as const,
      page: index + 1,
    }));
  }

  const visiblePages = [
    1,
    currentPage - 1,
    currentPage,
    currentPage + 1,
    totalPages,
  ]
    .filter((page) => page >= 1 && page <= totalPages)
    .filter((page, index, pages) => pages.indexOf(page) === index)
    .sort((left, right) => left - right);

  const items: PaginationItem[] = [];

  visiblePages.forEach((page, index) => {
    const previousPage = visiblePages[index - 1];

    if (previousPage !== undefined && page - previousPage > 1) {
      items.push({ type: "ellipsis", key: `ellipsis-${previousPage}-${page}` });
    }

    items.push({ type: "page", page });
  });

  return items;
}

export function CatalogPagination({
  pagination,
  query,
  category,
  pageSize,
}: CatalogPaginationProps) {
  if (pagination.totalPages <= 1) {
    return null;
  }

  const filters = { query, category, pageSize };
  const items = getPaginationItems(pagination.page, pagination.totalPages);

  return (
    <nav
      aria-label="Paginación del catálogo"
      className="pb-20 pt-4 lg:pb-24"
    >
      <ul className="flex flex-wrap items-center justify-center gap-2">
        {pagination.hasPrevious ? (
          <li>
            <Link
              href={buildCatalogPageHref(filters, pagination.page - 1)}
              prefetch={false}
              aria-label="Ir a la página anterior"
              className="type-small inline-flex min-h-11 items-center justify-center rounded-md border border-border px-4 font-medium text-foreground hover:border-accent hover:text-accent focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
            >
              Anterior
            </Link>
          </li>
        ) : null}

        {items.map((item) =>
          item.type === "ellipsis" ? (
            <li key={item.key} aria-hidden="true">
              <span className="inline-flex min-h-11 min-w-11 items-center justify-center">
                …
              </span>
            </li>
          ) : (
            <li key={item.page}>
              <Link
                href={buildCatalogPageHref(filters, item.page)}
                prefetch={false}
                aria-label={`Ir a la página ${item.page}`}
                aria-current={
                  item.page === pagination.page ? "page" : undefined
                }
                className="type-small inline-flex min-h-11 min-w-11 items-center justify-center rounded-md border border-border px-3 font-medium text-foreground hover:border-accent hover:text-accent focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent aria-[current=page]:border-accent aria-[current=page]:bg-accent aria-[current=page]:text-accent-foreground"
              >
                {item.page}
              </Link>
            </li>
          ),
        )}

        {pagination.hasNext ? (
          <li>
            <Link
              href={buildCatalogPageHref(filters, pagination.page + 1)}
              prefetch={false}
              aria-label="Ir a la página siguiente"
              className="type-small inline-flex min-h-11 items-center justify-center rounded-md border border-border px-4 font-medium text-foreground hover:border-accent hover:text-accent focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
            >
              Siguiente
            </Link>
          </li>
        ) : null}
      </ul>
    </nav>
  );
}
