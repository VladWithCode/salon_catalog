export type SocialLink = Readonly<{
  name: string;
  link: string;
}>;

export type ImageAsset = Readonly<{
  src: string;
  alt: string;
  width: number;
  height: number;
}>;

export type CatalogListingApiProduct = Readonly<{
  id: string;
  name: string;
  slug: string;
  description: string;
  image_filename: string | null;
}>;

export type CatalogListingApiCategory = Readonly<{
  name: string;
  products: readonly CatalogListingApiProduct[];
}>;

export type CatalogListingsApiResponse = Readonly<{
  categories: readonly CatalogListingApiCategory[];
}>;

export type CatalogPreviewProduct = Readonly<{
  id: string;
  name: string;
  slug: string;
  description: string;
  imageUrl: string;
}>;

export type CatalogPreviewCategory = Readonly<{
  name: string;
  products: readonly CatalogPreviewProduct[];
}>;

export type CatalogListingsStatus = "success" | "unavailable";

export type ContactRequestPayload = Readonly<{
  name: string;
  phone: string;
}>;

export type ContactRequestFieldErrors = Readonly<{
  name?: string;
  phone?: string;
}>;

export type ContactRequestSuccessResponse = Readonly<{
  ok: true;
  message: string;
}>;

export type ContactRequestValidationResponse = Readonly<{
  error: "validation_failed";
  fields: ContactRequestFieldErrors;
}>;

export type ContactRequestErrorCode =
  | "invalid_request"
  | "unsupported_media_type"
  | "request_too_large"
  | "contact_unavailable"
  | "backend_unavailable";

export type ContactRequestErrorResponse = Readonly<{
  error: ContactRequestErrorCode;
}>;

export type CatalogCategoryApiItem = Readonly<{
  id: string;
  name: string;
  product_count: number;
}>;

export type CatalogCategoriesApiResponse = Readonly<{
  categories: readonly CatalogCategoryApiItem[];
}>;

export type CatalogProductApiItem = Readonly<{
  id: string;
  name: string;
  slug: string;
  description: string;
  category_id: string;
  category_name: string;
  image_filename: string | null;
  available: boolean;
}>;

export type CatalogPaginationApi = Readonly<{
  page: number;
  page_size: number;
  total_items: number;
  total_pages: number;
  has_next: boolean;
  has_previous: boolean;
}>;

export type CatalogFiltersApi = Readonly<{
  query: string;
  category: string | null;
}>;

export type CatalogProductsApiResponse = Readonly<{
  items: readonly CatalogProductApiItem[];
  pagination: CatalogPaginationApi;
  filters: CatalogFiltersApi;
}>;

export type CatalogInvalidParametersApiFields = Readonly<{
  pagina?: string;
  por_pagina?: string;
}>;

export type CatalogInvalidParametersApiResponse = Readonly<{
  error: "invalid_parameters";
  fields: CatalogInvalidParametersApiFields;
}>;

export type CatalogBrowseCategory = Readonly<{
  id: string;
  name: string;
  productCount: number;
}>;

export type CatalogCategoriesResult =
  | Readonly<{ status: "success"; categories: readonly CatalogBrowseCategory[] }>
  | Readonly<{ status: "unavailable" }>;

export type CatalogBrowseProduct = Readonly<{
  id: string;
  name: string;
  slug: string;
  description: string;
  categoryId: string;
  categoryName: string;
  imageUrl: string;
  available: boolean;
}>;

export type CatalogBrowsePagination = Readonly<{
  page: number;
  pageSize: number;
  totalItems: number;
  totalPages: number;
  hasNext: boolean;
  hasPrevious: boolean;
}>;

export type CatalogBrowseFilters = Readonly<{
  query: string;
  category: string | null;
}>;

export type CatalogProductsRequest = Readonly<{
  query: string;
  category: string;
  page: number;
  pageSize: number;
}>;

export type CatalogProductsInvalidParametersFields = Readonly<{
  pagina?: string;
  por_pagina?: string;
}>;

export type CatalogProductsResult =
  | Readonly<{
      status: "success";
      items: readonly CatalogBrowseProduct[];
      pagination: CatalogBrowsePagination;
      filters: CatalogBrowseFilters;
    }>
  | Readonly<{
      status: "invalid_parameters";
      fields: CatalogProductsInvalidParametersFields;
    }>
  | Readonly<{ status: "unavailable" }>;

