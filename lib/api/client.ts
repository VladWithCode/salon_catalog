/**
 * Centralized fetch wrapper for the Go backend.
 *
 * - Reads the base URL from `NEXT_PUBLIC_API_BASE_URL` (default: localhost:8080).
 * - Throws a typed `ApiError` on non-2xx so callers can handle errors uniformly.
 * - All page-level fetchers (lib/api/*.ts) should use this; never `fetch` directly.
 */

export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export class ApiError extends Error {
  readonly status: number;
  readonly body: unknown;

  constructor(message: string, status: number, body: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

type ApiFetchOptions = Omit<RequestInit, "body"> & {
  /** JSON body — will be stringified and sets `Content-Type: application/json`. */
  json?: unknown;
  /** Optional query params, merged into the URL. */
  query?: Record<string, string | number | boolean | undefined | null>;
  /** When true, the call runs at build time / in a Server Component. */
  next?: { revalidate?: number | false; tags?: string[] };
};

function buildUrl(path: string, query?: ApiFetchOptions["query"]): string {
  const base = API_BASE_URL.replace(/\/$/, "");
  const cleanPath = path.startsWith("/") ? path : `/${path}`;
  const url = new URL(`${base}${cleanPath}`);
  if (query) {
    for (const [k, v] of Object.entries(query)) {
      if (v !== undefined && v !== null && v !== "") {
        url.searchParams.set(k, String(v));
      }
    }
  }
  return url.toString();
}

export async function apiFetch<T = unknown>(
  path: string,
  options: ApiFetchOptions = {},
): Promise<T> {
  const { json, query, headers, next, ...rest } = options;

  const init: RequestInit & { next?: ApiFetchOptions["next"] } = {
    ...rest,
    headers: {
      Accept: "application/json",
      ...(json !== undefined ? { "Content-Type": "application/json" } : {}),
      ...headers,
    },
    next,
  };

  if (json !== undefined) {
    init.body = JSON.stringify(json);
  }

  const res = await fetch(buildUrl(path, query), init);

  if (!res.ok) {
    let body: unknown = null;
    try {
      body = await res.json();
    } catch {
      try {
        body = await res.text();
      } catch {
        /* ignore */
      }
    }
    throw new ApiError(
      `API ${res.status} ${res.statusText} for ${path}`,
      res.status,
      body,
    );
  }

  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}
