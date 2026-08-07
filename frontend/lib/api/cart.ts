import "server-only";

import { cookies } from "next/headers";

import { getGoAPIBaseURL } from "@/lib/env";
import type {
  CartApiItem,
  CartApiResponse,
  CartErrorCode,
  CartItem,
  CartResult,
  CartState,
} from "@/lib/types";

const backendTimeoutMilliseconds = 5_000;

// Same host-only cookie name Go's session manager owns
// (internal/session/cart.go, CartCookieName). Next never decodes or signs
// it — it only forwards the opaque, already-signed value both directions.
const CART_COOKIE_NAME = "cart_id";

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isCartApiItem(value: unknown): value is CartApiItem {
  if (!isRecord(value)) return false;
  return (
    typeof value.product_id === "string" &&
    value.product_id.length > 0 &&
    typeof value.name === "string" &&
    typeof value.slug === "string" &&
    typeof value.image_filename === "string" &&
    typeof value.quantity === "number" &&
    typeof value.max_quantity === "number" &&
    typeof value.available === "boolean"
  );
}

function isCartApiResponse(value: unknown): value is CartApiResponse {
  if (!isRecord(value) || !isRecord(value.cart)) return false;
  const cart = value.cart;
  return (
    Array.isArray(cart.items) &&
    cart.items.every(isCartApiItem) &&
    typeof cart.total_items === "number"
  );
}

function toCartItem(item: CartApiItem): CartItem {
  return {
    productId: item.product_id,
    name: item.name,
    slug: item.slug,
    imageFilename: item.image_filename,
    quantity: item.quantity,
    maxQuantity: item.max_quantity,
    available: item.available,
  };
}

function toCartState(response: CartApiResponse): CartState {
  return {
    items: response.cart.items.map(toCartItem),
    totalItems: response.cart.total_items,
  };
}

function isJSONContentType(contentType: string | null): boolean {
  if (!contentType) return false;
  return contentType.toLowerCase().split(";")[0].trim() === "application/json";
}

/**
 * Every outgoing request to Go's cart API sets Origin to Go's own trusted
 * origin. This is not spoofing a browser: it is the server-to-server leg of
 * a same-origin design — the browser only ever talks to Next (this process),
 * and Next is the only caller of this function. Go's CSRFGuard
 * (internal/security/csrf.go) is satisfied the same way any first-party
 * caller satisfies it: by presenting the exact origin already listed in
 * CSRF_TRUSTED_ORIGINS. If GO_API_BASE_URL's origin is not itself in that
 * list, this call fails CSRF exactly as it should — that is a topology
 * misconfiguration to fix in environment variables, not something to work
 * around here.
 */
function goOrigin(): string {
  return new URL(getGoAPIBaseURL()).origin;
}

async function readCartCookieHeader(): Promise<string | undefined> {
  const store = await cookies();
  const cartCookie = store.get(CART_COOKIE_NAME);
  return cartCookie ? `${CART_COOKIE_NAME}=${cartCookie.value}` : undefined;
}

/**
 * Mirrors Go's own Set-Cookie for cart_id (internal/session/cart.go:
 * Path=/, HttpOnly, SameSite=Lax, Secure per CART_COOKIE_SECURE, MaxAge in
 * seconds) onto Next's own response cookie jar, without ever decoding or
 * re-signing the value — it is copied verbatim. Only applied when Go
 * actually sent a new cart_id (a fresh cart); an existing cart_id cookie
 * that Go merely read is never rewritten.
 */
async function applySetCookie(response: Response): Promise<void> {
  const setCookieValues = response.headers.getSetCookie?.() ?? [];
  const cartSetCookie = setCookieValues.find((value) =>
    value.startsWith(`${CART_COOKIE_NAME}=`),
  );
  if (!cartSetCookie) return;

  const [nameValue, ...attributePairs] = cartSetCookie.split(";").map((part) => part.trim());
  const value = nameValue.slice(CART_COOKIE_NAME.length + 1);

  let maxAge: number | undefined;
  let secure = false;
  for (const attribute of attributePairs) {
    const [rawKey, rawValue] = attribute.split("=");
    const key = rawKey.trim().toLowerCase();
    if (key === "max-age" && rawValue) {
      const parsed = Number.parseInt(rawValue, 10);
      if (Number.isFinite(parsed)) maxAge = parsed;
    }
    if (key === "secure") secure = true;
  }

  const store = await cookies();
  store.set(CART_COOKIE_NAME, value, {
    httpOnly: true,
    secure,
    sameSite: "lax",
    path: "/",
    maxAge,
  });
}

