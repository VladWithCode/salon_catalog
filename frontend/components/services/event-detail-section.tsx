import { ArrowUpRight } from "lucide-react";
import Image from "next/image";

import { CheckList } from "@/components/shared/check-list";
import type { EventKey } from "@/lib/copy/events";
import { cn } from "@/lib/utils";

type EventDetailSectionProps = Readonly<{
  id: string;
  eventKey: EventKey;
  eyebrow: string;
  title: string;
  description: string;
  highlights: readonly string[];
  footnote?: string;
  image: string;
  imageWidth: number;
  imageHeight: number;
  alt: string;
  alignment: "image-left" | "image-right";
  variant: "light" | "dark";
  ctaLabel: string;
  href: string;
}>;

export function EventDetailSection({
  id,
  eventKey,
  eyebrow,
  title,
  description,
  highlights,
  footnote,
  image,
  imageWidth,
  imageHeight,
  alt,
  alignment,
  variant,
  ctaLabel,
  href,
}: EventDetailSectionProps) {
  const inverted = variant === "dark";
  const imageOnRight = alignment === "image-right";
  const portrait = imageHeight > imageWidth;

  return (
    <section
      id={id}
      aria-labelledby={`${id}-title`}
      className={cn(
        "py-section scroll-mt-20",
        inverted ? "section-dark" : "section-light",
      )}
    >
      <div className="container-main grid max-w-7xl gap-10 lg:grid-cols-2 lg:items-center lg:gap-14">
        <div className={cn("min-w-0 space-y-6", imageOnRight ? "lg:order-1" : "lg:order-2")}>
          <div>
            <p className="type-eyebrow">{eyebrow}</p>
            <h2 id={`${id}-title`} className="type-h2 mt-4 font-medium">
              {title}
            </h2>
            <div aria-hidden="true" className="mt-5 h-0.5 w-12 bg-accent" />
          </div>
          <p
            className={cn(
              "type-lead measure-body text-muted-foreground",
              inverted && "text-primary-foreground/80",
            )}
          >
            {description}
          </p>
          <CheckList items={highlights} inverted={inverted} />
          {footnote ? (
            <p
              className={cn(
                "type-small text-muted-foreground",
                inverted && "text-primary-foreground/70",
              )}
            >
              {footnote}
            </p>
          ) : null}
          <a
            href={href}
            target="_blank"
            rel="noopener noreferrer"
            aria-label={`${ctaLabel} sobre ${title.toLocaleLowerCase("es")} por WhatsApp`}
            data-event={eventKey}
            className={cn(
              "type-button inline-flex min-h-12 items-center justify-center gap-2 rounded-md border border-accent px-5 font-semibold uppercase hover:bg-accent-strong hover:text-primary-foreground",
              inverted && "text-primary-foreground",
            )}
          >
            {ctaLabel}
            <ArrowUpRight aria-hidden="true" className="size-4" />
          </a>
        </div>

        <div className={cn("min-w-0", imageOnRight ? "lg:order-2" : "lg:order-1")}>
          <div
            className={cn(
              "relative overflow-hidden rounded-xl shadow-card",
              portrait ? "aspect-[3/4]" : "aspect-[3/2]",
            )}
          >
            <Image
              src={image}
              alt={alt}
              fill
              sizes="(min-width: 1280px) 36rem, (min-width: 1024px) 44vw, calc(100vw - 3rem)"
              className="object-cover object-center"
            />
          </div>
        </div>
      </div>
    </section>
  );
}
