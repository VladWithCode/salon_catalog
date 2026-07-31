import Image from "next/image";
import Link from "next/link";
import { ChevronDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import { FadeIn } from "@/lib/motion/fade-in";
import { brand } from "@/lib/copy/contact";
import { hero } from "@/lib/copy/home";

/**
 * Full-viewport hero: background video with a warm gradient overlay, the
 * wordmark, an eyebrow, a display title, and the primary CTA.
 */
export function HomeHero() {
  return (
    <section
      id="seccion-hero"
      aria-label={brand.name}
      className="relative flex min-h-[92vh] items-center justify-center overflow-hidden"
    >
      {/* Background video */}
      <div className="absolute inset-0 -z-10">
        <video
          className="h-full w-full object-cover"
          autoPlay
          muted
          loop
          playsInline
          preload="metadata"
          poster="/assets/chenacolo_3.jpeg"
          aria-hidden
        >
          <source src="/assets/chenacolo_vid.webm" type="video/webm" />
          <source src="/assets/chenacolo_vid.mp4" type="video/mp4" />
        </video>
        {/* Warm gradient overlay so text is always legible */}
        <div className="absolute inset-0 bg-gradient-to-b from-primary/55 via-primary/40 to-primary/70" />
      </div>

      {/* Foreground */}
      <div className="container-page flex flex-col items-center pb-24 pt-28 text-center text-primary-foreground">
        <FadeIn duration={0.8} y={12}>
          <Image
            src="/assets/logo_name_white.png"
            alt={brand.name}
            width={512}
            height={141}
            priority
            className="h-24 w-auto md:h-36 lg:h-40"
          />
        </FadeIn>

        <FadeIn delay={0.2} duration={0.8} y={10}>
          <h1 className="mt-8 max-w-4xl text-balance font-display text-display-lg font-medium md:text-display-xl">
            <span className="sr-only">{brand.name} — {hero.eyebrow}. </span>
            {hero.title}
          </h1>
        </FadeIn>

        <FadeIn delay={0.45} duration={0.8} y={10}>
          <div className="mt-10 flex flex-col items-center gap-4 sm:flex-row">
            <Button
              asChild
              size="lg"
              className="bg-accent text-accent-foreground hover:bg-accent/90"
            >
              <Link href={hero.primaryCta.href}>{hero.primaryCta.label}</Link>
            </Button>
            <Button
              asChild
              size="lg"
              variant="ghost"
              className="text-primary-foreground hover:bg-primary-foreground/10 hover:text-primary-foreground"
            >
              <Link href={hero.secondaryCta.href}>
                {hero.secondaryCta.label}
                <ChevronDown className="ml-1 h-4 w-4" aria-hidden />
              </Link>
            </Button>
          </div>
        </FadeIn>
      </div>
    </section>
  );
}
