import { ArrowRight } from "lucide-react";
import Image from "next/image";
import Link from "next/link";

import { servicesCopy } from "@/lib/copy/services";

export function ServicesHero() {
  const { hero } = servicesCopy;

  return (
    <section
      id="seccion-titulo"
      aria-labelledby="services-title"
      className="relative isolate flex min-h-[26rem] items-center overflow-hidden bg-primary py-block text-primary-foreground md:min-h-[34rem]"
    >
      <div aria-hidden="true" className="absolute inset-0 -z-10">
        <Image
          src={hero.background.src}
          alt=""
          fill
          priority
          fetchPriority="high"
          sizes="100vw"
          className="object-cover object-center"
        />
        <div className="absolute inset-0 image-overlay-gradient" />
      </div>

      <div className="container-main max-w-7xl">
        <div className="max-w-3xl">
          <p className="type-eyebrow">{hero.eyebrow}</p>
          <h1 id="services-title" className="type-h1 mt-5 font-medium">
            {hero.title}
          </h1>
          <p className="type-lead measure-body mt-6 text-primary-foreground/85">
            {hero.intro}
          </p>
          <Link
            href={hero.primaryCta.href}
            prefetch={false}
            className="type-button mt-9 inline-flex min-h-12 items-center justify-center gap-2 rounded-md bg-accent-strong px-6 font-semibold uppercase text-primary-foreground shadow-elevated hover:bg-secondary hover:text-secondary-foreground"
          >
            {hero.primaryCta.label}
            <ArrowRight aria-hidden="true" className="size-4" />
          </Link>
        </div>
      </div>
    </section>
  );
}
