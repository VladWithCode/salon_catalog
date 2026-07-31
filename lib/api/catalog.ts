import "server-only";
import { apiFetch } from "./client";
import type { CatalogListings } from "@/lib/types";

/**
 * Fetches catalog listings grouped by category, capped to 4 products per category.
 * Used by the home page's catalog preview.
 */
export async function getCatalogListings(): Promise<CatalogListings> {
  return apiFetch<CatalogListings>("/api/catalog/listings", {
    next: { revalidate: 300, tags: ["catalog", "listings"] },
  });
}
