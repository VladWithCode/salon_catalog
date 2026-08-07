import { ArrowUpRight } from "lucide-react";
import Image from "next/image";

import { CheckList } from "@/components/shared/check-list";
import type { EventSectionCopy } from "@/lib/copy/home";
import { cn } from "@/lib/utils";

type EventSectionProps = Readonly<{
  event: EventSectionCopy;
}>;

export function EventSection({ event }: EventSectionProps) {
  const inverted = event.variant === "dark";
  const imageOnRight = event.alignment === "image-right";

  return (
    <section
      id={event.id}
      aria-labelledby={`${event.id}-title`}
      className={cn("py-section scroll-mt-20", inverted ? "section-dark" : "section-light")}
    >
      <div className="container-main grid max-w-7xl gap-10 lg:grid-cols-12 lg:items-center lg:gap-14">
        <div
          className={cn(
            "order-2 lg:col-span-5",
            imageOnRight ? "lg:order-2" : "lg:order-1",
          )}
        >
          <div className="group relative aspect-[4/3] overflow-hidden rounded-xl shadow-card">
            <Image
              src={event.image.src}
              alt={event.image.alt}
              fill
              sizes="(min-width: 1024px) 42vw, 100vw"
              className="object-cover transition-transform duration-500 ease-elegant group-hover:scale-[1.02]"
            />
          </div>
        </div>

        <div
          className={cn(
            "order-1 space-y-6 lg:col-span-7",
            imageOnRight ? "lg:order-1" : "lg:order-2",
          )}
        >
          <div>
            <p className="type-eyebrow">
              {event.number} · {event.eyebrow}
            </p>
            <h2
              id={`${event.id}-title`}
              className="type-h2 mt-4 font-medium"
            >
              {event.title}
            </h2>
            <div aria-hidden="true" className="mt-5 h-0.5 w-12 bg-accent" />
          </div>
          <p
            className={cn(
              "type-lead measure-body text-muted-foreground",
              inverted && "text-primary-foreground/80",
            )}
          >
            {event.description}
          </p>
          <CheckList items={event.highlights} inverted={inverted} />
          {event.footnote ? (
            <p
              className={cn(
                "type-small text-muted-foreground",
                inverted && "text-primary-foreground/70",
              )}
            >
              {event.footnote}
            </p>
          ) : null}
          <a
            href={event.cta.href}
            target="_blank"
            rel="noopener noreferrer"
            className={cn(
              "type-button inline-flex min-h-12 items-center justify-center gap-2 rounded-md border border-accent px-5 font-semibold uppercase hover:bg-accent-strong hover:text-primary-foreground",
              inverted && "text-primary-foreground",
            )}
          >
            {event.cta.label}
            <ArrowUpRight aria-hidden="true" className="size-4" />
          </a>
        </div>
      </div>
    </section>
  );
}
