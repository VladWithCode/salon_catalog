import { getGoAPIBaseURL } from "@/lib/env";
import type {
  ContactRequestErrorCode,
  ContactRequestErrorResponse,
  ContactRequestFieldErrors,
  ContactRequestPayload,
  ContactRequestSuccessResponse,
  ContactRequestValidationResponse,
} from "@/lib/types";

const backendTimeoutMilliseconds = 5_000;
const maxRequestBodyBytes = 8 * 1024;
const httpStatusCreated = 201;
const httpStatusBadRequest = 400;
const httpStatusRequestTooLarge = 413;
const httpStatusUnsupportedMediaType = 415;
const httpStatusInternalServerError = 500;
const httpStatusBadGateway = 502;
const knownErrorStatuses = new Map<number, ContactRequestErrorCode>([
  [httpStatusBadRequest, "invalid_request"],
  [httpStatusRequestTooLarge, "request_too_large"],
  [httpStatusUnsupportedMediaType, "unsupported_media_type"],
  [httpStatusInternalServerError, "contact_unavailable"],
]);

function jsonResponse(value: unknown, status: number): Response {
  return Response.json(value, {
    status,
    headers: {
      "Cache-Control": "no-store",
      "X-Content-Type-Options": "nosniff",
    },
  });
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isExactContactPayload(value: unknown): value is ContactRequestPayload {
  if (!isRecord(value)) {
    return false;
  }

  const keys = Object.keys(value);
  return (
    keys.length === 2 &&
    keys.includes("name") &&
    keys.includes("phone") &&
    typeof value.name === "string" &&
    typeof value.phone === "string"
  );
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

function isContactErrorResponse(
  value: unknown,
  expectedError: ContactRequestErrorCode,
): value is ContactRequestErrorResponse {
  return isRecord(value) && value.error === expectedError;
}

async function readLimitedRequestBody(request: Request): Promise<Uint8Array | null> {
  const contentLength = request.headers.get("Content-Length");
  if (contentLength !== null) {
    const parsedLength = Number(contentLength);
    if (Number.isFinite(parsedLength) && parsedLength > maxRequestBodyBytes) {
      return null;
    }
  }

  if (request.body === null) {
    return new Uint8Array();
  }

  const reader = request.body.getReader();
  const chunks: Uint8Array[] = [];
  let totalBytes = 0;

  while (true) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }
    totalBytes += value.byteLength;
    if (totalBytes > maxRequestBodyBytes) {
      await reader.cancel();
      return null;
    }
    chunks.push(value);
  }

  const body = new Uint8Array(totalBytes);
  let offset = 0;
  for (const chunk of chunks) {
    body.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return body;
}

function hasJSONContentType(request: Request): boolean {
  const contentType = request.headers.get("Content-Type");
  return contentType?.split(";", 1)[0]?.trim().toLowerCase() === "application/json";
}

function hasFormContentType(request: Request): boolean {
  const contentType = request.headers.get("Content-Type");
  return (
    contentType?.split(";", 1)[0]?.trim().toLowerCase() ===
    "application/x-www-form-urlencoded"
  );
}

/**
 * Post/Redirect/Get for the no-JavaScript path. `<form action method="post">`
 * without JS always sends application/x-www-form-urlencoded and follows the
 * response as a navigation, so a JSON body would render as raw text in the
 * browser. Returning 303 to the contact anchor with a status code in the
 * query string gives that visitor a real page, and a refresh never
 * resubmits — the same PRG contract the cart and quote flows already use
 * (frontend/lib/actions/cart-actions.ts).
 */
function contactRedirect(request: Request, status: string): Response {
  const target = new URL("/", request.url);
  target.searchParams.set("contacto", status);
  target.hash = "contacto";
  return new Response(null, {
    status: 303,
    headers: { Location: `${target.pathname}${target.search}${target.hash}`, "Cache-Control": "no-store" },
  });
}

async function readFormSubmission(
  request: Request,
): Promise<{ name: string; phone: string } | null> {
  const encodedBody = await readLimitedRequestBody(request);
  if (encodedBody === null) return null;
  let decoded: string;
  try {
    decoded = new TextDecoder("utf-8", { fatal: true }).decode(encodedBody);
  } catch {
    return null;
  }
  const fields = new URLSearchParams(decoded);
  const name = fields.get("name");
  const phone = fields.get("phone");
  if (typeof name !== "string" || typeof phone !== "string") return null;
  return { name, phone };
}

export async function POST(request: Request): Promise<Response> {
  if (hasFormContentType(request)) {
    const submission = await readFormSubmission(request);
    if (submission === null) {
      return contactRedirect(request, "error");
    }
    const outcome = await forwardContactRequest(submission);
    return contactRedirect(request, outcome.status === httpStatusCreated ? "enviado" : "error");
  }

  if (!hasJSONContentType(request)) {
    return jsonResponse(
      { error: "unsupported_media_type" } satisfies ContactRequestErrorResponse,
      httpStatusUnsupportedMediaType,
    );
  }

  const encodedBody = await readLimitedRequestBody(request);
  if (encodedBody === null) {
    return jsonResponse(
      { error: "request_too_large" } satisfies ContactRequestErrorResponse,
      httpStatusRequestTooLarge,
    );
  }

  let payload: unknown;
  try {
    payload = JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(encodedBody));
  } catch {
    return jsonResponse(
      { error: "invalid_request" } satisfies ContactRequestErrorResponse,
      httpStatusBadRequest,
    );
  }

  if (!isExactContactPayload(payload)) {
    return jsonResponse(
      { error: "invalid_request" } satisfies ContactRequestErrorResponse,
      httpStatusBadRequest,
    );
  }

  const outcome = await forwardContactRequest({
    name: payload.name,
    phone: payload.phone,
  });
  return jsonResponse(outcome.body, outcome.status);
}

/**
 * Single place where a validated submission reaches Go, shared by the JSON
 * path (client component) and the urlencoded PRG path (no JavaScript), so
 * neither can drift from the other. Go stays the only validator of record —
 * this never decides on its own whether a request is acceptable.
 */
async function forwardContactRequest(submission: {
  name: string;
  phone: string;
}): Promise<{ status: number; body: unknown }> {
  try {
    const response = await fetch(`${getGoAPIBaseURL()}/api/contact-requests`, {
      method: "POST",
      cache: "no-store",
      credentials: "omit",
      redirect: "error",
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ name: submission.name, phone: submission.phone }),
      signal: AbortSignal.timeout(backendTimeoutMilliseconds),
    });

    const body: unknown = await response.json();
    if (response.status === httpStatusCreated && isContactSuccessResponse(body)) {
      return { status: response.status, body };
    }
    if (
      response.status === httpStatusBadRequest &&
      isContactValidationResponse(body)
    ) {
      return { status: response.status, body };
    }

    const expectedError = knownErrorStatuses.get(response.status);
    if (expectedError && isContactErrorResponse(body, expectedError)) {
      return { status: response.status, body };
    }
  } catch {
    // Return the same controlled response for network, timeout and parsing failures.
  }

  return {
    status: httpStatusBadGateway,
    body: { error: "backend_unavailable" } satisfies ContactRequestErrorResponse,
  };
}