// --- Product detail: GET /api/catalog/products/{identifier} ---
// Shape derived exclusively from internal/routes/catalog_product_api.go
// (publicAPICatalogProductDetail / publicAPICatalogProductDetailResponse),
// not from any earlier conceptual example. quantity is intentionally
// absent: the Go endpoint does not return it.

export type CatalogProductDetailApiCategory = Readonly<{
  id: string;
  name: string;
}>;

export type CatalogProductDetailApiProduct = Readonly<{
  id: string;
  name: string;
  slug: string;
  description: string;
  long_description: string;
  category: CatalogProductDetailApiCategory | null;
  available: boolean;
  image_filename: string | null;
  images: readonly string[];
}>;

export type CatalogProductDetailApiResponse = Readonly<{
  product: CatalogProductDetailApiProduct;
}>;

export type CatalogProductDetailCategory = Readonly<{
  id: string;
  name: string;
}>;

export type CatalogProductDetail = Readonly<{
  id: string;
  name: string;
  slug: string;
  description: string;
  longDescription: string;
  category: CatalogProductDetailCategory | null;
  available: boolean;
  imageFilename: string | null;
  images: readonly string[];
}>;

export type CatalogProductDetailErrorCode =
  | "invalid_identifier"
  | "product_not_found"
  | "catalog_unavailable"
  | "backend_unavailable"
  | "invalid_response"
  | "unexpected_status";

export type CatalogProductDetailResult =
  | Readonly<{ status: "success"; product: CatalogProductDetail }>
  | Readonly<{ status: "error"; code: CatalogProductDetailErrorCode }>;

// --- Cart: GET/POST/PATCH/DELETE /api/cart[/items[/{product_id}]] ---
// Shape derived exclusively from internal/routes/cart_api.go
// (cartAPIItem/cartAPIState/cartAPIResponse) and
// internal/routes/cart_api_mutations.go (writeCartAPIMutationError's code
// set). Go remains the sole source of truth for stock/availability — these
// types only describe what it already returns.

export type CartApiItem = Readonly<{
  product_id: string;
  name: string;
  slug: string;
  image_filename: string;
  quantity: number;
  max_quantity: number;
  available: boolean;
}>;

export type CartApiState = Readonly<{
  items: readonly CartApiItem[];
  total_items: number;
}>;

export type CartApiResponse = Readonly<{
  cart: CartApiState;
}>;

export type CartItem = Readonly<{
  productId: string;
  name: string;
  slug: string;
  imageFilename: string;
  quantity: number;
  maxQuantity: number;
  available: boolean;
}>;

export type CartState = Readonly<{
  items: readonly CartItem[];
  totalItems: number;
}>;

// Exact error codes writeCartAPIMutationError (internal/routes/cart_api_mutations.go)
// can produce, plus this fetcher's own network/parsing failure codes.
export type CartErrorCode =
  | "invalid_request"
  | "request_too_large"
  | "unsupported_media_type"
  | "product_not_found"
  | "product_unavailable"
  | "insufficient_stock"
  | "cart_item_not_found"
  | "idempotency_key_required"
  | "invalid_idempotency_key"
  | "idempotency_conflict"
  | "cart_unavailable"
  | "backend_unavailable"
  | "invalid_response";

export type CartResult =
  | Readonly<{ status: "success"; cart: CartState; replayed: boolean }>
  | Readonly<{ status: "error"; code: CartErrorCode }>;

// Exact error codes internal/routes/contact_requests.go's JSON quote-request
// contract can produce (see quoteRequestErrorCode there), plus this
// fetcher's own network/parsing failure codes.
export type QuoteRequestErrorCode =
  | "invalid_request"
  | "request_too_large"
  | "unsupported_media_type"
  | "cart_empty"
  | "product_removed"
  | "product_unavailable"
  | "invalid_quantity"
  | "catalog_unavailable"
  | "db_error"
  | "idempotency_key_required"
  | "invalid_idempotency_key"
  | "idempotency_conflict"
  | "backend_unavailable"
  | "invalid_response";

export type QuoteRequestResult =
  | Readonly<{ status: "success"; replayed: boolean }>
  | Readonly<{ status: "error"; code: QuoteRequestErrorCode }>;
