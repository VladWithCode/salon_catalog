import type { Metadata } from "next";
import Link from "next/link";

import { CartItemRow } from "@/components/cart/cart-item-row";
import { clearCartAction } from "@/lib/actions/cart-actions";
import { fetchCartState } from "@/lib/api/cart";
import { cartErrorMessageFor, cartStatusMessageFor } from "@/lib/copy/cart";

export const metadata: Metadata = {
  title: "Tu selección",
  description: "Revisa los productos que has agregado desde el catálogo.",
};

type CartPageProps = Readonly<{
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}>;

function firstValue(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

export default async function CartPage({ searchParams }: CartPageProps) {
  const params = await searchParams;
  const statusMessage = cartStatusMessageFor(firstValue(params.cart_status));
  const errorMessage = cartErrorMessageFor(firstValue(params.cart_error));

  const result = await fetchCartState();

  return (
    <article className="section-light py-section">
      <div className="container-main max-w-3xl space-y-section">
        <h1 className="type-h1 font-medium">Tu selección</h1>

        {statusMessage ? (
          <div
            role="status"
            className="rounded-lg border border-border bg-card px-4 py-3 type-small"
          >
            {statusMessage}
          </div>
        ) : null}
        {errorMessage ? (
          <div
            role="alert"
            className="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 type-small text-destructive"
          >
            {errorMessage}
          </div>
        ) : null}

        {result.status === "error" ? (
          <div role="alert" className="type-body text-muted-foreground">
            El carrito no está disponible en este momento. Intenta de nuevo más tarde.
          </div>
        ) : result.cart.items.length === 0 ? (
          <p className="type-body text-muted-foreground">
            Tu selección está vacía.{" "}
            <Link href="/catalogo" className="text-accent underline underline-offset-4">
              Ver catálogo
            </Link>
          </p>
        ) : (
          <>
            <ul>
              {result.cart.items.map((item) => (
                <CartItemRow key={item.productId} item={item} />
              ))}
            </ul>

            <div className="flex flex-wrap items-center gap-4">
              <Link
                href="/solicitar-cotizacion"
                className="type-button inline-flex min-h-11 items-center rounded-lg bg-accent px-5 text-accent-foreground hover:bg-accent/90 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent"
              >
                Solicitar cotización
              </Link>
              <form action={clearCartAction}>
                <input type="hidden" name="return_to" value="/carrito" />
                <button
                  type="submit"
                  className="type-button min-h-11 px-4 text-muted-foreground hover:text-destructive focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
                >
                  Vaciar selección
                </button>
              </form>
            </div>
          </>
        )}
      </div>
    </article>
  );
}
