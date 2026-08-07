"use server";

import { randomBytes } from "node:crypto";
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";

import { submitQuoteRequest } from "@/lib/api/quote";

function fieldValue(formData: FormData, key: string): string {
  const raw = formData.get(key);
  return typeof raw === "string" ? raw.trim() : "";
}

/**
 * Generated server-side, once per form render, exactly the same contract
 * as generateCartIdempotencyKey (frontend/lib/actions/cart-actions.ts): a
 * fresh key here means a fresh page load/navigation is a new intention;
 * the *same* rendered form retried (double click, resubmit before
 * navigation) reuses the hidden field's already-generated value, which Go
 * (internal/db.SubmitQuoteIdempotent) then treats as a replay of the same
 * intention rather than a new quote. Never generated inside the action
 * itself — that would let a retried submission mint a new key and defeat
 * the whole point.
 */
export async function generateQuoteIdempotencyKey(): Promise<string> {
  return randomBytes(18).toString("base64url");
}

/**
 * A real progressive <form action={submitQuoteRequestAction}> (see
 * frontend/app/(site)/solicitar-cotizacion/page.tsx) works with JavaScript
 * disabled the same way the cart's AddToCartForm does — Server Actions are
 * a native form submission, not a fetch call from client JS. Go remains
 * the sole source of truth for the cart and for creating the quote; this
 * action only forwards requester fields and follows PRG via redirect().
 */
export async function submitQuoteRequestAction(formData: FormData): Promise<void> {
  const name = fieldValue(formData, "name");
  const phone = fieldValue(formData, "phone");
  const email = fieldValue(formData, "email");
  const eventDate = fieldValue(formData, "event_date");
  const eventType = fieldValue(formData, "event_type");
  const idempotencyKey = fieldValue(formData, "idempotency_key");

  if (!idempotencyKey) {
    redirect("/solicitar-cotizacion?quote_error=invalid_idempotency_key");
  }

  const result = await submitQuoteRequest({ name, phone, email, eventDate, eventType, idempotencyKey });

  revalidatePath("/", "layout");
  if (result.status === "error") {
    redirect(`/solicitar-cotizacion?quote_error=${encodeURIComponent(result.code)}`);
  }
  redirect("/solicitar-cotizacion?quote_status=sent");
}
