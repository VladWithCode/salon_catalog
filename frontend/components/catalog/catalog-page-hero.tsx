export function CatalogPageHero() {
  return (
    <section
      aria-labelledby="catalog-page-title"
      className="section-dark overflow-hidden"
    >
      <div className="container-main max-w-7xl py-16 sm:py-20 lg:py-24">
        <div className="max-w-4xl space-y-stack">
          <p className="type-eyebrow">Catálogo</p>
          <h1 id="catalog-page-title" className="type-h1 font-medium">
            Catálogo Villa Chenacolo
          </h1>
          <p className="type-lead measure-body text-primary-foreground/75">
            Explora nuestro catálogo de productos. Cada uno de ellos ha sido
            elegido buscando el balance perfecto para ser funcional y tener
            gran estilo.
          </p>
        </div>
      </div>
    </section>
  );
}
