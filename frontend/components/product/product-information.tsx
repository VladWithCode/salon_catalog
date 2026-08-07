import type { CatalogProductDetail } from "@/lib/types";

type ProductInformationProps = Readonly<{
  product: CatalogProductDetail;
}>;

export function ProductInformation({ product }: ProductInformationProps) {
  const hasDescription = product.description.trim().length > 0;
  const hasLongDescription = product.longDescription.trim().length > 0;

  return (
    <div className="space-y-6">
      <div>
        {product.category ? (
          <p className="type-eyebrow">{product.category.name}</p>
        ) : null}
        <h1 className="type-h1 mt-2 font-medium">{product.name}</h1>
      </div>

      {hasDescription ? (
        <p className="type-lead text-muted-foreground">{product.description}</p>
      ) : null}

      {hasLongDescription ? (
        <div className="space-y-2">
          <h2 className="type-h3 font-medium">Detalles</h2>
          {/* whitespace-pre-line preserves line breaks via CSS only —
              never interprets HTML, matches the "no dangerouslySetInnerHTML"
              requirement. */}
          <p className="type-body measure-body whitespace-pre-line text-muted-foreground">
            {product.longDescription}
          </p>
        </div>
      ) : null}
    </div>
  );
}
