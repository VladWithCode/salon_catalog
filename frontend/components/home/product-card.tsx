import Image from "next/image";
import Link from "next/link";

import type { CatalogPreviewProduct } from "@/lib/types";

type ProductCardProps = Readonly<{
  product: CatalogPreviewProduct;
}>;

export function ProductCard({ product }: ProductCardProps) {
  return (
    <Link
      href={`/catalogo/producto/${encodeURIComponent(product.slug)}`}
      prefetch={false}
      className="group block overflow-hidden rounded-lg bg-card text-card-foreground shadow-card transition-shadow hover:shadow-elevated focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent"
    >
      <div className="relative aspect-square overflow-hidden bg-muted">
        <Image
          src={product.imageUrl}
          alt={`Imagen del producto ${product.name}`}
          fill
          sizes="(min-width: 1024px) 20vw, (min-width: 640px) 45vw, 90vw"
          className="object-cover transition-transform duration-500 ease-elegant group-hover:scale-105 group-focus-visible:scale-105"
        />
      </div>
      <div className="min-h-24 p-4">
        <h4 className="type-small line-clamp-2 font-semibold">{product.name}</h4>
        {product.description ? (
          <p className="mt-2 line-clamp-2 text-sm text-muted-foreground">
            {product.description}
          </p>
        ) : null}
      </div>
    </Link>
  );
}
