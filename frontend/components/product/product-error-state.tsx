import Link from "next/link";

/**
 * Shown only for catalog_unavailable / backend_unavailable /
 * invalid_response / unexpected_status — invalid_identifier and
 * product_not_found are handled by notFound() before this component is
 * ever reached (see page.tsx). Copy is fixed and identical across those
 * four codes on purpose: none of them is something a visitor can act on
 * differently, and none may ever surface a remote status code, URL, or
 * technical detail.
 */
export function ProductErrorState() {
  return (
    <div className="section-light py-section">
      <div className="container-main max-w-2xl space-y-4 text-center">
        <h1 className="type-h2 font-medium">No pudimos cargar este producto</h1>
        <p className="type-body text-muted-foreground">
          El catálogo no está disponible en este momento. Intenta de nuevo más
          tarde.
        </p>
        <Link
          href="/catalogo"
          className="type-button inline-flex min-h-11 items-center rounded-lg bg-accent-strong px-5 text-primary-foreground hover:bg-accent-strong/90 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent"
        >
          Volver al catálogo
        </Link>
      </div>
    </div>
  );
}
