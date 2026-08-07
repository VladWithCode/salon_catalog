import Link from "next/link";

/**
 * Segment-local not-found so a broken/old link (or a stale QR-adjacent
 * slug) lands on something that still looks like the site instead of
 * Next's bare default 404. Copy mirrors Go's own "No se encontró el
 * producto" (internal/routes/catalog.go).
 */
export default function ProductNotFound() {
  return (
    <div className="section-light py-section">
      <div className="container-main max-w-2xl space-y-4 text-center">
        <h1 className="type-h2 font-medium">No encontramos ese producto</h1>
        <p className="type-body text-muted-foreground">
          Puede que el producto ya no esté disponible o que el enlace sea
          incorrecto.
        </p>
        <Link
          href="/catalogo"
          className="type-button inline-flex min-h-11 items-center rounded-lg bg-accent px-5 text-accent-foreground hover:bg-accent/90 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent"
        >
          Volver al catálogo
        </Link>
      </div>
    </div>
  );
}
