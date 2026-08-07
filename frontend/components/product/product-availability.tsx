type ProductAvailabilityProps = Readonly<{
  available: boolean;
}>;

/**
 * Text-only, never color-only — matches the standard already used by
 * internal/templates/components/catalog.templ and by the Go cart flow
 * ("Disponible" / "No disponible"). No role="status": this is static
 * server-rendered content on first load, not a live update, so an
 * assertive/polite live region would be unnecessary aria-live.
 */
export function ProductAvailability({ available }: ProductAvailabilityProps) {
  return (
    <p className="type-small font-medium text-muted-foreground">
      {available ? "Disponible" : "No disponible"}
    </p>
  );
}
