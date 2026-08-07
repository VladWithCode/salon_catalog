import type { CatalogProductsRequest } from "@/lib/types";

export const DEFAULT_CATALOG_PAGE_SIZE = 16;

export type CatalogSearchParams = Readonly<
  Record<string, string | readonly string[] | undefined>
>;

export type CatalogQueryResult =
  | Readonly<{
      status: "valid";
      request: CatalogProductsRequest;
    }>
  | Readonly<{
      status: "invalid";
      fields: Readonly<{
        pagina?: string;
        por_pagina?: string;
      }>;
    }>;

export type CatalogURLFilters = Readonly<{
  query: string;
  category: string;
  pageSize: number;
}>;

function buildCatalogURLSearchParams({
  query,
  category,
  pageSize,
}: CatalogURLFilters): URLSearchParams {
  const searchParams = new URLSearchParams();

  if (query !== "") {
    searchParams.set("buscar", query);
  }

  if (category !== "") {
    searchParams.set("categoria", category);
  }

  if (pageSize !== DEFAULT_CATALOG_PAGE_SIZE) {
    searchParams.set("por_pagina", String(pageSize));
  }

  return searchParams;
}

type IntegerResult =
  | Readonly<{ status: "valid"; value: number }>
  | Readonly<{ status: "invalid" }>;

function getFirstValue(value: string | readonly string[] | undefined): string {
  if (typeof value === "string") {
    return value;
  }

  return value?.[0] ?? "";
}

function parseInteger(
  value: string,
  defaultValue: number,
  maximum?: number,
): IntegerResult {
  if (value === "") {
    return { status: "valid", value: defaultValue };
  }

  if (!/^\d+$/.test(value)) {
    return { status: "invalid" };
  }

  const parsed = Number(value);
  if (
    !Number.isSafeInteger(parsed) ||
    parsed < 1 ||
    (maximum !== undefined && parsed > maximum)
  ) {
    return { status: "invalid" };
  }

  return { status: "valid", value: parsed };
}

export function readCatalogURLFilters(
  searchParams: CatalogSearchParams,
): CatalogURLFilters {
  const rawPageSize = getFirstValue(searchParams.por_pagina).trim();
  const pageSize = parseInteger(
    rawPageSize,
    DEFAULT_CATALOG_PAGE_SIZE,
    100,
  );

  return {
    query: getFirstValue(searchParams.buscar).trim(),
    category: getFirstValue(searchParams.categoria).trim(),
    pageSize:
      pageSize.status === "valid"
        ? pageSize.value
        : DEFAULT_CATALOG_PAGE_SIZE,
  };
}

export function parseCatalogSearchParams(
  searchParams: CatalogSearchParams,
): CatalogQueryResult {
  const filters = readCatalogURLFilters(searchParams);
  const rawPage = getFirstValue(searchParams.pagina).trim();
  const rawPageSize = getFirstValue(searchParams.por_pagina).trim();
  const page = parseInteger(rawPage, 1);
  const pageSize = parseInteger(
    rawPageSize,
    DEFAULT_CATALOG_PAGE_SIZE,
    100,
  );
  const fields: {
    pagina?: string;
    por_pagina?: string;
  } = {};

  if (page.status === "invalid") {
    fields.pagina = "Debe ser un entero mayor o igual a 1.";
  }

  if (pageSize.status === "invalid") {
    fields.por_pagina = "Debe ser un entero entre 1 y 100.";
  }

  if (page.status === "invalid" || pageSize.status === "invalid") {
    return { status: "invalid", fields };
  }

  return {
    status: "valid",
    request: {
      query: filters.query,
      category: filters.category,
      page: page.value,
      pageSize: pageSize.value,
    },
  };
}

export function buildCatalogHref({
  query,
  category,
  pageSize,
}: CatalogURLFilters): string {
  const searchParams = buildCatalogURLSearchParams({ query, category, pageSize });

  const queryString = searchParams.toString();
  return queryString === "" ? "/catalogo" : `/catalogo?${queryString}`;
}

export function buildCatalogPageHref(
  filters: CatalogURLFilters,
  page: number,
): string {
  const searchParams = buildCatalogURLSearchParams(filters);

  if (page > 1) {
    searchParams.set("pagina", String(page));
  }

  const queryString = searchParams.toString();
  return queryString === "" ? "/catalogo" : `/catalogo?${queryString}`;
}
