import Image from "next/image";
import Link from "next/link";

import { AddToCartForm } from "@/components/cart/add-to-cart-form";
import type { CatalogBrowseProduct } from "@/lib/types";

type CatalogProductCardProps = Readonly<{
  product: CatalogBrowseProduct;
}>;

export async function CatalogProductCard({ product }: CatalogProductCardProps) {
  const hasDescription = product.description.trim().length > 0;

  return (
    <article className="flex h-full min-w-0 flex-col overflow-hidden rounded-lg border border-border bg-card text-card-foreground shadow-card">
      <div className="relative aspect-square overflow-hidden bg-muted">
        <Image
          src={product.imageUrl}
          alt={`Imagen del producto ${product.name}`}
          fill
          sizes="(min-width: 1280px) 22vw, (min-width: 1024px) 30vw, (min-width: 768px) 45vw, calc(100vw - 3rem)"
          className="object-cover"
        />
      </div>

      <div className="flex flex-1 flex-col p-5">
        <p className="type-eyebrow">{product.categoryName}</p>
        <h3 className="type-h3 mt-2 font-medium">{product.name}</h3>

        <div className="mt-3 min-h-[3rem]">
          {hasDescription ? (
            <p className="type-small line-clamp-2 text-muted-foreground">
              {product.description}
            </p>
          ) : null}
        </div>

        <p className="type-small mt-4 font-medium text-muted-foreground">
          {product.available ? "Disponible" : "No disponible"}
        </p>

        <Link
          href={`/catalogo/producto/${encodeURIComponent(product.slug)}`}
          prefetch={false}
          aria-label={`Ver detalle de ${product.name}`}
          className="type-small mt-5 inline-flex min-h-11 items-center self-start font-medium text-accent-strong underline decoration-accent-strong/40 underline-offset-4 hover:decoration-accent-strong focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent"
        >
          Ver detalle
        </Link>

        <div className="mt-3">
          <AddToCartForm
            productId={product.id}
            available={product.available}
            returnTo="/catalogo"
            label="Añadir a selección"
          />
        </div>
      </div>
    </article>
  );
}
