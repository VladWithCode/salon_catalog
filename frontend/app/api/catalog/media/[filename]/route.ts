import { getGoAPIBaseURL } from "@/lib/env";

const backendTimeoutMilliseconds = 5_000;
const safeFilenamePattern = /^[\p{L}\p{N}._:-]+$/u;

type RouteContext = Readonly<{
  params: Promise<Readonly<{ filename: string }>>;
}>;

function errorResponse(status: number, error: string): Response {
  return Response.json(
    { error },
    {
      status,
      headers: {
        "Cache-Control": "no-store",
        "X-Content-Type-Options": "nosniff",
      },
    },
  );
}

function isSafeFilename(filename: string): boolean {
  return (
    filename.length > 0 &&
    filename !== "." &&
    filename !== ".." &&
    !filename.includes("/") &&
    !filename.includes("\\") &&
    !filename.includes("\0") &&
    safeFilenamePattern.test(filename)
  );
}

export async function GET(
  _request: Request,
  context: RouteContext,
): Promise<Response> {
  const { filename } = await context.params;
  if (!isSafeFilename(filename)) {
    return errorResponse(400, "invalid_filename");
  }

  try {
    const response = await fetch(
      `${getGoAPIBaseURL()}/static/uploads/${encodeURIComponent(filename)}`,
      {
        method: "GET",
        cache: "no-store",
        credentials: "omit",
        redirect: "error",
        headers: {
          Accept: "image/*",
        },
        signal: AbortSignal.timeout(backendTimeoutMilliseconds),
      },
    );

    if (response.status === 404) {
      return errorResponse(404, "image_not_found");
    }
    if (!response.ok) {
      return errorResponse(502, "backend_unavailable");
    }

    const contentType = response.headers.get("Content-Type");
    if (!contentType?.toLowerCase().startsWith("image/") || !response.body) {
      return errorResponse(502, "invalid_image_response");
    }

    return new Response(response.body, {
      status: httpStatusOK,
      headers: {
        "Cache-Control": "no-store",
        "Content-Type": contentType,
        "X-Content-Type-Options": "nosniff",
      },
    });
  } catch {
    return errorResponse(502, "backend_unavailable");
  }
}

const httpStatusOK = 200;
