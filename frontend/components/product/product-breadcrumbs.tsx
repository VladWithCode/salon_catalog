import Link from "next/link";

type ProductBreadcrumbsProps = Readonly<{
  productName: string;
}>;

const linkClassName =
  "hover:text-accent-strong focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent";

export function ProductBreadcrumbs({ productName }: ProductBreadcrumbsProps) {
  return (
    <nav aria-label="Ruta de navegación" className="type-small text-muted-foreground">
      <ol className="flex flex-wrap items-center gap-2">
        <li>
          <Link href="/" className={linkClassName}>
            Inicio
          </Link>
        </li>
        <li aria-hidden="true">/</li>
        <li>
          <Link href="/catalogo" className={linkClassName}>
            Catálogo
          </Link>
        </li>
        <li aria-hidden="true">/</li>
        <li aria-current="page" className="text-foreground">
          {productName}
        </li>
      </ol>
    </nav>
  );
}
