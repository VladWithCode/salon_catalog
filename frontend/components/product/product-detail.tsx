import type { CatalogProductDetail } from "@/lib/types";

import { ProductActions } from "@/components/product/product-actions";
import { ProductAvailability } from "@/components/product/product-availability";
import { ProductBreadcrumbs } from "@/components/product/product-breadcrumbs";
import { ProductGallery } from "@/components/product/product-gallery";
import { ProductInformation } from "@/components/product/product-information";

type ProductDetailProps = Readonly<{
  product: CatalogProductDetail;
}>;

/**
 * Mobile order (source order, no reflow needed): breadcrumbs, gallery,
 * information, availability, actions. Desktop (lg:): two balanced
 * columns — gallery left, information/availability/actions right — via a
 * single grid, matching 06-product-page.md section N.
 */
export function ProductDetail({ product }: ProductDetailProps) {
  return (
    <article className="section-light py-section">
      <div className="container-main max-w-6xl space-y-section">
        <ProductBreadcrumbs productName={product.name} />

        <div className="grid gap-10 lg:grid-cols-2 lg:items-start">
          <ProductGallery
            imageFilename={product.imageFilename}
            images={product.images}
            productName={product.name}
          />

          <div className="space-y-8">
            <ProductInformation product={product} />
            <ProductAvailability available={product.available} />
            <ProductActions productId={product.id} available={product.available} />
          </div>
        </div>
      </div>
    </article>
  );
}
