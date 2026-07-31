/**
 * Shared types — mirror the Go backend's JSON shapes.
 * Keep in sync with /home/vladwb/developer/salon_catalog/internal/db/*.go
 *
 * Field names use the same casing as Go's `json:"…"` struct tags so the
 * runtime shape matches what the API actually returns.
 */

export type SocialLink = {
  id: string;
  name: string;
  link: string;
  /** Optional display label override; falls back to `name`. */
  label?: string;
  /** Optional icon hint; the React side picks the SVG based on this. */
  platform?: "facebook" | "instagram" | "whatsapp" | "tiktok" | "youtube" | "x";
};

export type Category = {
  id: string;
  name: string;
  slug: string;
  description: string;
  long_description: string;
  header_img: string;
  header_img_id: string;
  display_img: string;
  display_img_id: string;
  product_count: number;
  qrcode_filename: string;
};

export type Product = {
  id: string;
  name: string;
  slug: string;
  description: string;
  long_description: string;
  main_img: string;
  main_img_id: string;
  gallery: string[];
  gallery_ids: string[];
  category: string;
  category_id: string;
  available: boolean;
  quantity: number;
  qrcode_filename: string;
};

/**
 * Slim product used in listings (home catalog preview, cross-sell rows).
 * Mirrors `db.CatalogProd` from the Go project.
 */
export type CatalogProd = {
  id: string;
  name: string;
  slug: string;
  description: string;
  long_description: string;
  category: string;
  category_id: string;
  image_url: string;
  images: string[];
  available: boolean;
  quantity: number;
};

/** `GET /api/catalog/listings` response — keyed by category name. */
export type CatalogListings = Record<string, CatalogProd[]>;

export type EventKind = {
  id: string;
  name: string;
  description: string;
};

export type ContactRequest = {
  name: string;
  phone: string;
  eventDate?: string;
  message?: string;
};

export type ContactRequestResponse = {
  ok: boolean;
  id?: string;
  error?: string;
};
