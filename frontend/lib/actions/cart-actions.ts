"use server";

import { randomBytes } from "node:crypto";
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";

import {
  addCartItem,
  clearCart,
  removeCartItem,
  setCartItemQuantity,
} from "@/lib/api/cart";

// return_to allowlist mirrors internal/routes/cart_forms.go's
// sanitizeCartReturnTo exactly: only these path families, no open redirect,
// invalid/missing always falls back to /carrito without ever reflecting the
// client-supplied value.
const cartReturnToAllowedPaths = ["/carrito", "/catalogo", "/solicitar-cotizacion"];
const cartFallbackReturnTo = "/carrito";

function sanitizeReturnTo(raw: FormDataEntryValue | null): string {
  if (typeof raw !== "string" || raw === "") return cartFallbackReturnTo;
  if (!raw.startsWith("/") || raw.startsWith("//")) return cartFallbackReturnTo;
  if (raw.includes("\\") || raw.includes("#")) return cartFallbackReturnTo;
  for (const char of raw) {
    const codePoint = char.codePointAt(0) ?? 0;
    if (codePoint < 0x20 || codePoint === 0x7f) return cartFallbackReturnTo;
  }
  let parsed: URL;
  try {
    parsed = new URL(raw, "http://internal.invalid");
  } catch {
    return cartFallbackReturnTo;
  }
  if (parsed.username || parsed.password || parsed.hash) return cartFallbackReturnTo;
  const matches = cartReturnToAllowedPaths.some(
    (allowed) => parsed.pathname === allowed || parsed.pathname.startsWith(`${allowed}/`),
  );
  return matches ? raw : cartFallbackReturnTo;
}

function buildRedirectLocation(
  returnTo: string,
  statusCode: string | null,
  errorCode: string | null,
): string {
  let parsed: URL;
  try {
    parsed = new URL(returnTo, "http://internal.invalid");
  } catch {
    parsed = new URL(cartFallbackReturnTo, "http://internal.invalid");
  }
  parsed.searchParams.delete("cart_status");
  parsed.searchParams.delete("cart_error");
  if (statusCode) parsed.searchParams.set("cart_status", statusCode);
  if (errorCode) parsed.searchParams.set("cart_error", errorCode);
  return `${parsed.pathname}${parsed.search}`;
}

function afterMutation(returnTo: string, statusCode: string | null, errorCode: string | null): never {
  // Every page that can show cart state (header count, /carrito, product
  // detail's availability) is a Server Component reading fresh data on
  // every request — revalidating the layout is what makes the header
  // count reflect a mutation without any client-side cart store.
  revalidatePath("/", "layout");
  redirect(buildRedirectLocation(returnTo, statusCode, errorCode));
}

function normalizeProductId(raw: FormDataEntryValue | null): string | null {
  if (typeof raw !== "string") return null;
  const uuidPattern = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;
  return uuidPattern.test(raw) ? raw : null;
}

function parsePositiveQuantity(raw: FormDataEntryValue | null): number | null {
  if (typeof raw !== "string") return null;
  const quantity = Number.parseInt(raw.trim(), 10);
  return Number.isInteger(quantity) && quantity >= 1 ? quantity : null;
}

/**
 * Generates a fresh Idempotency-Key server-side, exactly once per form
 * render, the same way internal/templates/components/catalog.templ's
 * cart.NewIdempotencyKey() call works for Go's own progressive forms — see
 * frontend/components/cart/add-to-cart-form.tsx, the only caller that
 * embeds this into a hidden field. It is never generated inside this
 * action (that would let a retried submission mint a new key and defeat
 * idempotency entirely — the same mistake 5B7B6A fixed on the Go side for
 * PUT /carrito).
 */
export async function generateCartIdempotencyKey(): Promise<string> {
  return randomBytes(18).toString("base64url");
}

export async function addToCartAction(formData: FormData): Promise<void> {
  const returnTo = sanitizeReturnTo(formData.get("return_to"));
  const productId = normalizeProductId(formData.get("product_id"));
  const quantity = parsePositiveQuantity(formData.get("quantity"));
  const idempotencyKey = formData.get("idempotency_key");

  if (!productId || !quantity || typeof idempotencyKey !== "string" || idempotencyKey === "") {
    afterMutation(returnTo, null, "invalid_request");
  }

  const result = await addCartItem(productId, quantity, idempotencyKey);
  if (result.status === "error") {
    afterMutation(returnTo, null, result.code);
  }
  afterMutation(returnTo, "added", null);
}

export async function updateCartItemQuantityAction(formData: FormData): Promise<void> {
  const returnTo = sanitizeReturnTo(formData.get("return_to"));
  const productId = normalizeProductId(formData.get("product_id"));
  const quantity = parsePositiveQuantity(formData.get("quantity"));

  if (!productId || !quantity) {
    afterMutation(returnTo, null, "invalid_request");
  }

  const result = await setCartItemQuantity(productId, quantity);
  if (result.status === "error") {
    afterMutation(returnTo, null, result.code);
  }
  afterMutation(returnTo, "updated", null);
}

export async function removeCartItemAction(formData: FormData): Promise<void> {
  const returnTo = sanitizeReturnTo(formData.get("return_to"));
  const productId = normalizeProductId(formData.get("product_id"));

  if (!productId) {
    afterMutation(returnTo, null, "invalid_request");
  }

  const result = await removeCartItem(productId);
  if (result.status === "error") {
    afterMutation(returnTo, null, result.code);
  }
  afterMutation(returnTo, "removed", null);
}

export async function clearCartAction(formData: FormData): Promise<void> {
  const returnTo = sanitizeReturnTo(formData.get("return_to"));

  const result = await clearCart();
  if (result.status === "error") {
    afterMutation(returnTo, null, result.code);
  }
  afterMutation(returnTo, "cleared", null);
}
