import "server-only";

import { getGoAPIBaseURL } from "@/lib/env";
import type {
  CatalogListingApiCategory,
  CatalogListingApiProduct,
  CatalogListingsApiResponse,
  CatalogListingsStatus,
  CatalogPreviewCategory,
} from "@/lib/types";

const backendTimeoutMilliseconds = 5_000;
const fallbackImageUrl = "/assets/chenacolo_24.jpeg";
const safeFilenamePattern = /^[\p{L}\p{N}._:-]+$/u;

export type CatalogListingsResult = Readonly<{
  status: CatalogListingsStatus;
  categories: readonly CatalogPreviewCategory[];
}>;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isCatalogListingApiProduct(
  value: unknown,
): value is CatalogListingApiProduct {
  if (!isRecord(value)) {
    return false;
  }

  return (
    typeof value.id === "string" &&
    value.id.trim().length > 0 &&
    typeof value.name === "string" &&
    value.name.trim().length > 0 &&
    typeof value.slug === "string" &&
    value.slug.trim().length > 0 &&
    typeof value.description === "string" &&
    (value.image_filename === null ||
      typeof value.image_filename === "string")
  );
}

function parseCatalogListingApiCategory(
  value: unknown,
): CatalogListingApiCategory | null {
  if (
    !isRecord(value) ||
    typeof value.name !== "string" ||
    value.name.trim().length === 0 ||
    !Array.isArray(value.products)
  ) {
    return null;
  }

  return {
    name: value.name,
    products: value.products.filter(isCatalogListingApiProduct),
  };
}

function parseCatalogListingsApiResponse(
  value: unknown,
): CatalogListingsApiResponse | null {
  if (!isRecord(value) || !Array.isArray(value.categories)) {
    return null;
  }

  return {
    categories: value.categories
      .map(parseCatalogListingApiCategory)
      .filter((category) => category !== null),
  };
}

function isSafeFilename(filename: string): boolean {
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

function toImageUrl(imageFilename: string | null): string {
  if (imageFilename === null || !isSafeFilename(imageFilename)) {
    return fallbackImageUrl;
  }

  return `/api/catalog/media/${encodeURIComponent(imageFilename)}`;
}

function compareByName(
  left: { readonly name: string },
  right: { readonly name: string },
): number {
  return left.name.localeCompare(right.name, "es", { sensitivity: "base" });
}

export async function getCatalogListings(): Promise<CatalogListingsResult> {
  try {
    const response = await fetch(
      `${getGoAPIBaseURL()}/api/catalog/listings`,
      {
        method: "GET",
        cache: "no-store",
        credentials: "omit",
        headers: {
          Accept: "application/json",
        },
        signal: AbortSignal.timeout(backendTimeoutMilliseconds),
      },
    );

    if (!response.ok) {
      return { status: "unavailable", categories: [] };
    }

    const parsed = parseCatalogListingsApiResponse(await response.json());
    if (parsed === null) {
      return { status: "unavailable", categories: [] };
    }

    const categories = parsed.categories
      .map((category) => ({
        name: category.name,
        products: category.products
          .map((product) => ({
            id: product.id,
            name: product.name,
            slug: product.slug,
            description: product.description,
            imageUrl: toImageUrl(product.image_filename),
          }))
          .sort(compareByName)
          .slice(0, 4),
      }))
      .sort(compareByName);

    return { status: "success", categories };
  } catch {
    return { status: "unavailable", categories: [] };
  }
}
