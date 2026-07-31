import Link from "next/link";
import { ArrowUpRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { CheckList } from "@/components/shared/check-list";
import { ImageTile } from "@/components/shared/image-tile";
import { RevealOnView } from "@/lib/motion/reveal-on-view";
import type { EventKindCopy } from "@/lib/copy/home";
import { cn } from "@/lib/utils";

type EventSectionProps = {
  kind: EventKindCopy;
};

/**
 * One event-kind section (Bodas, Quinceañeras, …). Alternates layout
 * (image left/right) based on index, and alternates background between
 * cream and deep cocoa via the `dark` flag in the copy.
 */
export function EventSection({ kind }: EventSectionProps) {
  const isEven = Number(kind.num) % 2 === 0;

  return (
    <section
      id={`seccion-${kind.id}`}
      className={cn(
        "py-section",
        kind.dark ? "bg-primary text-primary-foreground" : "bg-background",
      )}
    >
      <div className="container-page grid items-center gap-10 lg:grid-cols-12 lg:gap-14">
        <RevealOnView
          className={cn("lg:col-span-5", isEven && "lg:order-2")}
          y={20}
        >
          <div className="group">
            <ImageTile
              src={kind.image}
              alt={kind.imageAlt}
              ratio="aspect-[4/3]"
              sizes="(min-width: 1024px) 40vw, 100vw"
              className="shadow-card"
            />
          </div>
        </RevealOnView>

        <RevealOnView
          className={cn("lg:col-span-7", isEven && "lg:order-1")}
          y={20}
          delay={0.1}
        >
          <p className="text-eyebrow uppercase font-medium tracking-[0.18em] text-accent">
            {kind.num}
          </p>
          <h2 className="mt-3 font-display text-display-md font-medium">
            {kind.title}
          </h2>
          <span aria-hidden className="gold-rule mt-5" />
          <p
            className={cn(
              "mt-6 max-w-[60ch] text-body-lg",
              kind.dark ? "text-primary-foreground/75" : "text-muted-foreground",
            )}
          >
            {kind.lede}
          </p>

          <CheckList items={kind.items} dark={kind.dark} className="mt-7" />

          {kind.footnote && (
            <p
              className={cn(
                "mt-6 text-body-sm",
                kind.dark ? "text-primary-foreground/55" : "text-muted-foreground",
              )}
            >
              {kind.footnote}
            </p>
          )}

          <Button
            asChild
            variant="outline"
            className={cn(
              "mt-8",
              kind.dark
                ? "border-accent/60 bg-transparent text-primary-foreground hover:bg-accent hover:text-accent-foreground"
                : "border-primary/30 bg-transparent text-foreground hover:bg-primary hover:text-primary-foreground",
            )}
          >
            <Link href={kind.whatsapp} target="_blank" rel="noopener noreferrer">
              Me interesa saber más
              <ArrowUpRight className="ml-1 h-4 w-4" aria-hidden />
            </Link>
          </Button>
        </RevealOnView>
      </div>
    </section>
  );
}
