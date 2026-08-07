import { Check } from "lucide-react";

import { cn } from "@/lib/utils";

type CheckListProps = Readonly<{
  items: readonly string[];
  inverted?: boolean;
}>;

export function CheckList({ items, inverted = false }: CheckListProps) {
  return (
    <ul className="space-y-3">
      {items.map((item) => (
        <li
          key={item}
          className={cn(
            "type-body flex items-start gap-3 text-muted-foreground",
            inverted && "text-primary-foreground/75",
          )}
        >
          <Check
            aria-hidden="true"
            className="mt-1.5 size-4 shrink-0 text-accent"
          />
          <span>{item}</span>
        </li>
      ))}
    </ul>
  );
}
