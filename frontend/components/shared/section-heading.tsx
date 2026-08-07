import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

type SectionHeadingProps = Readonly<{
  eyebrow: string;
  title: ReactNode;
  lede?: string;
  align?: "start" | "center";
  inverted?: boolean;
  className?: string;
}>;

export function SectionHeading({
  eyebrow,
  title,
  lede,
  align = "start",
  inverted = false,
  className,
}: SectionHeadingProps) {
  return (
    <header
      className={cn(
        "space-y-stack",
        align === "center" && "mx-auto max-w-4xl text-center",
        className,
      )}
    >
      <p
        className={cn(
          "type-eyebrow",
          align === "center" && "text-center",
          inverted && "text-accent-on-dark",
        )}
      >
        {eyebrow}
      </p>
      <h2
        className={cn(
          "type-h1 font-medium",
          align === "center" && "text-center",
        )}
      >
        {title}
      </h2>
      {lede ? (
        <p
          className={cn(
            "type-lead measure-body text-muted-foreground",
            align === "center" && "mx-auto text-center",
            inverted && "text-primary-foreground/75",
          )}
        >
          {lede}
        </p>
      ) : null}
    </header>
  );
}
