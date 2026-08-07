"use client";

import { MapPin, Menu, Phone, X } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState } from "react";

import { Logo } from "@/components/site/logo";
import { contact } from "@/lib/copy/contact";
import { primaryNavLinks, quoteNavLink } from "@/lib/copy/nav";
import { cn } from "@/lib/utils";

const mobileNavigationID = "mobile-site-navigation";

export function MobileNav() {
  const pathname = usePathname();
  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) {
      return;
    }

    const trigger = triggerRef.current;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    const frame = window.requestAnimationFrame(() => {
      const firstFocusable = panelRef.current?.querySelector<HTMLElement>(
        'button, a[href], [tabindex]:not([tabindex="-1"])',
      );
      firstFocusable?.focus();
    });

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        setOpen(false);
        return;
      }

      if (event.key !== "Tab" || !panelRef.current) {
        return;
      }

      const focusable = Array.from(
        panelRef.current.querySelectorAll<HTMLElement>(
          'button:not([disabled]), a[href], [tabindex]:not([tabindex="-1"])',
        ),
      );
      const first = focusable.at(0);
      const last = focusable.at(-1);

      if (!first || !last) {
        event.preventDefault();
        panelRef.current.focus();
        return;
      }

      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };

    document.addEventListener("keydown", handleKeyDown);

    return () => {
      window.cancelAnimationFrame(frame);
      document.removeEventListener("keydown", handleKeyDown);
      document.body.style.overflow = previousOverflow;
      trigger?.focus();
    };
  }, [open]);

  const close = () => setOpen(false);

  return (
    <div className="md:hidden">
      <button
        ref={triggerRef}
        type="button"
        aria-label={open ? "Cerrar menú principal" : "Abrir menú principal"}
        aria-expanded={open}
        aria-controls={mobileNavigationID}
        onClick={() => setOpen((current) => !current)}
        className="inline-flex size-11 items-center justify-center rounded-md text-foreground hover:bg-secondary"
      >
        {open ? <X aria-hidden="true" /> : <Menu aria-hidden="true" />}
      </button>

      {open ? (
        <div
          className="fixed inset-0 z-[60] bg-primary/55"
          onMouseDown={(event) => {
            if (event.target === event.currentTarget) {
              close();
            }
          }}
        >
          <div
            ref={panelRef}
            id={mobileNavigationID}
            role="dialog"
            aria-modal="true"
            aria-labelledby="mobile-navigation-title"
            tabIndex={-1}
            className="absolute inset-y-0 right-0 flex w-[min(20rem,90vw)] flex-col overflow-y-auto bg-background p-6 shadow-elevated"
          >
            <div className="flex items-center justify-between gap-4 border-b pb-5">
              <div id="mobile-navigation-title">
                <Logo />
                <span className="sr-only">Menú principal</span>
              </div>
              <button
                type="button"
                aria-label="Cerrar menú principal"
                onClick={close}
                className="inline-flex size-11 shrink-0 items-center justify-center rounded-md hover:bg-secondary"
              >
                <X aria-hidden="true" />
              </button>
            </div>

            <nav aria-label="Principal móvil" className="py-5">
              <ul>
                {primaryNavLinks.map((link) => {
                  const active =
                    link.href === "/"
                      ? pathname === "/"
                      : pathname === link.href ||
                        pathname.startsWith(`${link.href}/`);
                  return (
                    <li key={link.href}>
                      <Link
                        href={link.href}
                        prefetch={false}
                        onClick={close}
                        aria-current={active ? "page" : undefined}
                        className={cn(
                          "flex min-h-11 items-center border-b py-4 text-lg font-medium hover:text-accent",
                          active && "text-accent",
                        )}
                      >
                        {link.label}
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </nav>

            <div className="mt-auto space-y-4 border-t pt-6 text-sm text-muted-foreground">
              <p className="flex items-start gap-3">
                <MapPin aria-hidden="true" className="mt-0.5 size-5 shrink-0" />
                <span>{contact.address}</span>
              </p>
              <a
                href={contact.phoneHref}
                className="flex min-h-11 items-center gap-3 hover:text-foreground"
              >
                <Phone aria-hidden="true" className="size-5 shrink-0" />
                {contact.phone}
              </a>
              <Link
                href={quoteNavLink.href}
                prefetch={false}
                onClick={close}
                className="type-button flex min-h-12 items-center justify-center rounded-md bg-primary px-5 font-medium uppercase text-primary-foreground shadow-soft hover:bg-accent hover:text-accent-foreground"
              >
                Solicitar cotización
              </Link>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
