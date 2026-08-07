import Link from "next/link";

import { buildCatalogHref } from "@/lib/catalog/query";
import type { CatalogCategoriesResult } from "@/lib/types";
import { cn } from "@/lib/utils";

type CategoryFilterProps = Readonly<{
  categoriesResult: CatalogCategoriesResult;
  query: string;
  category: string;
  pageSize: number;
}>;

function categoryLinkClass(isActive: boolean): string {
  return cn(
    "type-small inline-flex min-h-11 items-center whitespace-nowrap rounded-md border px-4 font-medium transition-colors",
    isActive
      ? "border-primary bg-primary text-primary-foreground"
      : "border-border bg-card text-foreground hover:border-accent hover:text-accent-strong",
  );
}

export function CategoryFilter({
  categoriesResult,
  query,
  category,
  pageSize,
}: CategoryFilterProps) {
  if (categoriesResult.status === "unavailable") {
    return (
      <p role="alert" className="type-small text-destructive">
        No pudimos cargar las categorías en este momento.
      </p>
    );
  }

  return (
    <nav aria-label="Categorías del catálogo" className="space-y-3">
      <p className="type-small font-medium">Filtrar por categoría</p>
      <ul className="flex max-w-full gap-2 overflow-x-auto pb-2 lg:flex-wrap lg:overflow-visible lg:pb-0">
        <li>
          <Link
            href={buildCatalogHref({ query, category: "", pageSize })}
            prefetch={false}
            aria-current={category === "" ? "page" : undefined}
            className={categoryLinkClass(category === "")}
          >
            Todos
          </Link>
        </li>
        {categoriesResult.categories.map((item) => {
          const isActive = category === item.name || category === item.id;

          return (
            <li key={item.id}>
              <Link
                href={buildCatalogHref({
                  query,
                  category: item.name,
                  pageSize,
                })}
                prefetch={false}
                aria-current={isActive ? "page" : undefined}
                className={categoryLinkClass(isActive)}
              >
                {item.name}
              </Link>
            </li>
          );
        })}
      </ul>
      {categoriesResult.categories.length === 0 ? (
        <p role="status" className="type-small text-muted-foreground">
          No hay categorías disponibles.
        </p>
      ) : null}
    </nav>
  );
}
