"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Phone } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Logo } from "./logo";
import { NavMenu } from "./nav-menu";
import { MobileNav } from "./mobile-nav";
import { contact } from "@/lib/copy/contact";
import { cn } from "@/lib/utils";

/**
 * Sticky site header.
 *
 * - Transparent over the hero, with light (cream) text.
 * - On scroll (>16px): frosted cream background, dark text, soft shadow,
 *   and a 1px gold underline that slides in.
 */
export function SiteHeader() {
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 16);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  return (
    <header
      data-scrolled={scrolled || undefined}
      className={cn(
        "group fixed inset-x-0 top-0 z-50 transition-[background-color,box-shadow,color] duration-300",
        scrolled
          ? "bg-background/85 text-foreground shadow-soft backdrop-blur-md"
          : "bg-transparent text-primary-foreground",
      )}
    >
      <div className="container-page flex h-16 items-center justify-between gap-4 md:h-20">
        <Logo
          variant={scrolled ? "light" : "dark"}
          collapseOnMobile
          className="relative z-10"
        />

        <NavMenu className="hidden md:flex" />

        <div className="flex items-center gap-2 md:gap-4">
          <a
            href={contact.phoneHref}
            className="hidden items-center gap-2 text-body-sm font-medium transition-colors hover:text-accent lg:flex"
          >
            <Phone className="h-4 w-4" aria-hidden />
            {contact.phoneDisplay}
          </a>
          <Button
            asChild
            size="sm"
            className={cn(
              "hidden md:inline-flex",
              scrolled
                ? "bg-primary text-primary-foreground hover:bg-primary/90"
                : "bg-accent text-accent-foreground hover:bg-accent/90",
            )}
          >
            <Link href="/solicitar-cotizacion">Cotizar</Link>
          </Button>
          <MobileNav />
        </div>
      </div>

      {/* Gold hairline underline, slides in when scrolled */}
      <span
        aria-hidden
        className={cn(
          "pointer-events-none absolute inset-x-0 bottom-0 h-px origin-center bg-accent/40 transition-transform duration-300",
          scrolled ? "scale-x-100" : "scale-x-0",
        )}
      />
    </header>
  );
}
