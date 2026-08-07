import { NextResponse } from "next/server";

import { fetchGoHealth } from "@/lib/api/server";

export const dynamic = "force-dynamic";

const unavailableResponse = {
  status: "error",
  service: "go",
  message: "Backend unavailable",
} as const;

export async function GET() {
  try {
    const response = await fetchGoHealth();

    return NextResponse.json(response.body, {
      status: response.status,
      headers: {
        "Cache-Control": "no-store",
      },
    });
  } catch {
    return NextResponse.json(unavailableResponse, {
      status: 502,
      headers: {
        "Cache-Control": "no-store",
      },
    });
  }
}
