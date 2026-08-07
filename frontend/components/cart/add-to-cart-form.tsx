import { addToCartAction, generateCartIdempotencyKey } from "@/lib/actions/cart-actions";

type AddToCartFormProps = Readonly<{
  productId: string;
  available: boolean;
  returnTo: string;
  label?: string;
  className?: string;
}>;

/**
 * Server Component: a real <form> whose action is a Server Action
 * (addToCartAction). This is what makes it work without JavaScript — Next
 * always processes a Server Action form submission server-side, JS or not,
 * the same structural guarantee Go's own progressive forms
 * (internal/templates/components/catalog.templ) rely on. The
 * Idempotency-Key is generated once per render here, matching
 * cart.NewIdempotencyKey()'s per-render contract on the Go side — never
 * regenerated client-side, never reused across a different product or
 * quantity.
 */
export async function AddToCartForm({
  productId,
  available,
  returnTo,
  label = "Añadir a selección",
  className,
}: AddToCartFormProps) {
  const idempotencyKey = await generateCartIdempotencyKey();

  return (
    <form action={addToCartAction}>
      <input type="hidden" name="product_id" value={productId} />
      <input type="hidden" name="quantity" value="1" />
      <input type="hidden" name="idempotency_key" value={idempotencyKey} />
      <input type="hidden" name="return_to" value={returnTo} />
      <button
        type="submit"
        disabled={!available}
        className={
          className ??
          "type-button inline-flex min-h-11 w-full items-center justify-center rounded-lg bg-accent px-5 text-accent-foreground hover:bg-accent/90 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent disabled:cursor-not-allowed disabled:bg-muted disabled:text-muted-foreground"
        }
      >
        {available ? label : "No disponible"}
      </button>
    </form>
  );
}
