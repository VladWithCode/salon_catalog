"use client";

import { ArrowDown, ArrowRight } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { useSyncExternalStore } from "react";

import { useReducedMotion } from "@/hooks/use-reduced-motion";
import { homeCopy } from "@/lib/copy/home";

const subscribeToHydration = () => () => undefined;
const getClientSnapshot = () => true;
const getServerSnapshot = () => false;

export function HomeHero() {
  const reducedMotion = useReducedMotion();
  const hydrated = useSyncExternalStore(
    subscribeToHydration,
    getClientSnapshot,
    getServerSnapshot,
  );
  const { hero } = homeCopy;

  return (
    <section
      id="inicio"
      aria-labelledby="home-title"
      className="relative -mt-16 flex min-h-[88vh] items-center justify-center overflow-hidden bg-primary px-stack pb-16 pt-32 text-primary-foreground md:-mt-20 md:min-h-[92vh] md:pb-20 md:pt-40"
    >
      <div className="absolute inset-0 z-0" aria-hidden="true">
        {hydrated && !reducedMotion ? (
          <video
            autoPlay
            muted
            loop
            playsInline
            preload="metadata"
            poster={hero.poster.src}
            tabIndex={-1}
            className="absolute inset-0 z-10 size-full object-cover"
          >
            <source src={hero.video.webm} type="video/webm" />
            <source src={hero.video.mp4} type="video/mp4" />
          </video>
        ) : null}
        <Image
          src={hero.poster.src}
          alt=""
          fill
          priority
          fetchPriority="high"
          sizes="100vw"
          className="object-cover object-center"
        />
        <div className="absolute inset-0 z-20 bg-gradient-to-b from-primary/60 via-primary/50 to-primary/70" />
      </div>

      <div className="relative z-30 mx-auto flex w-full max-w-5xl flex-col items-center text-center">
        <Image
          src={hero.wordmark.src}
          alt={hero.wordmark.alt}
          width={hero.wordmark.width}
          height={hero.wordmark.height}
          sizes="(min-width: 1024px) 32rem, (min-width: 768px) 26rem, 18rem"
          className="mb-7 h-auto w-[min(18rem,78vw)] md:w-[26rem] lg:w-[32rem]"
        />
        <p className="type-eyebrow mb-6 text-center text-primary-foreground/80">
          {hero.eyebrow}
        </p>
        <h1
          id="home-title"
          className="font-display text-5xl leading-[1.02] font-medium tracking-[-0.02em] text-balance md:text-7xl lg:text-8xl"
        >
          {hero.title}
        </h1>
        <p className="type-lead mx-auto mt-6 max-w-2xl text-center text-primary-foreground/85">
          {hero.intro}
        </p>
        <div className="mt-9 flex flex-col items-center gap-3 sm:flex-row">
          <Link
            href={hero.primaryCta.href}
            prefetch={false}
            className="type-button inline-flex min-h-12 items-center justify-center gap-1.5 rounded-md bg-accent-strong px-4 text-[0.8125rem] font-semibold uppercase text-primary-foreground shadow-elevated hover:bg-secondary sm:gap-2 sm:px-6 sm:text-button"
          >
            {hero.primaryCta.label}
            <ArrowRight aria-hidden="true" className="size-4" />
          </Link>
          <Link
            href={hero.secondaryCta.href}
            className="type-button inline-flex min-h-12 items-center justify-center gap-1.5 rounded-md border border-primary-foreground/55 px-4 text-[0.8125rem] font-semibold uppercase text-primary-foreground hover:bg-primary-foreground hover:text-primary sm:gap-2 sm:px-6 sm:text-button"
          >
            {hero.secondaryCta.label}
            <ArrowDown aria-hidden="true" className="size-4" />
          </Link>
        </div>
      </div>
    </section>
  );
}
