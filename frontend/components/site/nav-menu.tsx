"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";

import { primaryNavLinks } from "@/lib/copy/nav";
import { cn } from "@/lib/utils";

function linkIsActive(pathname: string, hash: string, href: string): boolean {
  if (href.startsWith("/#")) {
    return pathname === "/" && hash === href.slice(1);
  }
  if (href === "/") {
    return pathname === "/" && hash.length === 0;
  }
  return pathname === href || pathname.startsWith(`${href}/`);
}

export function NavMenu() {
  const pathname = usePathname();
  const [hash, setHash] = useState("");

  useEffect(() => {
    const updateHash = () => setHash(window.location.hash);
    updateHash();
    window.addEventListener("hashchange", updateHash);
    return () => window.removeEventListener("hashchange", updateHash);
  }, [pathname]);

  return (
    <nav aria-label="Principal" className="hidden md:block">
      <ul className="flex items-center gap-1 lg:gap-3">
        {primaryNavLinks.map((link) => {
          const active = linkIsActive(pathname, hash, link.href);
          return (
            <li key={link.href}>
              <Link
                href={link.href}
                prefetch={false}
                aria-current={active ? "page" : undefined}
                className={cn(
                  "type-small relative flex min-h-11 items-center px-2 font-medium text-foreground/75 hover:text-foreground lg:px-3",
                  active && "text-foreground",
                )}
              >
                {link.label}
                <span
                  aria-hidden="true"
                  className={cn(
                    "absolute inset-x-2 bottom-1 h-0.5 origin-center scale-x-0 bg-accent transition-transform duration-300 ease-elegant",
                    active && "scale-x-100",
                  )}
                />
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
