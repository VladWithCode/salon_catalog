import type { Metadata } from "next";
import Link from "next/link";

import { generateQuoteIdempotencyKey, submitQuoteRequestAction } from "@/lib/actions/quote-actions";
import { fetchCartState } from "@/lib/api/cart";
import { contact } from "@/lib/copy/contact";

export const metadata: Metadata = {
  title: "Solicitar Cotización",
  description: "Solicita una cotización para tu evento en Villa Chenacolo.",
};

type QuoteRequestPageProps = Readonly<{
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}>;

function firstValue(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

// Mirrors internal/routes/contact_requests.go's quoteRequestErrorCode set
// exactly, so a Go-side error always has Next-side copy for it.
function quoteErrorMessageFor(code: string | undefined): string | undefined {
  switch (code) {
    case "cart_empty":
      return "Tu selección está vacía. Agrega productos desde el catálogo antes de solicitar cotización.";
    case "product_removed":
      return "Uno de los productos de tu selección ya no está disponible en el catálogo.";
    case "product_unavailable":
      return "Uno de los productos de tu selección ya no está disponible.";
    case "invalid_quantity":
      return "La cantidad de un producto en tu selección ya no es válida.";
    case "invalid_request":
      return "Revisa los datos del formulario: nombre y teléfono son obligatorios.";
    case "idempotency_conflict":
      return "Ya hay una solicitud distinta en proceso con esta misma acción. Recarga la página e intenta de nuevo.";
    case "idempotency_key_required":
    case "invalid_idempotency_key":
    case "catalog_unavailable":
    case "db_error":
    case "backend_unavailable":
    case "invalid_response":
      return "No se pudo procesar tu solicitud en este momento. Intenta de nuevo más tarde.";
    default:
      return undefined;
  }
}

export default async function QuoteRequestPage({ searchParams }: QuoteRequestPageProps) {
  const params = await searchParams;
  const errorMessage = quoteErrorMessageFor(firstValue(params.quote_error));
  const wasSent = firstValue(params.quote_status) === "sent";

  const cartResult = await fetchCartState();
  const backendUnavailable = cartResult.status === "error";
  const cartItems = cartResult.status === "success" ? cartResult.cart.items : [];
  const cartEmpty = !backendUnavailable && cartItems.length === 0;
  const idempotencyKey = await generateQuoteIdempotencyKey();

  return (
    <article className="section-light py-section">
      <div className="container-main max-w-2xl space-y-8">
        <div className="space-y-4 text-center">
          <h1 className="type-h1 font-medium">Solicitar Cotización</h1>
          <p className="type-lead text-muted-foreground">
            Cuéntanos sobre tu evento y nos pondremos en contacto contigo.
          </p>
        </div>

        {wasSent ? (
          <div
            role="status"
            className="rounded-lg border border-border bg-card px-4 py-4 text-center type-body"
          >
            Solicitud enviada con éxito. Nos pondremos en contacto contigo pronto.
            <div className="mt-3">
              <Link href="/carrito" className="text-accent underline underline-offset-4">
                Ver tu selección
              </Link>
            </div>
          </div>
        ) : (
          <>
            {errorMessage ? (
              <div
                role="alert"
                className="rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 type-small text-destructive"
              >
                {errorMessage}
              </div>
            ) : null}

            {backendUnavailable ? (
              <div role="alert" className="type-body text-muted-foreground text-center">
                El servicio de cotizaciones no está disponible en este momento. Intenta de
                nuevo más tarde, o contáctanos directamente:
                <div className="mt-4 flex flex-wrap items-center justify-center gap-4">
                  <a
                    href={contact.whatsapp}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="type-button inline-flex min-h-11 items-center rounded-lg bg-accent px-5 text-accent-foreground hover:bg-accent/90"
                  >
                    Escribir por WhatsApp
                  </a>
                </div>
              </div>
            ) : cartEmpty ? (
              <p className="type-body text-muted-foreground text-center">
                Tu selección está vacía.{" "}
                <Link href="/catalogo" className="text-accent underline underline-offset-4">
                  Ver catálogo
                </Link>{" "}
                y agrega productos antes de solicitar cotización.
              </p>
            ) : (
              <>
                <section aria-labelledby="quote-cart-summary-heading" className="rounded-lg border border-border p-4">
                  <h2 id="quote-cart-summary-heading" className="type-h3 font-medium mb-2">
                    Productos en tu selección
                  </h2>
                  <ul className="space-y-1 type-small text-muted-foreground">
                    {cartItems.map((item) => (
                      <li key={item.productId}>
                        {item.name} × {item.quantity}
                      </li>
                    ))}
                  </ul>
                </section>

                <form action={submitQuoteRequestAction} className="space-y-5">
                  <input type="hidden" name="idempotency_key" value={idempotencyKey} />
                  <div className="space-y-1">
                    <label htmlFor="name" className="type-small font-medium">
                      Nombre *
                    </label>
                    <input
                      id="name"
                      name="name"
                      type="text"
                      required
                      minLength={2}
                      autoComplete="name"
                      className="w-full rounded-lg border border-border px-3 py-2"
                    />
                  </div>

                  <div className="space-y-1">
                    <label htmlFor="phone" className="type-small font-medium">
                      Teléfono *
                    </label>
                    <input
                      id="phone"
                      name="phone"
                      type="tel"
                      required
                      minLength={10}
                      autoComplete="tel"
                      className="w-full rounded-lg border border-border px-3 py-2"
                    />
                  </div>

                  <div className="space-y-1">
                    <label htmlFor="email" className="type-small font-medium">
                      Correo (opcional)
                    </label>
                    <input
                      id="email"
                      name="email"
                      type="email"
                      autoComplete="email"
                      className="w-full rounded-lg border border-border px-3 py-2"
                    />
                  </div>

                  <div className="space-y-1">
                    <label htmlFor="event_date" className="type-small font-medium">
                      Fecha del evento (opcional)
                    </label>
                    <input
                      id="event_date"
                      name="event_date"
                      type="date"
                      autoComplete="off"
                      className="w-full rounded-lg border border-border px-3 py-2"
                    />
                  </div>

                  <button
                    type="submit"
                    className="type-button inline-flex min-h-11 items-center rounded-lg bg-accent px-5 text-accent-foreground hover:bg-accent/90 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent"
                  >
                    Enviar solicitud
                  </button>
                </form>
              </>
            )}
          </>
        )}

        <p className="type-body text-muted-foreground text-center">
          También puedes escribirnos directamente:
        </p>
        <div className="flex flex-wrap items-center justify-center gap-4">
          <a
            href={contact.whatsapp}
            target="_blank"
            rel="noopener noreferrer"
            className="type-button inline-flex min-h-11 items-center rounded-lg border border-border px-5 text-foreground hover:border-accent focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent"
          >
            Escribir por WhatsApp
          </a>
          <a
            href={contact.phoneHref}
            className="type-button inline-flex min-h-11 items-center rounded-lg border border-border px-5 text-foreground hover:border-accent focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent"
          >
            Llamar {contact.phone}
          </a>
        </div>
      </div>
    </article>
  );
}
