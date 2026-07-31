import { RevealOnView } from "@/lib/motion/reveal-on-view";
import { aboutStrip } from "@/lib/copy/home";

/**
 * "¿Quiénes somos?" — a single ceremonial pull-quote moment on a deep
 * cocoa background between the gallery and the contact form.
 */
export function AboutStrip() {
  return (
    <section id="seccion-nosotros" className="relative overflow-hidden bg-primary text-primary-foreground">
      {/* Subtle texture */}
      <div
        aria-hidden
        className="absolute inset-0 opacity-[0.06]"
        style={{
          backgroundImage: "url(/assets/bg_pattern.png)",
          backgroundRepeat: "repeat",
          backgroundSize: "16rem",
        }}
      />
      <div className="container-page relative py-24 md:py-32">
        <RevealOnView duration={0.7} className="mx-auto max-w-3xl text-center">
          <p className="text-eyebrow uppercase font-medium tracking-[0.18em] text-accent">
            {aboutStrip.eyebrow}
          </p>
          <p className="mt-6 font-display text-display-sm leading-snug md:text-display-md">
            {aboutStrip.paragraphs[0]}
          </p>
          <p className="mt-6 font-display text-display-sm italic leading-snug text-primary-foreground/85 md:text-display-md">
            {aboutStrip.paragraphs[1]}
          </p>
          <span aria-hidden className="gold-rule mx-auto mt-10" />
        </RevealOnView>
      </div>
    </section>
  );
}
