import Image from "next/image";

import {
  removeCartItemAction,
  updateCartItemQuantityAction,
} from "@/lib/actions/cart-actions";
import type { CartItem } from "@/lib/types";

const fallbackImageUrl = "/assets/chenacolo_24.jpeg";
const safeImageFilenamePattern = /^[A-Za-z0-9._:-]+$/;

function mediaURL(filename: string): string {
  if (
    filename.length === 0 ||
    filename === "." ||
    filename === ".." ||
    !safeImageFilenamePattern.test(filename)
  ) {
    return fallbackImageUrl;
  }
  return `/api/catalog/media/${encodeURIComponent(filename)}`;
}

type CartItemRowProps = Readonly<{
  item: CartItem;
}>;

export function CartItemRow({ item }: CartItemRowProps) {
  return (
    <li className="flex gap-4 border-b border-border py-4 last:border-b-0">
      <div className="relative h-20 w-20 shrink-0 overflow-hidden rounded-lg bg-muted">
        <Image
          src={mediaURL(item.imageFilename)}
          alt={item.name}
          fill
          sizes="80px"
          className="object-cover"
        />
      </div>

      <div className="min-w-0 flex-1 space-y-2">
        <div>
          <p className="type-body font-medium">{item.name}</p>
          {!item.available ? (
            <p className="type-small text-destructive">Ya no está disponible.</p>
          ) : null}
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <form action={updateCartItemQuantityAction} className="flex items-center gap-2">
            <input type="hidden" name="product_id" value={item.productId} />
            <input type="hidden" name="return_to" value="/carrito" />
            <label htmlFor={`quantity-${item.productId}`} className="type-small text-muted-foreground">
              Cantidad
            </label>
            <input
              id={`quantity-${item.productId}`}
              type="number"
              name="quantity"
              min={1}
              max={Math.max(item.maxQuantity, 1)}
              defaultValue={item.quantity}
              className="min-h-11 w-16 rounded-md border border-border px-2 text-center"
            />
            <button
              type="submit"
              className="type-small min-h-11 rounded-md border border-border px-3 hover:border-accent focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
            >
              Actualizar
            </button>
          </form>

          <form action={removeCartItemAction}>
            <input type="hidden" name="product_id" value={item.productId} />
            <input type="hidden" name="return_to" value="/carrito" />
            <button
              type="submit"
              className="type-small min-h-11 rounded-md px-3 text-muted-foreground hover:text-destructive focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
            >
              Eliminar
            </button>
          </form>
        </div>
      </div>
    </li>
  );
}
