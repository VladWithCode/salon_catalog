"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Menu, MapPin, Phone } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Logo } from "./logo";
import { navLinks } from "@/lib/copy/nav";
import { brand, contact } from "@/lib/copy/contact";
import { cn } from "@/lib/utils";

/**
 * Mobile navigation: hamburger button that opens a right-side sheet.
 * Closes itself on navigation (pathname change).
 */
export function MobileNav() {
  const [open, setOpen] = useState(false);
  const pathname = usePathname();
  const [prevPathname, setPrevPathname] = useState(pathname);

  // Close the sheet when the route changes. Adjusting state during render
  // (instead of in an effect) avoids a paint with the stale menu open.
  if (prevPathname !== pathname) {
    setPrevPathname(pathname);
    setOpen(false);
  }

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="md:hidden text-current"
          aria-label="Abrir menú"
          aria-expanded={open}
        >
          <Menu className="h-6 w-6" />
        </Button>
      </SheetTrigger>
      <SheetContent
        side="right"
        className="w-[min(20rem,90vw)] bg-background p-0 flex flex-col"
      >
        <SheetHeader className="bg-primary px-6 py-6 text-left">
          <SheetTitle className="sr-only">{brand.name}</SheetTitle>
          <Logo variant="dark" />
          <p className="text-body-sm text-primary-foreground/70">
            {brand.tagline}
          </p>
        </SheetHeader>

        <nav aria-label="Menú móvil" className="flex-1 overflow-y-auto px-2 py-4">
          <ul className="space-y-1">
            {navLinks.map((link) => (
              <li key={link.href}>
                <Link
                  href={link.href}
                  className={cn(
                    "block rounded-md px-4 py-3 text-lg transition-colors",
                    link.emphasis
                      ? "font-semibold text-accent"
                      : "text-foreground hover:bg-secondary hover:text-foreground",
                  )}
                >
                  {link.label}
                </Link>
              </li>
            ))}
          </ul>
        </nav>

        <div className="border-t border-border px-6 py-5 space-y-3 text-body-sm text-muted-foreground">
          <p className="flex items-start gap-3">
            <MapPin className="h-4 w-4 mt-0.5 shrink-0 text-accent" aria-hidden />
            <span>{contact.address}</span>
          </p>
          <p className="flex items-center gap-3">
            <Phone className="h-4 w-4 shrink-0 text-accent" aria-hidden />
            <a href={contact.phoneHref} className="hover:text-accent">
              {contact.phoneDisplay}
            </a>
          </p>
          <Button asChild className="w-full bg-accent text-accent-foreground hover:bg-accent/90 mt-2">
            <Link href="/solicitar-cotizacion">Solicitar cotización</Link>
          </Button>
        </div>
      </SheetContent>
    </Sheet>
  );
}
