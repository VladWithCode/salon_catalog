import { DEFAULT_CATALOG_PAGE_SIZE } from "@/lib/catalog/query";

type CatalogSearchFormProps = Readonly<{
  query: string;
  category: string;
  pageSize: number;
}>;

export function CatalogSearchForm({
  query,
  category,
  pageSize,
}: CatalogSearchFormProps) {
  return (
    <form action="/catalogo" method="get" role="search" className="space-y-3">
      <label htmlFor="catalog-search" className="type-small block font-medium">
        Buscar en el catálogo
      </label>
      <div className="flex flex-col gap-3 sm:flex-row">
        <input
          id="catalog-search"
          name="buscar"
          type="search"
          defaultValue={query}
          autoComplete="off"
          className="type-body min-h-12 min-w-0 flex-1 rounded-md border border-input bg-card px-4 text-foreground shadow-soft placeholder:text-muted-foreground"
          placeholder="Buscar productos"
        />
        <button
          type="submit"
          className="type-button min-h-12 rounded-md bg-primary px-6 font-medium text-primary-foreground shadow-soft hover:bg-primary/90"
        >
          Buscar
        </button>
      </div>
      {category !== "" ? (
        <input type="hidden" name="categoria" value={category} />
      ) : null}
      {pageSize !== DEFAULT_CATALOG_PAGE_SIZE ? (
        <input type="hidden" name="por_pagina" value={pageSize} />
      ) : null}
    </form>
  );
}
