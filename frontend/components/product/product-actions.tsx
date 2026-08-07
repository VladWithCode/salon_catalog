import Link from "next/link";

import { AddToCartForm } from "@/components/cart/add-to-cart-form";

const actionClassName =
  "type-button inline-flex min-h-11 items-center rounded-lg px-5 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent";

type ProductActionsProps = Readonly<{
  productId: string;
  available: boolean;
}>;

/**
 * Cart integration unblocked (Fase 10): the real PostgreSQL suite passed
 * completely (see 09-final-completion.md / 10-production-cutover.md). Add
 * goes through the real Go-authoritative cart (frontend/lib/api/cart.ts,
 * server-to-server, same-origin from the browser's perspective). Quote
 * request stays a plain navigation to Go's own page — see
 * 10-production-cutover.md for why that specific flow (not the cart) is
 * still not reimplemented in Next.
 */
export function ProductActions({ productId, available }: ProductActionsProps) {
  return (
    <div className="space-y-4">
      <AddToCartForm productId={productId} available={available} returnTo="/carrito" />
      <div className="flex flex-wrap items-center gap-4">
        <Link
          href="/catalogo"
          className={`${actionClassName} border border-border text-foreground hover:border-accent`}
        >
          Volver al catálogo
        </Link>
        <a
          href="/solicitar-cotizacion"
          className={`${actionClassName} bg-accent text-accent-foreground hover:bg-accent/90`}
        >
          Solicitar cotización
        </a>
      </div>
    </div>
  );
}
