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

export async function POST(request: Request): Promise<Response> {
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
      body: JSON.stringify({ name: payload.name, phone: payload.phone }),
      signal: AbortSignal.timeout(backendTimeoutMilliseconds),
    });

    const body: unknown = await response.json();
    if (response.status === httpStatusCreated && isContactSuccessResponse(body)) {
      return jsonResponse(body, response.status);
    }
    if (
      response.status === httpStatusBadRequest &&
      isContactValidationResponse(body)
    ) {
      return jsonResponse(body, response.status);
    }

    const expectedError = knownErrorStatuses.get(response.status);
    if (expectedError && isContactErrorResponse(body, expectedError)) {
      return jsonResponse(body, response.status);
    }
  } catch {
    // Return the same controlled response for network, timeout and parsing failures.
  }

  return jsonResponse(
    { error: "backend_unavailable" } satisfies ContactRequestErrorResponse,
    httpStatusBadGateway,
  );
}
