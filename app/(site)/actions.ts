"use server";

import { z } from "zod";
import { API_BASE_URL } from "@/lib/api/client";

const contactSchema = z.object({
  name: z
    .string()
    .trim()
    .min(2, "El nombre debe tener al menos 2 caracteres"),
  phone: z
    .string()
    .trim()
    .min(10, "El teléfono debe tener al menos 10 dígitos"),
  eventDate: z.string().trim().optional(),
  message: z.string().trim().max(2000).optional(),
});

export type ContactFormState = {
  ok: boolean;
  /** Field-level errors keyed by input name. */
  fieldErrors?: Partial<Record<"name" | "phone" | "eventDate" | "message", string>>;
  /** Form-level error message (server failure). */
  formError?: string;
};

export async function submitContactForm(
  _prev: ContactFormState,
  formData: FormData,
): Promise<ContactFormState> {
  const parsed = contactSchema.safeParse({
    name: formData.get("name"),
    phone: formData.get("phone"),
    eventDate: formData.get("eventDate") || undefined,
    message: formData.get("message") || undefined,
  });

  if (!parsed.success) {
    const fieldErrors: ContactFormState["fieldErrors"] = {};
    for (const issue of parsed.error.issues) {
      const field = issue.path[0] as keyof NonNullable<ContactFormState["fieldErrors"]>;
      if (!fieldErrors[field]) fieldErrors[field] = issue.message;
    }
    return { ok: false, fieldErrors };
  }

  try {
    const res = await fetch(`${API_BASE_URL}/api/contact-requests`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(parsed.data),
      cache: "no-store",
    });

    if (!res.ok) {
      const body = (await res.json().catch(() => null)) as
        | { error?: string }
        | null;
      return {
        ok: false,
        formError: body?.error ?? "No pudimos enviar tu solicitud. Inténtalo de nuevo.",
      };
    }

    return { ok: true };
  } catch {
    return {
      ok: false,
      formError: "No pudimos enviar tu solicitud. Inténtalo de nuevo.",
    };
  }
}
