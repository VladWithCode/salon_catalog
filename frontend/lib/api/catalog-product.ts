import "server-only";

import { getGoAPIBaseURL } from "@/lib/env";
import type {
  CatalogProductDetail,
  CatalogProductDetailApiCategory,
  CatalogProductDetailApiResponse,
  CatalogProductDetailCategory,
  CatalogProductDetailResult,
} from "@/lib/types";

const backendTimeoutMilliseconds = 5_000;

// Mirrors internal/routes/catalog_product_api.go's
// catalogProductIdentifierMaxLength, itself derived from products.slug's
// VARCHAR(200) column (sql/migrations/20250703200655_add_products_table.sql).
const identifierMaxLength = 200;

// Mirrors catalogProductImageFilenamePattern in
// internal/routes/catalog_product_api.go exactly (ASCII-only), not the
// Unicode pattern frontend/app/api/catalog/media/[filename]/route.ts uses —
// that inconsistency already exists between the Go endpoint and the Next
// image proxy and is not introduced or resolved here (see 6B2's delivery
// report). This fetcher validates against the same pattern the Go endpoint
// itself already enforced before ever returning a filename.
const safeImageFilenamePattern = /^[A-Za-z0-9._:-]+$/;

// UUID shape check, used only to validate the id field of a parsed
// response — not to decide whether the caller's identifier is a UUID or a
// slug (that decision belongs entirely to Go's resolver; this fetcher never
// re-implements it or branches on it).
const uuidPattern =
  /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;
const nilUUID = "00000000-0000-0000-0000-000000000000";

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isValidNonNilUUID(value: unknown): value is string {
  return typeof value === "string" && uuidPattern.test(value) && value !== nilUUID;
}

function isNonEmptyTrimmedString(value: unknown): value is string {
  return typeof value === "string" && value.trim().length > 0;
}

function isSafeImageFilename(value: unknown): value is string {
  if (typeof value !== "string" || value.length === 0) {
    return false;
  }
  if (value === "." || value === "..") {
    return false;
  }
  if (value.includes("/") || value.includes("\\") || value.includes("\0")) {
    return false;
  }
  return safeImageFilenamePattern.test(value);
}

/**
 * Counts Unicode characters (code points), matching Go's
 * utf8.RuneCountInString — JavaScript's string.length counts UTF-16 code
 * units instead, which would undercount astral-plane characters and
 * overcount nothing, but is still the wrong unit for this comparison.
 */
function unicodeCharacterCount(value: string): number {
  return Array.from(value).length;
}

/**
 * Validates the identifier the way internal/routes/catalog_product_api.go's
 * isValidCatalogProductIdentifier does: non-empty after trimming only to
 * detect emptiness, no NUL or other control bytes, no path separators, and
 * at most 200 Unicode characters. The original, untrimmed value is what
 * gets encoded into the URL — this function never corrects or normalizes
 * it.
 */
function isValidCatalogProductIdentifier(identifier: string): boolean {
  if (identifier.trim().length === 0) {
    return false;
  }
  if (identifier.includes("/") || identifier.includes("\\")) {
    return false;
  }
  for (const char of identifier) {
    const codePoint = char.codePointAt(0) ?? 0;
    if (codePoint === 0 || codePoint < 0x20 || codePoint === 0x7f) {
      return false;
    }
  }
  return unicodeCharacterCount(identifier) <= identifierMaxLength;
}

function isCatalogProductDetailApiCategory(
  value: unknown,
): value is CatalogProductDetailApiCategory {
  if (!isRecord(value)) {
    return false;
  }
  return isValidNonNilUUID(value.id) && isNonEmptyTrimmedString(value.name);
}

function isCatalogProductDetailApiResponse(
  value: unknown,
): value is CatalogProductDetailApiResponse {
  if (!isRecord(value) || !isRecord(value.product)) {
    return false;
  }

  const product = value.product;

  if (!isValidNonNilUUID(product.id)) {
    return false;
  }
  if (!isNonEmptyTrimmedString(product.name)) {
    return false;
  }
  if (
    typeof product.slug !== "string" ||
    product.slug.trim().length === 0 ||
    unicodeCharacterCount(product.slug) > identifierMaxLength
  ) {
    return false;
  }
  if (typeof product.description !== "string") {
    return false;
  }
  if (typeof product.long_description !== "string") {
    return false;
  }
  if (product.category !== null && !isCatalogProductDetailApiCategory(product.category)) {
    return false;
  }
  if (typeof product.available !== "boolean") {
    return false;
  }
  if (product.image_filename !== null && !isSafeImageFilename(product.image_filename)) {
    return false;
  }
  if (!Array.isArray(product.images)) {
    return false;
  }
  // Every element must already satisfy the Go contract's own filename
  // safety check; unlike the catalog-listing fetchers, an invalid element
  // here disqualifies the whole response rather than being silently
  // dropped (per this phase's explicit instruction).
  if (!product.images.every((image) => isSafeImageFilename(image))) {
    return false;
  }

  return true;
}

function toCatalogProductDetailCategory(
  category: CatalogProductDetailApiCategory,
): CatalogProductDetailCategory {
  return { id: category.id, name: category.name };
}

function toCatalogProductDetail(
  response: CatalogProductDetailApiResponse,
): CatalogProductDetail {
  const product = response.product;
  return {
    id: product.id,
    name: product.name,
    slug: product.slug,
    description: product.description,
    longDescription: product.long_description,
    category: product.category === null ? null : toCatalogProductDetailCategory(product.category),
    available: product.available,
    imageFilename: product.image_filename,
    images: product.images,
  };
}

function isJSONContentType(contentType: string | null): boolean {
  if (!contentType) {
    return false;
  }
  return contentType.toLowerCase().split(";")[0].trim() === "application/json";
}

/**
 * Fetches a single product's public detail from Go's
 * GET /api/catalog/products/{identifier} — server-only, same-origin
 * server-to-server call, never the legacy GET /api/products/{slug}. The
 * caller (a future page's params, in 6B4 — not implemented here) supplies
 * identifier directly; this function never reads it from a cookie, query
 * string, or header.
 */
export async function fetchCatalogProductDetail(
  identifier: string,
): Promise<CatalogProductDetailResult> {
  if (!isValidCatalogProductIdentifier(identifier)) {
    return { status: "error", code: "invalid_identifier" };
  }

  let response: Response;
  try {
    response = await fetch(
      `${getGoAPIBaseURL()}/api/catalog/products/${encodeURIComponent(identifier)}`,
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
  } catch {
    // Backend down, DNS failure, connection refused, or the timeout above
    // firing — none of these carry detail worth exposing to a caller.
    return { status: "error", code: "backend_unavailable" };
  }

  if (response.status === 400) {
    return { status: "error", code: "invalid_identifier" };
  }
  if (response.status === 404) {
    return { status: "error", code: "product_not_found" };
  }
  if (response.status === 503) {
    return { status: "error", code: "catalog_unavailable" };
  }
  if (response.status !== 200) {
    return { status: "error", code: "unexpected_status" };
  }

  if (!isJSONContentType(response.headers.get("Content-Type"))) {
    return { status: "error", code: "invalid_response" };
  }

  let body: unknown;
  try {
    body = await response.json();
  } catch {
    return { status: "error", code: "invalid_response" };
  }

  if (!isCatalogProductDetailApiResponse(body)) {
    return { status: "error", code: "invalid_response" };
  }

  return { status: "success", product: toCatalogProductDetail(body) };
}
