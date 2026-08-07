import { ArrowRight } from "lucide-react";
import Link from "next/link";

import { servicesCopy } from "@/lib/copy/services";

export function ServicesCta() {
  const { closing } = servicesCopy;

  return (
    <section
      id={closing.id}
      aria-labelledby={`${closing.id}-title`}
      className="section-highlight py-section scroll-mt-20"
    >
      <div className="container-main max-w-5xl">
        <div className="rounded-xl border bg-card p-8 text-center shadow-card sm:p-12 lg:p-16">
          <p className="type-eyebrow text-center">{closing.eyebrow}</p>
          <h2
            id={`${closing.id}-title`}
            className="type-h2 mt-5 text-center font-medium"
          >
            {closing.title}
          </h2>
          <div className="mt-8 flex flex-col items-center gap-4">
            <Link
              href={closing.cta.href}
              prefetch={false}
              className="type-button inline-flex min-h-12 items-center gap-2 rounded-md bg-primary px-6 font-semibold uppercase text-primary-foreground shadow-soft hover:bg-accent hover:text-accent-foreground"
            >
              {closing.cta.label}
              <ArrowRight aria-hidden="true" className="size-4" />
            </Link>
            <a
              href={closing.whatsappHref}
              target="_blank"
              rel="noopener noreferrer"
              className="type-small inline-flex min-h-11 items-center text-primary underline decoration-accent underline-offset-4 hover:text-accent"
            >
              {closing.whatsappLabel}
            </a>
          </div>
        </div>
      </div>
    </section>
  );
}
