"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { navLinks } from "@/lib/copy/nav";
import { cn } from "@/lib/utils";

type NavMenuProps = {
  className?: string;
};

/**
 * Desktop navigation menu. The active link gets a gold underline.
 */
export function NavMenu({ className }: NavMenuProps) {
  const pathname = usePathname();

  return (
    <nav aria-label="Principal" className={cn("items-center gap-1", className)}>
      {navLinks
        .filter((l) => !l.emphasis)
        .map((link) => {
          const isActive =
            link.href === "/"
              ? pathname === "/"
              : pathname.startsWith(link.href.split("#")[0]) &&
                link.href.split("#")[0] !== "/";
          return (
            <Link
              key={link.href}
              href={link.href}
              aria-current={isActive ? "page" : undefined}
              className={cn(
                "relative px-3 py-2 text-button uppercase tracking-[0.08em] transition-colors",
                "hover:text-accent",
                isActive ? "text-accent" : "text-current",
              )}
            >
              {link.label}
              {isActive && (
                <span
                  aria-hidden
                  className="absolute inset-x-3 -bottom-0.5 h-px bg-accent"
                />
              )}
            </Link>
          );
        })}
    </nav>
  );
}
