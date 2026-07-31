import { contact } from "@/lib/copy/contact";
import { WhatsAppIcon } from "./social-icons";

/**
 * Floating WhatsApp pill, bottom-right. Always available, opens a new chat
 * with a pre-filled message.
 */
export function WhatsAppFab() {
  return (
    <a
      href={contact.whatsapp.href}
      target="_blank"
      rel="noopener noreferrer"
      aria-label="Chatea con nosotros por WhatsApp"
      className="fixed bottom-5 right-5 z-40 flex h-12 items-center gap-2 rounded-full bg-[#25D366] px-4 text-white shadow-elevated transition-transform duration-200 hover:scale-[1.04] focus-visible:scale-[1.04] print:hidden"
    >
      <WhatsAppIcon className="h-5 w-5" />
      <span className="hidden text-body-sm font-semibold sm:inline">
        WhatsApp
      </span>
    </a>
  );
}
