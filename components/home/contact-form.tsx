"use client";

import { useActionState } from "react";
import { CheckCircle2, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  submitContactForm,
  type ContactFormState,
} from "@/app/(site)/actions";
import { contactSection } from "@/lib/copy/home";

const initialState: ContactFormState = { ok: false };

/**
 * Contact form. Posts through a Next.js server action (works without JS),
 * which validates with zod and forwards to the Go backend.
 */
export function ContactForm() {
  const [state, formAction, pending] = useActionState(
    submitContactForm,
    initialState,
  );

  if (state.ok) {
    return (
      <div className="flex flex-col items-center gap-4 rounded-lg bg-card/95 p-8 text-center shadow-card backdrop-blur-sm">
        <CheckCircle2 className="h-10 w-10 text-accent" aria-hidden />
        <p className="text-body-lg font-medium text-foreground">
          {contactSection.form.success}
        </p>
        <p className="text-body-sm text-muted-foreground">
          {contactSection.form.subtitle}
        </p>
      </div>
    );
  }

  return (
    <form
      action={formAction}
      className="rounded-lg bg-card/95 p-6 shadow-card backdrop-blur-sm md:p-8"
      noValidate={false}
    >
      <h3 className="font-display text-display-sm text-foreground">
        {contactSection.form.title}
      </h3>
      <p className="mt-2 text-body-sm text-muted-foreground">
        {contactSection.form.subtitle}
      </p>

      <div className="mt-6 space-y-5">
        <div className="space-y-2">
          <Label htmlFor="contact-name">
            Nombre completo <span aria-hidden className="text-destructive">*</span>
          </Label>
          <Input
            id="contact-name"
            name="name"
            type="text"
            autoComplete="name"
            required
            placeholder="Tu nombre completo"
            aria-invalid={!!state.fieldErrors?.name}
            aria-describedby={state.fieldErrors?.name ? "contact-name-error" : undefined}
          />
          {state.fieldErrors?.name && (
            <p id="contact-name-error" className="text-body-sm text-destructive">
              {state.fieldErrors.name}
            </p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="contact-phone">
            Teléfono <span aria-hidden className="text-destructive">*</span>
          </Label>
          <Input
            id="contact-phone"
            name="phone"
            type="tel"
            autoComplete="tel"
            required
            placeholder="Tu número de teléfono"
            aria-invalid={!!state.fieldErrors?.phone}
            aria-describedby={state.fieldErrors?.phone ? "contact-phone-error" : undefined}
          />
          {state.fieldErrors?.phone && (
            <p id="contact-phone-error" className="text-body-sm text-destructive">
              {state.fieldErrors.phone}
            </p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="contact-eventDate">
            Fecha del evento{" "}
            <span className="text-body-sm font-normal text-muted-foreground">(opcional)</span>
          </Label>
          <Input
            id="contact-eventDate"
            name="eventDate"
            type="date"
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="contact-message">
            Mensaje{" "}
            <span className="text-body-sm font-normal text-muted-foreground">(opcional)</span>
          </Label>
          <Textarea
            id="contact-message"
            name="message"
            rows={3}
            placeholder="Cuéntanos un poco sobre tu evento"
          />
        </div>
      </div>

      {state.formError && (
        <p role="alert" className="mt-4 text-body-sm text-destructive">
          {state.formError}
        </p>
      )}

      <Button
        type="submit"
        disabled={pending}
        className="mt-6 w-full bg-accent text-accent-foreground hover:bg-accent/90"
      >
        {pending ? (
          <>
            <Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden />
            Enviando…
          </>
        ) : (
          contactSection.form.submit
        )}
      </Button>
    </form>
  );
}
