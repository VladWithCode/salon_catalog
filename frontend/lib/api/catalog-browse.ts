import "server-only";

import { getGoAPIBaseURL } from "@/lib/env";
import type {
  CatalogBrowseCategory,
  CatalogBrowseFilters,
  CatalogBrowsePagination,
  CatalogBrowseProduct,
  CatalogCategoriesApiResponse,
  CatalogCategoriesResult,
  CatalogCategoryApiItem,
  CatalogFiltersApi,
  CatalogInvalidParametersApiResponse,
  CatalogPaginationApi,
  CatalogProductApiItem,
  CatalogProductsApiResponse,
  CatalogProductsRequest,
  CatalogProductsResult,
} from "@/lib/types";

const backendTimeoutMilliseconds = 5_000;
const fallbackImageUrl = "/assets/chenacolo_24.jpeg";
const safeFilenamePattern = /^[A-Za-z0-9._:-]+$/;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isNonEmptyTrimmedString(value: unknown): value is string {
  return typeof value === "string" && value.trim().length > 0;
}

function isNonNegativeInteger(value: unknown): value is number {
  return typeof value === "number" && Number.isInteger(value) && value >= 0;
}

function isJSONContentType(contentType: string | null): boolean {
  if (!contentType) {
    return false;
  }
  return contentType.toLowerCase().split(";")[0].trim() === "application/json";
}

function isSafeImageFilename(filename: string): boolean {
  return (
    filename.length > 0 &&
    filename !== "." &&
    filename !== ".." &&
    !filename.includes("/") &&
    !filename.includes("\\") &&
    !filename.includes("\0") &&
    safeFilenamePattern.test(filename)
  );
}

function toCatalogBrowseImageUrl(imageFilename: string | null): string {
  if (imageFilename === null) {
    return fallbackImageUrl;
  }

  const trimmed = imageFilename.trim();
  if (!isSafeImageFilename(trimmed)) {
    return fallbackImageUrl;
  }

  return `/api/catalog/media/${encodeURIComponent(trimmed)}`;
}

function isCatalogCategoryApiItem(
  value: unknown,
): value is CatalogCategoryApiItem {
  if (!isRecord(value)) {
    return false;
  }

  return (
    isNonEmptyTrimmedString(value.id) &&
    isNonEmptyTrimmedString(value.name) &&
    isNonNegativeInteger(value.product_count)
  );
}

function parseCatalogCategoriesApiResponse(
  value: unknown,
): CatalogCategoriesApiResponse | null {
  if (!isRecord(value) || !Array.isArray(value.categories)) {
    return null;
  }

  return {
    categories: value.categories.filter(isCatalogCategoryApiItem),
  };
}

function toCatalogBrowseCategory(
  item: CatalogCategoryApiItem,
): CatalogBrowseCategory {
  return {
    id: item.id.trim(),
    name: item.name.trim(),
    productCount: item.product_count,
  };
}

export async function fetchCatalogCategories(): Promise<CatalogCategoriesResult> {
  try {
    const response = await fetch(
      `${getGoAPIBaseURL()}/api/catalog/categories`,
      {
        method: "GET",
        cache: "no-store",
        redirect: "error",
        credentials: "omit",
        headers: {
          Accept: "application/json",
        },
        signal: AbortSignal.timeout(backendTimeoutMilliseconds),
      },
    );

    if (response.status !== 200 || !isJSONContentType(response.headers.get("Content-Type"))) {
      return { status: "unavailable" };
    }

    const parsed = parseCatalogCategoriesApiResponse(await response.json());
    if (parsed === null) {
      return { status: "unavailable" };
    }

    return {
      status: "success",
      categories: parsed.categories.map(toCatalogBrowseCategory),
    };
  } catch {
    return { status: "unavailable" };
  }
}

function isCatalogProductApiItem(
  value: unknown,
): value is CatalogProductApiItem {
  if (!isRecord(value)) {
    return false;
  }

  return (
    isNonEmptyTrimmedString(value.id) &&
    isNonEmptyTrimmedString(value.name) &&
    isNonEmptyTrimmedString(value.slug) &&
    typeof value.description === "string" &&
    isNonEmptyTrimmedString(value.category_id) &&
    isNonEmptyTrimmedString(value.category_name) &&
    (value.image_filename === null || typeof value.image_filename === "string") &&
    typeof value.available === "boolean"
  );
}

function toCatalogBrowseProduct(
  item: CatalogProductApiItem,
): CatalogBrowseProduct {
  return {
    id: item.id.trim(),
    name: item.name.trim(),
    slug: item.slug.trim(),
    description: item.description,
    categoryId: item.category_id.trim(),
    categoryName: item.category_name.trim(),
    imageUrl: toCatalogBrowseImageUrl(item.image_filename),
    available: item.available,
  };
}