async function cartRequest(
  path: string,
  init: RequestInit,
  allowCookieWrite: boolean,
): Promise<CartResult> {
  const cookieHeader = await readCartCookieHeader();
  const headers = new Headers(init.headers);
  headers.set("Accept", "application/json");
  headers.set("Origin", goOrigin());
  if (cookieHeader) headers.set("Cookie", cookieHeader);

  let response: Response;
  try {
    response = await fetch(`${getGoAPIBaseURL()}${path}`, {
      ...init,
      headers,
      cache: "no-store",
      redirect: "error",
      signal: AbortSignal.timeout(backendTimeoutMilliseconds),
    });
  } catch {
    return { status: "error", code: "backend_unavailable" };
  }

  // Next only allows writing to the cookie jar from a Server Action or
  // Route Handler, never from a plain Server Component render (the layout
  // and /carrito's own GET both call fetchCartState() during render).
  // Mutations (add/patch/delete/clear) are only ever invoked from
  // frontend/lib/actions/cart-actions.ts, i.e. always inside a real Server
  // Action — allowCookieWrite is true exactly there.
  if (allowCookieWrite) {
    await applySetCookie(response);
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

  if (response.status === 200) {
    if (!isCartApiResponse(body)) {
      return { status: "error", code: "invalid_response" };
    }
    return {
      status: "success",
      cart: toCartState(body),
      replayed: response.headers.get("Idempotency-Replayed") === "true",
    };
  }

  if (isRecord(body) && isKnownCartErrorCode(body.error)) {
    return { status: "error", code: body.error };
  }
  return { status: "error", code: "invalid_response" };
}

const knownCartErrorCodes = new Set<CartErrorCode>([
  "invalid_request",
  "request_too_large",
  "unsupported_media_type",
  "product_not_found",
  "product_unavailable",
  "insufficient_stock",
  "cart_item_not_found",
  "idempotency_key_required",
  "invalid_idempotency_key",
  "idempotency_conflict",
  "cart_unavailable",
]);

function isKnownCartErrorCode(value: unknown): value is CartErrorCode {
  return typeof value === "string" && knownCartErrorCodes.has(value as CartErrorCode);
}

/**
 * Read-only — safe to call from a plain Server Component render (the site
 * layout and /carrito's own page both do). Never writes the cookie jar.
 */
export async function fetchCartState(): Promise<CartResult> {
  return cartRequest("/api/cart", { method: "GET" }, false);
}

/**
 * quantity is the amount to add on top of whatever is already in the cart
 * (Go's AddItemIdempotent semantics, not absolute). idempotencyKey must be
 * generated by the caller once per distinct user intention (see
 * frontend/lib/actions/cart-actions.ts) — this function never mints one
 * itself, so it can never accidentally reuse a key across two different
 * clicks.
 */
export async function addCartItem(
  productId: string,
  quantity: number,
  idempotencyKey: string,
): Promise<CartResult> {
  return cartRequest(
    "/api/cart/items",
    {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Idempotency-Key": idempotencyKey,
      },
      body: JSON.stringify({ product_id: productId, quantity }),
    },
    true,
  );
}

export async function setCartItemQuantity(
  productId: string,
  quantity: number,
): Promise<CartResult> {
  return cartRequest(
    `/api/cart/items/${encodeURIComponent(productId)}`,
    {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ quantity }),
    },
    true,
  );
}

export async function removeCartItem(productId: string): Promise<CartResult> {
  return cartRequest(
    `/api/cart/items/${encodeURIComponent(productId)}`,
    { method: "DELETE" },
    true,
  );
}

export async function clearCart(): Promise<CartResult> {
  return cartRequest("/api/cart", { method: "DELETE" }, true);
}
