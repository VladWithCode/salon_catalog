import "server-only";

import { cookies } from "next/headers";

import { getGoAPIBaseURL } from "@/lib/env";
import type { QuoteRequestErrorCode, QuoteRequestResult } from "@/lib/types";

const backendTimeoutMilliseconds = 5_000;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isJSONContentType(contentType: string | null): boolean {
  if (!contentType) return false;
  return contentType.toLowerCase().split(";")[0].trim() === "application/json";
}

const knownQuoteRequestErrorCodes = new Set<QuoteRequestErrorCode>([
  "invalid_request",
  "request_too_large",
  "unsupported_media_type",
  "cart_empty",
  "product_removed",
  "product_unavailable",
  "invalid_quantity",
  "catalog_unavailable",
  "db_error",
  "idempotency_key_required",
  "invalid_idempotency_key",
  "idempotency_conflict",
]);

function isKnownQuoteRequestErrorCode(value: unknown): value is QuoteRequestErrorCode {
  return typeof value === "string" && knownQuoteRequestErrorCodes.has(value as QuoteRequestErrorCode);
}

/**
 * Same server-to-server, same-origin, no-CORS design as
 * frontend/lib/api/cart.ts's goOrigin()/cartRequest(): the browser only
 * ever talks to Next; this function (only ever called from
 * frontend/lib/actions/quote-actions.ts, a real Server Action) is the one
 * making the trusted call to Go, presenting Go's own configured origin and
 * forwarding the signed cart cookie so Go resolves the same cart identity
 * it would for any other cart request. cart_id, products, quantities are
 * never sent in the body — Go reloads all of that itself
 * (internal/routes/contact_requests.go, validateQuoteCart).
 */
export async function submitQuoteRequest(fields: {
  name: string;
  phone: string;
  email?: string;
  eventDate?: string;
  eventType?: string;
  idempotencyKey: string;
}): Promise<QuoteRequestResult> {
  const store = await cookies();
  const cartCookie = store.get("cart_id");
  const headers = new Headers({
    "Content-Type": "application/json",
    Accept: "application/json",
    Origin: new URL(getGoAPIBaseURL()).origin,
    "Idempotency-Key": fields.idempotencyKey,
  });
  if (cartCookie) headers.set("Cookie", `cart_id=${cartCookie.value}`);

  let response: Response;
  try {
    response = await fetch(`${getGoAPIBaseURL()}/solicitar-cotizacion`, {
      method: "POST",
      headers,
      body: JSON.stringify({
        name: fields.name,
        phone: fields.phone,
        email: fields.email ?? "",
        event_date: fields.eventDate ?? "",
        event_type: fields.eventType ?? "",
      }),
      cache: "no-store",
      redirect: "error",
      signal: AbortSignal.timeout(backendTimeoutMilliseconds),
    });
  } catch {
    return { status: "error", code: "backend_unavailable" };
  }

  // Go's cart-session middleware may mint a fresh cart_id cookie on this
  // request (the same first-visit case fetchCartState's read path can hit)
  // — mirror it exactly like cartRequest's applySetCookie does, since this
  // call always runs inside a real Server Action and is therefore allowed
  // to write the cookie jar.
  const setCookieValues = response.headers.getSetCookie?.() ?? [];
  const cartSetCookie = setCookieValues.find((value) => value.startsWith("cart_id="));
  if (cartSetCookie) {
    const [nameValue, ...attributePairs] = cartSetCookie.split(";").map((part) => part.trim());
    const value = nameValue.slice("cart_id=".length);
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
    store.set("cart_id", value, { httpOnly: true, secure, sameSite: "lax", path: "/", maxAge });
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

  if (response.status === 200 && isRecord(body) && body.status === "success") {
    return { status: "success", replayed: body.replayed === true };
  }
  if (isRecord(body) && isKnownQuoteRequestErrorCode(body.error)) {
    return { status: "error", code: body.error };
  }
  return { status: "error", code: "invalid_response" };
}
