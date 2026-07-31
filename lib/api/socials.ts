import "server-only";
import { apiFetch } from "./client";
import type { SocialLink } from "@/lib/types";

/**
 * Fetches social media links for the footer / contact section.
 * Returns an empty array on failure so the layout never breaks, but logs
 * the error so it's visible during development.
 */
export async function getSocialLinks(): Promise<SocialLink[]> {
  try {
    const res = await apiFetch<{ links: SocialLink[] } | SocialLink[]>(
      "/api/socials",
      { next: { revalidate: 600, tags: ["socials"] } },
    );
    return Array.isArray(res) ? res : (res.links ?? []);
  } catch (err) {
    if (process.env.NODE_ENV !== "production") {
      console.warn("[api/socials] failed to load social links:", err);
    }
    return [];
  }
}
