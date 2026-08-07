import Image from "next/image";

import { homeCopy } from "@/lib/copy/home";

export function AboutStrip() {
  const { about } = homeCopy;

  return (
    <section
      id="seccion-nosotros"
      aria-labelledby="about-title"
      className="section-dark relative overflow-hidden py-section"
    >
      <Image
        src={about.pattern.src}
        alt={about.pattern.alt}
        fill
        sizes="100vw"
        className="object-cover opacity-[0.08]"
      />
      <div className="container-main relative z-10 max-w-4xl">
        <div className="space-y-stack">
          <p className="type-eyebrow text-center">{about.eyebrow}</p>
          <h2 id="about-title" className="sr-only">
            {about.eyebrow}
          </h2>
          <p className="type-h2 text-center font-display font-medium">
            {about.paragraphs[0]}
          </p>
          <p className="type-h3 text-center font-display italic text-primary-foreground/85">
            {about.paragraphs[1]}
          </p>
          <div aria-hidden="true" className="mx-auto h-px w-16 bg-accent" />
        </div>
      </div>
    </section>
  );
}
