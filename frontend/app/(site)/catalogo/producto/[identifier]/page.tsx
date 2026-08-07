import type { Metadata } from "next";
import { notFound } from "next/navigation";

import { ProductDetail } from "@/components/product/product-detail";
import { ProductErrorState } from "@/components/product/product-error-state";
import { fetchCatalogProductDetail } from "@/lib/api/catalog-product";

type ProductPageProps = Readonly<{
  params: Promise<{ identifier: string }>;
}>;

// fetchCatalogProductDetail(identifier) is called once here and once more
// below; Next's fetch request memoization deduplicates identical
// fetch(url, options) calls within one render pass automatically (it does
// not touch the underlying no-store policy in
// frontend/lib/api/catalog-product.ts — that stays as-is), so this is not
// a second network round trip in practice.
export async function generateMetadata({
  params,
}: ProductPageProps): Promise<Metadata> {
  const { identifier } = await params;
  const result = await fetchCatalogProductDetail(identifier);

  if (result.status !== "success") {
    return { title: "Producto" };
  }

  const { product } = result;
  return {
    title: product.name,
    description:
      product.description.trim().length > 0 ? product.description : undefined,
  };
}

export default async function ProductDetailPage({ params }: ProductPageProps) {
  const { identifier } = await params;
  const result = await fetchCatalogProductDetail(identifier);

  if (result.status === "error") {
    if (result.code === "invalid_identifier" || result.code === "product_not_found") {
      notFound();
    }
    return <ProductErrorState />;
  }

  return <ProductDetail product={result.product} />;
}
