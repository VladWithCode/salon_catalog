"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";

import { Logo } from "@/components/site/logo";
import { MobileNav } from "@/components/site/mobile-nav";
import { NavMenu } from "@/components/site/nav-menu";
import { contact } from "@/lib/copy/contact";
import { quoteNavLink } from "@/lib/copy/nav";
import { cn } from "@/lib/utils";

type SiteHeaderProps = Readonly<{
  cartCount: number;
}>;

export function SiteHeader({ cartCount }: SiteHeaderProps) {
  const pathname = usePathname();
  const [scrolled, setScrolled] = useState(pathname !== "/");

  useEffect(() => {
    const updateScrolledState = () => {
      setScrolled(pathname !== "/" || window.scrollY > 16);
    };

    updateScrolledState();
    window.addEventListener("scroll", updateScrolledState, { passive: true });
    return () => window.removeEventListener("scroll", updateScrolledState);
  }, [pathname]);

  return (
    <header
      data-scrolled={scrolled}
      className={cn(
        "sticky top-0 z-50 h-16 bg-transparent text-foreground md:h-20",
        "after:absolute after:inset-x-0 after:bottom-0 after:h-px after:origin-center after:scale-x-0 after:bg-accent/40 after:transition-transform after:duration-300 after:ease-elegant",
        !scrolled && pathname === "/" && "text-primary-foreground",
        scrolled &&
          "bg-background/85 shadow-soft backdrop-blur-md after:scale-x-100",
      )}
    >
      <div className="mx-auto flex h-full max-w-7xl items-center justify-between gap-2 px-4 md:gap-3 md:px-4 lg:px-10">
        <Logo variant={!scrolled && pathname === "/" ? "dark" : "light"} />
        <div
          className={cn(
            "hidden md:contents",
            !scrolled && pathname === "/" && "[&_a]:!text-primary-foreground",
          )}
        >
          <NavMenu />
        </div>
        <div className="flex shrink-0 items-center gap-2 lg:gap-3">
          <a
            href={contact.phoneHref}
            aria-label={`Llamar al ${contact.phone}`}
            className={cn(
              "hidden min-h-11 items-center text-xs font-medium text-foreground/75 hover:text-foreground lg:inline-flex",
              !scrolled && pathname === "/" && "!text-primary-foreground/85",
            )}
          >
            {contact.phone}
          </a>
          <Link
            href="/carrito"
            aria-label={
              cartCount > 0
                ? `Ver tu selección, ${cartCount} ${cartCount === 1 ? "producto" : "productos"}`
                : "Ver tu selección"
            }
            className={cn(
              "type-small hidden min-h-11 items-center gap-1 font-medium text-foreground/75 hover:text-foreground md:inline-flex",
              !scrolled && pathname === "/" && "!text-primary-foreground/85",
            )}
          >
            Selección{cartCount > 0 ? ` (${cartCount})` : ""}
          </Link>
          <Link
            href={quoteNavLink.href}
            prefetch={false}
            className={cn(
              "type-button hidden min-h-11 items-center rounded-md border border-accent px-3 font-medium uppercase hover:bg-accent-strong hover:text-primary-foreground md:inline-flex lg:px-4",
              !scrolled && pathname === "/" && "text-primary-foreground",
            )}
          >
            <span className="xl:hidden">{quoteNavLink.label}</span>
            <span className="hidden xl:inline">Solicitar cotización</span>
          </Link>
          <div
            className={cn(
              !scrolled &&
                pathname === "/" &&
                "[&>div>button]:!text-primary-foreground [&>div>button]:hover:!bg-primary/40",
            )}
          >
            <MobileNav />
          </div>
        </div>
      </div>
    </header>
  );
}
