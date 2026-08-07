import "server-only";

import { contact } from "@/lib/copy/contact";
import { getGoAPIBaseURL } from "@/lib/env";
import type { SocialLink } from "@/lib/types";

const backendTimeoutMilliseconds = 5_000;

export const fallbackSocialLinks: readonly SocialLink[] = [
  { name: "WhatsApp", link: contact.whatsapp },
];

function isSafeExternalURL(value: string): boolean {
  try {
    const url = new URL(value);
    return url.protocol === "https:" || url.protocol === "http:";
  } catch {
    return false;
  }
}

function isSocialLink(value: unknown): value is SocialLink {
  if (!value || typeof value !== "object") {
    return false;
  }

  const link = value as Record<string, unknown>;
  return (
    typeof link.name === "string" &&
    link.name.trim().length > 0 &&
    typeof link.link === "string" &&
    isSafeExternalURL(link.link)
  );
}

function withWhatsAppFallback(links: readonly SocialLink[]): SocialLink[] {
  const hasWhatsApp = links.some((link) =>
    link.name.toLocaleLowerCase("es").includes("whatsapp"),
  );

  return hasWhatsApp ? [...links] : [...links, ...fallbackSocialLinks];
}

export async function getSocialLinks(): Promise<readonly SocialLink[]> {
  try {
    const response = await fetch(`${getGoAPIBaseURL()}/api/socials`, {
      method: "GET",
      cache: "no-store",
      credentials: "omit",
      headers: {
        Accept: "application/json",
      },
      signal: AbortSignal.timeout(backendTimeoutMilliseconds),
    });

    if (!response.ok) {
      return fallbackSocialLinks;
    }

    const body: unknown = await response.json();
    if (!Array.isArray(body) || !body.every(isSocialLink)) {
      return fallbackSocialLinks;
    }

    return withWhatsAppFallback(body);
  } catch {
    return fallbackSocialLinks;
  }
}
