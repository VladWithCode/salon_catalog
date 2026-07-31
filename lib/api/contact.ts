import { apiFetch } from "./client";
import type { ContactRequest, ContactRequestResponse } from "@/lib/types";

/**
 * Submits the home-page contact form to the Go backend.
 * Returns the parsed response (or throws ApiError on non-2xx).
 *
 * Called from a client component, so no `server-only` import here.
 */
export async function postContactRequest(
  payload: ContactRequest,
): Promise<ContactRequestResponse> {
  return apiFetch<ContactRequestResponse>("/api/contact-requests", {
    method: "POST",
    json: payload,
  });
}
