"use client";

import { useEffect, useRef, useState } from "react";

import { submitContactRequest } from "@/lib/api/contact";
import { homeCopy } from "@/lib/copy/home";
import type { ContactRequestFieldErrors } from "@/lib/types";

type SubmissionStatus = "initial" | "submitting" | "success" | "unavailable";

function validateContactFields(
  name: string,
  phone: string,
): ContactRequestFieldErrors {
  const validation = homeCopy.contact.form.validation;
  const errors: { name?: string; phone?: string } = {};
  const trimmedName = name.trim();
  const trimmedPhone = phone.trim();
  const nameLength = Array.from(trimmedName).length;

  if (nameLength === 0) {
    errors.name = validation.nameRequired;
  } else if (nameLength < 2) {
    errors.name = validation.nameTooShort;
  } else if (nameLength > 120) {
    errors.name = validation.nameTooLong;
  }

  if (trimmedPhone.length === 0) {
    errors.phone = validation.phoneRequired;
  } else {
    const digitCount = trimmedPhone.match(/[0-9]/g)?.length ?? 0;
    const hasValidCharacters = /^[0-9 +()\-]+$/.test(trimmedPhone);
    if (
      trimmedPhone.length > 32 ||
      !hasValidCharacters ||
      digitCount < 10 ||
      digitCount > 15
    ) {
      errors.phone = validation.phoneInvalid;
    }
  }

  return errors;
}

export function ContactForm() {
  const [name, setName] = useState("");
  const [phone, setPhone] = useState("");
  const [fieldErrors, setFieldErrors] =
    useState<ContactRequestFieldErrors>({});
  const [status, setStatus] = useState<SubmissionStatus>("initial");
  const nameRef = useRef<HTMLInputElement>(null);
  const phoneRef = useRef<HTMLInputElement>(null);
  const successRef = useRef<HTMLParagraphElement>(null);
  const submissionInProgressRef = useRef(false);
  const abortControllerRef = useRef<AbortController | null>(null);
  const copy = homeCopy.contact.form;
  const isSubmitting = status === "submitting";

  useEffect(() => {
    if (status === "success") {
      successRef.current?.focus();
    }
  }, [status]);

  useEffect(
    () => () => {
      abortControllerRef.current?.abort();
    },
    [],
  );

  const focusFirstInvalidField = (errors: ContactRequestFieldErrors) => {
    if (errors.name) {
      nameRef.current?.focus();
      return;
    }
    if (errors.phone) {
      phoneRef.current?.focus();
    }
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (submissionInProgressRef.current) {
      return;
    }

    const localErrors = validateContactFields(name, phone);
    setFieldErrors(localErrors);
    setStatus("initial");
    if (localErrors.name || localErrors.phone) {
      focusFirstInvalidField(localErrors);
      return;
    }

    submissionInProgressRef.current = true;
    setStatus("submitting");
    const abortController = new AbortController();
    abortControllerRef.current = abortController;

    const result = await submitContactRequest(
      { name: name.trim(), phone: phone.trim() },
      abortController.signal,
    );

    submissionInProgressRef.current = false;
    abortControllerRef.current = null;
    if (abortController.signal.aborted) {
      return;
    }

    if (result.status === "success") {
      setName("");
      setPhone("");
      setFieldErrors({});
      setStatus("success");
      return;
    }

    if (result.status === "validation_error") {
      setFieldErrors(result.fields);
      setStatus("initial");
      focusFirstInvalidField(result.fields);
      return;
    }

    setStatus("unavailable");
  };

  const fieldClassName =
    "mt-2 min-h-12 w-full rounded-md border border-input bg-card px-4 text-foreground shadow-soft placeholder:text-muted-foreground focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 aria-invalid:border-destructive aria-invalid:outline-destructive";

  return (
    <form
      action="/api/contact-requests"
      method="post"
      aria-labelledby="contact-form-title"
      aria-busy={isSubmitting}
      noValidate
      onSubmit={handleSubmit}
      className="rounded-xl border border-primary-foreground/20 bg-card/95 p-6 text-card-foreground shadow-elevated backdrop-blur-sm sm:p-8"
    >
      <h3 id="contact-form-title" className="type-h3 font-medium">
        {copy.title}
      </h3>
      <p className="type-body mt-3 text-muted-foreground">{copy.description}</p>

      <div className="mt-7 space-y-5">
        <div>
          <label htmlFor="contact-name" className="type-small font-semibold">
            {copy.fields.name.label}
          </label>
          <input
            ref={nameRef}
            id="contact-name"
            name="name"
            type="text"
            autoComplete="name"
            required
            value={name}
            onChange={(event) => setName(event.target.value)}
            aria-invalid={fieldErrors.name ? true : undefined}
            aria-describedby={
              fieldErrors.name
                ? "contact-name-help contact-name-error"
                : "contact-name-help"
            }
            className={fieldClassName}
          />
          <p id="contact-name-help" className="type-small mt-2 text-muted-foreground">
            {copy.fields.name.helper}
          </p>
          {fieldErrors.name ? (
            <p id="contact-name-error" className="type-small mt-2 text-destructive">
              {fieldErrors.name}
            </p>
          ) : null}
        </div>

        <div>
          <label htmlFor="contact-phone" className="type-small font-semibold">
            {copy.fields.phone.label}
          </label>
          <input
            ref={phoneRef}
            id="contact-phone"
            name="phone"
            type="tel"
            inputMode="tel"
            autoComplete="tel"
            maxLength={32}
            required
            value={phone}
            onChange={(event) => setPhone(event.target.value)}
            aria-invalid={fieldErrors.phone ? true : undefined}
            aria-describedby={
              fieldErrors.phone
                ? "contact-phone-help contact-phone-error"
                : "contact-phone-help"
            }
            className={fieldClassName}
          />
          <p id="contact-phone-help" className="type-small mt-2 text-muted-foreground">
            {copy.fields.phone.helper}
          </p>
          {fieldErrors.phone ? (
            <p id="contact-phone-error" className="type-small mt-2 text-destructive">
              {fieldErrors.phone}
            </p>
          ) : null}
        </div>
      </div>

      <button
        id="contact-submit-button"
        type="submit"
        disabled={isSubmitting}
        className="type-button mt-7 inline-flex min-h-12 w-full items-center justify-center rounded-md bg-accent px-6 font-semibold uppercase text-accent-foreground shadow-soft hover:bg-secondary focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-accent disabled:cursor-wait disabled:opacity-70 sm:w-auto"
      >
        {isSubmitting ? copy.button.submitting : copy.button.initial}
      </button>

      {status === "unavailable" ? (
        <p role="alert" className="type-small mt-4 text-destructive">
          {copy.unavailableMessage}
        </p>
      ) : null}
      {status === "success" ? (
        <p
          ref={successRef}
          tabIndex={-1}
          aria-live="polite"
          className="type-body mt-4 rounded-md border border-accent/40 bg-accent/10 p-4 font-semibold focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
        >
          {copy.successMessage}
        </p>
      ) : null}

      <noscript>
        <style>{"#contact-submit-button { display: none !important; }"}</style>
        <p className="type-small mt-4 text-muted-foreground">
          {copy.noScriptMessage}
        </p>
      </noscript>
    </form>
  );
}
