import { Check } from "lucide-react";
import { cn } from "@/lib/utils";

type CheckListProps = {
  items: readonly string[];
  /** When true, render in colors suitable for a dark background. */
  dark?: boolean;
  className?: string;
};

/**
 * Feature checklist with a gold check icon per item.
 * Used by every event section on the home page.
 */
export function CheckList({ items, dark = false, className }: CheckListProps) {
  return (
    <ul className={cn("space-y-3", className)}>
      {items.map((item) => (
        <li key={item} className="flex items-start gap-3">
          <Check
            aria-hidden
            className="mt-1 h-4 w-4 shrink-0 text-accent"
            strokeWidth={2.5}
          />
          <span
            className={cn(
              "text-body",
              dark ? "text-primary-foreground/80" : "text-foreground/80",
            )}
          >
            {item}
          </span>
        </li>
      ))}
    </ul>
  );
}
