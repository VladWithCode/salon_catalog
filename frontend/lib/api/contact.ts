import type {
  ContactRequestFieldErrors,
  ContactRequestPayload,
  ContactRequestSuccessResponse,
  ContactRequestValidationResponse,
} from "@/lib/types";

export type ContactSubmissionResult =
  | Readonly<{ status: "success" }>
  | Readonly<{
      status: "validation_error";
      fields: ContactRequestFieldErrors;
    }>
  | Readonly<{ status: "unavailable" }>;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isContactSuccessResponse(
  value: unknown,
): value is ContactRequestSuccessResponse {
  return (
    isRecord(value) &&
    value.ok === true &&
    typeof value.message === "string" &&
    value.message.length > 0
  );
}

function isContactFieldErrors(value: unknown): value is ContactRequestFieldErrors {
  if (!isRecord(value)) {
    return false;
  }

  const keys = Object.keys(value);
  return (
    keys.every((key) => key === "name" || key === "phone") &&
    (value.name === undefined || typeof value.name === "string") &&
    (value.phone === undefined || typeof value.phone === "string")
  );
}

function isContactValidationResponse(
  value: unknown,
): value is ContactRequestValidationResponse {
  return (
    isRecord(value) &&
    value.error === "validation_failed" &&
    isContactFieldErrors(value.fields)
  );
}

export async function submitContactRequest(
  payload: ContactRequestPayload,
  signal?: AbortSignal,
): Promise<ContactSubmissionResult> {
  try {
    const response = await fetch("/api/contact-requests", {
      method: "POST",
      cache: "no-store",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ name: payload.name, phone: payload.phone }),
      signal,
    });

    const body: unknown = await response.json();
    if (response.status === 201 && isContactSuccessResponse(body)) {
      return { status: "success" };
    }
    if (response.status === 400 && isContactValidationResponse(body)) {
      return { status: "validation_error", fields: body.fields };
    }
  } catch {
    // The form renders the same editorial state for transport and contract errors.
  }

  return { status: "unavailable" };
}