function isCatalogPaginationApi(value: unknown): value is CatalogPaginationApi {
  if (!isRecord(value)) {
    return false;
  }

  const page = value.page;
  const pageSize = value.page_size;
  const totalItems = value.total_items;
  const totalPages = value.total_pages;

  return (
    typeof page === "number" &&
    Number.isInteger(page) &&
    page >= 1 &&
    typeof pageSize === "number" &&
    Number.isInteger(pageSize) &&
    pageSize >= 1 &&
    pageSize <= 100 &&
    isNonNegativeInteger(totalItems) &&
    isNonNegativeInteger(totalPages) &&
    typeof value.has_next === "boolean" &&
    typeof value.has_previous === "boolean"
  );
}

function toCatalogBrowsePagination(
  pagination: CatalogPaginationApi,
): CatalogBrowsePagination {
  return {
    page: pagination.page,
    pageSize: pagination.page_size,
    totalItems: pagination.total_items,
    totalPages: pagination.total_pages,
    hasNext: pagination.has_next,
    hasPrevious: pagination.has_previous,
  };
}

function isCatalogFiltersApi(value: unknown): value is CatalogFiltersApi {
  if (!isRecord(value)) {
    return false;
  }

  return (
    typeof value.query === "string" &&
    (value.category === null || typeof value.category === "string")
  );
}

function toCatalogBrowseFilters(filters: CatalogFiltersApi): CatalogBrowseFilters {
  return {
    query: filters.query,
    category: filters.category,
  };
}

function parseCatalogProductsApiResponse(
  value: unknown,
): CatalogProductsApiResponse | null {
  if (
    !isRecord(value) ||
    !Array.isArray(value.items) ||
    !isCatalogPaginationApi(value.pagination) ||
    !isCatalogFiltersApi(value.filters)
  ) {
    return null;
  }

  return {
    items: value.items.filter(isCatalogProductApiItem),
    pagination: value.pagination,
    filters: value.filters,
  };
}

function parseCatalogInvalidParametersApiResponse(
  value: unknown,
): CatalogInvalidParametersApiResponse | null {
  if (!isRecord(value) || value.error !== "invalid_parameters" || !isRecord(value.fields)) {
    return null;
  }

  const rawFields = value.fields;
  const fields: { pagina?: string; por_pagina?: string } = {};

  if (typeof rawFields.pagina === "string") {
    fields.pagina = rawFields.pagina;
  }
  if (typeof rawFields.por_pagina === "string") {
    fields.por_pagina = rawFields.por_pagina;
  }

  return {
    error: "invalid_parameters",
    fields,
  };
}

function buildCatalogProductsSearchParams(
  request: CatalogProductsRequest,
): URLSearchParams {
  const params = new URLSearchParams();

  if (request.query !== "") {
    params.set("buscar", request.query);
  }
  if (request.category !== "") {
    params.set("categoria", request.category);
  }
  params.set("pagina", String(request.page));
  params.set("por_pagina", String(request.pageSize));

  return params;
}

export async function fetchCatalogProducts(
  request: CatalogProductsRequest,
): Promise<CatalogProductsResult> {
  try {
    const params = buildCatalogProductsSearchParams(request);
    const response = await fetch(
      `${getGoAPIBaseURL()}/api/catalog/products?${params.toString()}`,
      {
        method: "GET",
        cache: "no-store",
        redirect: "error",
        credentials: "omit",
        headers: {
          Accept: "application/json",
        },
        signal: AbortSignal.timeout(backendTimeoutMilliseconds),
      },
    );

    if (!isJSONContentType(response.headers.get("Content-Type"))) {
      return { status: "unavailable" };
    }

    if (response.status === 200) {
      const parsed = parseCatalogProductsApiResponse(await response.json());
      if (parsed === null) {
        return { status: "unavailable" };
      }

      return {
        status: "success",
        items: parsed.items.map(toCatalogBrowseProduct),
        pagination: toCatalogBrowsePagination(parsed.pagination),
        filters: toCatalogBrowseFilters(parsed.filters),
      };
    }

    if (response.status === 400) {
      const parsed = parseCatalogInvalidParametersApiResponse(
        await response.json(),
      );
      if (parsed === null) {
        return { status: "unavailable" };
      }

      return {
        status: "invalid_parameters",
        fields: parsed.fields,
      };
    }

    return { status: "unavailable" };
  } catch {
    return { status: "unavailable" };
  }
}
