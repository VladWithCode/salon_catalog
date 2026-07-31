import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

type SectionHeadingProps = {
  /** Small uppercase kicker rendered above the title. */
  eyebrow?: string;
  /** The title content. Pass a string or a React node for italicized words. */
  title: ReactNode;
  /** Optional supporting paragraph under the title. */
  lede?: string;
  /** Alignment of the heading block. Default "center". */
  align?: "left" | "center";
  /** When true, render in colors suitable for a dark (primary) background. */
  dark?: boolean;
  /** Heading size. Default "lg". */
  size?: "md" | "lg" | "xl";
  /** Render a small gold rule under the title. Default false. */
  rule?: boolean;
  className?: string;
  /** Heading level. Default "h2". */
  as?: "h1" | "h2" | "h3";
};

const sizeClasses = {
  md: "text-display-sm md:text-display-md",
  lg: "text-display-md md:text-display-lg",
  xl: "text-display-lg md:text-display-xl",
} as const;

/**
 * Shared section heading: eyebrow → title → optional gold rule → optional lede.
 * Used across the home page and reusable on every other page.
 */
export function SectionHeading({
  eyebrow,
  title,
  lede,
  align = "center",
  dark = false,
  size = "lg",
  rule = false,
  className,
  as: Heading = "h2",
}: SectionHeadingProps) {
  return (
    <div
      className={cn(
        "max-w-3xl",
        align === "center" && "mx-auto text-center",
        className,
      )}
    >
      {eyebrow && (
        <p className="text-eyebrow uppercase font-medium tracking-[0.18em] text-accent mb-4">
          {eyebrow}
        </p>
      )}
      <Heading
        className={cn(
          "font-display font-medium text-balance",
          sizeClasses[size],
          dark ? "text-primary-foreground" : "text-foreground",
        )}
      >
        {title}
      </Heading>
      {rule && (
        <span
          aria-hidden
          className={cn(
            "gold-rule mt-6",
            align === "center" && "mx-auto",
          )}
        />
      )}
      {lede && (
        <p
          className={cn(
            "mt-6 text-body-lg text-pretty",
            dark ? "text-primary-foreground/70" : "text-muted-foreground",
          )}
        >
          {lede}
        </p>
      )}
    </div>
  );
}

/**
 * Splits `title` around `word` and renders `word` in italic Playfair.
 * Falls back to the plain title if the word isn't found.
 */
export function EmphasizedTitle({
  title,
  word,
}: {
  title: string;
  word?: string;
}) {
  if (!word) return <>{title}</>;
  const idx = title.toLowerCase().indexOf(word.toLowerCase());
  if (idx === -1) return <>{title}</>;
  const before = title.slice(0, idx);
  const match = title.slice(idx, idx + word.length);
  const after = title.slice(idx + word.length);
  return (
    <>
      {before}
      <em className="italic">{match}</em>
      {after}
    </>
  );
}
