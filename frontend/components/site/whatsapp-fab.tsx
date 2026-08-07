import { MessageCircle } from "lucide-react";

import { contact } from "@/lib/copy/contact";

// 02-shared-layout.md §6/§9 wants the FAB reachable at every breakpoint
// (icon-only under sm, pill from sm up). It previously carried
// "md:hidden lg:inline-flex", which left 768–1023px with no WhatsApp
// affordance at all — the header CTA in that range is "Cotizar", not
// WhatsApp. Kept visible throughout instead.
export function WhatsAppFab() {
  return (
    <a
      href={contact.whatsapp}
      target="_blank"
      rel="noopener noreferrer"
      aria-label="Chatea con nosotros por WhatsApp"
      className="fixed right-4 bottom-4 z-40 inline-flex h-12 min-w-12 items-center justify-center gap-2 rounded-full bg-[#25D366] px-3 text-primary shadow-elevated transition-transform duration-200 ease-elegant hover:scale-[1.03] print:hidden sm:right-6 sm:bottom-6 sm:px-5"
    >
      <MessageCircle aria-hidden="true" className="size-5" />
      <span className="hidden text-sm font-medium sm:inline">WhatsApp</span>
    </a>
  );
}
