import { afterEach, beforeEach, describe, expect, test } from "bun:test";

import { fetchCatalogProductDetail } from "@/lib/api/catalog-product";

const validProduct = {
  id: "01890f3a-dc02-7cb5-a4cc-451231879f0b",
  name: "Mesa Redonda",
  slug: "mesa-redonda",
  description: "Una mesa redonda",
  long_description: "",
  category: { id: "44444444-4444-4444-4444-444444444444", name: "Mobiliario" },
  available: true,
  image_filename: "mesa-redonda.jpg",
  images: ["mesa-redonda-1.jpg", "mesa-redonda-2.jpg"],
};

type FetchCall = Readonly<{ url: string; init: RequestInit | undefined }>;

let lastCall: FetchCall | undefined;
let originalFetch: typeof fetch;
let originalEnv: string | undefined;

function mockFetchOnce(response: Response | (() => Response)) {
  originalFetch = global.fetch;
  global.fetch = ((url: string, init?: RequestInit) => {
    lastCall = { url, init };
    const resolved = typeof response === "function" ? response() : response;
    return Promise.resolve(resolved);
  }) as typeof fetch;
}

function mockFetchThrow(error: unknown) {
  originalFetch = global.fetch;
  global.fetch = (() => Promise.reject(error)) as typeof fetch;
}

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

beforeEach(() => {
  originalEnv = process.env.GO_API_BASE_URL;
  process.env.GO_API_BASE_URL = "http://127.0.0.1:8080";
  lastCall = undefined;
});

afterEach(() => {
  global.fetch = originalFetch;
  if (originalEnv === undefined) {
    delete process.env.GO_API_BASE_URL;
  } else {
    process.env.GO_API_BASE_URL = originalEnv;
  }
});

describe("fetchCatalogProductDetail — identifier validation (no network)", () => {
  test("empty identifier is rejected before any fetch", async () => {
    const result = await fetchCatalogProductDetail("");
    expect(result).toEqual({ status: "error", code: "invalid_identifier" });
  });

  test("whitespace-only identifier is rejected", async () => {
    const result = await fetchCatalogProductDetail("   ");
    expect(result).toEqual({ status: "error", code: "invalid_identifier" });
  });

  test("identifier with a slash is rejected", async () => {
    const result = await fetchCatalogProductDetail("mesa/redonda");
    expect(result).toEqual({ status: "error", code: "invalid_identifier" });
  });

  test("identifier with a backslash is rejected", async () => {
    const result = await fetchCatalogProductDetail("mesa\\redonda");
    expect(result).toEqual({ status: "error", code: "invalid_identifier" });
  });

  test("identifier with a NUL byte is rejected", async () => {
    const result = await fetchCatalogProductDetail("mesa\0redonda");
    expect(result).toEqual({ status: "error", code: "invalid_identifier" });
  });

  test("200 Unicode characters is accepted (reaches fetch)", async () => {
    mockFetchOnce(jsonResponse(200, { product: validProduct }));
    const identifier = "é".repeat(200);
    const result = await fetchCatalogProductDetail(identifier);
    expect(result.status).toBe("success");
  });

  test("201 Unicode characters is rejected before fetch (counts characters, not UTF-16 units)", async () => {
    const identifier = "é".repeat(201);
    const result = await fetchCatalogProductDetail(identifier);
    expect(result).toEqual({ status: "error", code: "invalid_identifier" });
    expect(lastCall).toBeUndefined();
  });

  test("a valid identifier is percent-encoded exactly once into the request URL", async () => {
    mockFetchOnce(jsonResponse(200, { product: validProduct }));
    await fetchCatalogProductDetail("cojín-decorativo");
    expect(lastCall?.url).toBe(
      "http://127.0.0.1:8080/api/catalog/products/coj%C3%ADn-decorativo",
    );
  });

  test("UUID identifier passes validation and reaches fetch", async () => {
    mockFetchOnce(jsonResponse(200, { product: validProduct }));
    const result = await fetchCatalogProductDetail(validProduct.id);
    expect(result.status).toBe("success");
  });
});

describe("fetchCatalogProductDetail — success and validated fields", () => {
  test("minimal valid product returns a normalized success result", async () => {
    mockFetchOnce(jsonResponse(200, { product: validProduct }));
    const result = await fetchCatalogProductDetail("mesa-redonda");
    expect(result).toEqual({
      status: "success",
      product: {
        id: validProduct.id,
        name: "Mesa Redonda",
        slug: "mesa-redonda",
        description: "Una mesa redonda",
        longDescription: "",
        category: { id: validProduct.category.id, name: "Mobiliario" },
        available: true,
        imageFilename: "mesa-redonda.jpg",
        images: ["mesa-redonda-1.jpg", "mesa-redonda-2.jpg"],
      },
    });
  });

  test("category null is preserved as null, not coerced", async () => {
    mockFetchOnce(jsonResponse(200, { product: { ...validProduct, category: null } }));
    const result = await fetchCatalogProductDetail("mesa-redonda");
    expect(result.status).toBe("success");
    if (result.status === "success") {
      expect(result.product.category).toBeNull();
    }
  });

  test("available=false is preserved, not treated as an error", async () => {
    mockFetchOnce(jsonResponse(200, { product: { ...validProduct, available: false } }));
    const result = await fetchCatalogProductDetail("mesa-redonda");
    expect(result.status).toBe("success");
    if (result.status === "success") {
      expect(result.product.available).toBe(false);
    }
  });

  test("empty images array is preserved as an empty array", async () => {
    mockFetchOnce(jsonResponse(200, { product: { ...validProduct, images: [] } }));
    const result = await fetchCatalogProductDetail("mesa-redonda");
    expect(result.status).toBe("success");
    if (result.status === "success") {
      expect(result.product.images).toEqual([]);
    }
  });

  test("image order from the backend is preserved", async () => {
    mockFetchOnce(
      jsonResponse(200, { product: { ...validProduct, images: ["c.jpg", "a.jpg", "b.jpg"] } }),
    );
    const result = await fetchCatalogProductDetail("mesa-redonda");
    expect(result.status).toBe("success");
    if (result.status === "success") {
      expect(result.product.images).toEqual(["c.jpg", "a.jpg", "b.jpg"]);
    }
  });
});

