import { getGoAPIBaseURL } from "@/lib/env";

const backendTimeoutMilliseconds = 5_000;

export type GoHealthResponse = {
  status: "ok";
  service: "go";
};

type GoHealthResult = {
  body: GoHealthResponse;
  status: number;
};

function isGoHealthResponse(value: unknown): value is GoHealthResponse {
  if (!value || typeof value !== "object") {
    return false;
  }

  const response = value as Record<string, unknown>;
  return response.status === "ok" && response.service === "go";
}

export async function fetchGoHealth(): Promise<GoHealthResult> {
  const response = await fetch(`${getGoAPIBaseURL()}/api/_health`, {
    method: "GET",
    cache: "no-store",
    credentials: "omit",
    headers: {
      Accept: "application/json",
    },
    signal: AbortSignal.timeout(backendTimeoutMilliseconds),
  });

  const body: unknown = await response.json();
  if (!isGoHealthResponse(body)) {
    throw new Error("Invalid backend health response");
  }

  return {
    body,
    status: response.status,
  };
}