describe("fetchCatalogProductDetail — HTTP error mapping", () => {
  test("400 maps to invalid_identifier", async () => {
    mockFetchOnce(jsonResponse(400, { error: "invalid_identifier" }));
    const result = await fetchCatalogProductDetail("mesa-redonda");
    expect(result).toEqual({ status: "error", code: "invalid_identifier" });
  });

  test("404 maps to product_not_found", async () => {
    mockFetchOnce(jsonResponse(404, { error: "product_not_found" }));
    const result = await fetchCatalogProductDetail("no-existe");
    expect(result).toEqual({ status: "error", code: "product_not_found" });
  });

  test("503 maps to catalog_unavailable", async () => {
    mockFetchOnce(jsonResponse(503, { error: "catalog_unavailable" }));
    const result = await fetchCatalogProductDetail("mesa-redonda");
    expect(result).toEqual({ status: "error", code: "catalog_unavailable" });
  });

  test("an unexpected status maps to unexpected_status", async () => {
    mockFetchOnce(jsonResponse(500, { error: "something_else" }));
    const result = await fetchCatalogProductDetail("mesa-redonda");
    expect(result).toEqual({ status: "error", code: "unexpected_status" });
  });
});

describe("fetchCatalogProductDetail — network and invalid response", () => {
  test("a rejected fetch (backend down) maps to backend_unavailable", async () => {
    mockFetchThrow(new TypeError("fetch failed"));
    const result = await fetchCatalogProductDetail("mesa-redonda");
    expect(result).toEqual({ status: "error", code: "backend_unavailable" });
  });

  test("non-JSON content type maps to invalid_response", async () => {
    mockFetchOnce(
      new Response("<html>not json</html>", {
        status: 200,
        headers: { "Content-Type": "text/html" },
      }),
    );
    const result = await fetchCatalogProductDetail("mesa-redonda");
    expect(result).toEqual({ status: "error", code: "invalid_response" });
  });

  test("root array instead of object maps to invalid_response", async () => {
    mockFetchOnce(jsonResponse(200, [validProduct]));
    const result = await fetchCatalogProductDetail("mesa-redonda");
    expect(result).toEqual({ status: "error", code: "invalid_response" });
  });

  test("missing product key maps to invalid_response", async () => {
    mockFetchOnce(jsonResponse(200, { notProduct: validProduct }));
    const result = await fetchCatalogProductDetail("mesa-redonda");
    expect(result).toEqual({ status: "error", code: "invalid_response" });
  });

  test("invalid UUID for id maps to invalid_response", async () => {
    mockFetchOnce(jsonResponse(200, { product: { ...validProduct, id: "not-a-uuid" } }));
    const result = await fetchCatalogProductDetail("mesa-redonda");
    expect(result).toEqual({ status: "error", code: "invalid_response" });
  });

  test("unsafe image_filename maps to invalid_response", async () => {
    mockFetchOnce(
      jsonResponse(200, { product: { ...validProduct, image_filename: "../../etc/passwd" } }),
    );
    const result = await fetchCatalogProductDetail("mesa-redonda");
    expect(result).toEqual({ status: "error", code: "invalid_response" });
  });

  test("images: null maps to invalid_response (never coerced to [])", async () => {
    mockFetchOnce(jsonResponse(200, { product: { ...validProduct, images: null } }));
    const result = await fetchCatalogProductDetail("mesa-redonda");
    expect(result).toEqual({ status: "error", code: "invalid_response" });
  });

  test("available as a non-boolean maps to invalid_response", async () => {
    mockFetchOnce(jsonResponse(200, { product: { ...validProduct, available: "yes" } }));
    const result = await fetchCatalogProductDetail("mesa-redonda");
    expect(result).toEqual({ status: "error", code: "invalid_response" });
  });
});

describe("fetchCatalogProductDetail — request shape", () => {
  test("uses cache: no-store and sends no credentials", async () => {
    mockFetchOnce(jsonResponse(200, { product: validProduct }));
    await fetchCatalogProductDetail("mesa-redonda");
    expect(lastCall?.init?.cache).toBe("no-store");
    expect(lastCall?.init?.credentials).toBe("omit");
  });

  test("issues exactly one fetch call", async () => {
    let calls = 0;
    mockFetchOnce(() => {
      calls += 1;
      return jsonResponse(200, { product: validProduct });
    });
    await fetchCatalogProductDetail("mesa-redonda");
    expect(calls).toBe(1);
  });
});
