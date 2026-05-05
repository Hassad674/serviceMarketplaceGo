"use client"

import { useState } from "react"
import { Mail, Building2, Loader2, AlertTriangle, CheckCircle2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { ApiError } from "@/shared/lib/api-client"
import { useInvitationPreview, useAcceptInvitation } from "../hooks/use-team"

// Soleil v2 — Public landing page reached from the invitation email link.
// Editorial Fraunces title, ivoire surface, corail-soft icon chip,
// password form with inline validation. On success, hard-redirect to
// /team so TanStack cache + cookies re-bootstrap.

const MIN_PASSWORD_LENGTH = 8
const PASSWORD_RULE = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)(?=.*[^A-Za-z0-9]).{8,}$/

type AcceptInvitationPageProps = {
  token: string
}

export function AcceptInvitationPage({ token }: AcceptInvitationPageProps) {
  const t = useTranslations("acceptInvitation")
  const {
    data: preview,
    isLoading: previewLoading,
    error: previewError,
  } = useInvitationPreview(token)
  const acceptMutation = useAcceptInvitation()

  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [touched, setTouched] = useState(false)

  if (previewLoading) {
    return (
      <div className="w-full max-w-md rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-8 shadow-[var(--shadow-card)]">
        <div className="flex items-center gap-3 font-serif text-[13.5px] italic text-[var(--muted-foreground)]">
          <Loader2 className="h-4 w-4 animate-spin" />
          {t("loading")}
        </div>
      </div>
    )
  }

  if (previewError || !preview) {
    const status =
      previewError instanceof ApiError ? previewError.status : undefined
    return (
      <div className="w-full max-w-md rounded-2xl border border-[var(--primary-soft)] bg-[var(--surface)] p-8 shadow-[var(--shadow-card)]">
        <div className="flex items-start gap-3">
          <span
            aria-hidden="true"
            className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[var(--primary-soft)] text-[var(--primary-deep)]"
          >
            <AlertTriangle className="h-5 w-5" strokeWidth={1.8} />
          </span>
          <div>
            <h1 className="font-serif text-[20px] font-medium tracking-[-0.01em] text-[var(--foreground)]">
              {status === 404 || status === 410
                ? t("expiredTitle")
                : t("genericErrorTitle")}
            </h1>
            <p className="mt-1 font-serif text-[13.5px] italic text-[var(--muted-foreground)]">
              {status === 404 || status === 410
                ? t("expiredDescription")
                : t("genericErrorDescription")}
            </p>
          </div>
        </div>
      </div>
    )
  }

  const passwordTooShort = touched && password.length < MIN_PASSWORD_LENGTH
  const passwordWeak =
    touched && password.length >= MIN_PASSWORD_LENGTH && !PASSWORD_RULE.test(password)
  const passwordMismatch =
    touched && confirmPassword.length > 0 && password !== confirmPassword
  const hasValidationError = passwordTooShort || passwordWeak || passwordMismatch
  const canSubmit =
    password.length >= MIN_PASSWORD_LENGTH &&
    PASSWORD_RULE.test(password) &&
    password === confirmPassword &&
    !acceptMutation.isPending

  const acceptError =
    acceptMutation.error instanceof ApiError ? acceptMutation.error : null

  function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setTouched(true)
    if (!canSubmit) return
    acceptMutation.mutate(
      { token, password },
      {
        onSuccess: () => {
          window.location.href = "/team"
        },
      },
    )
  }

  const orgLabel =
    preview.organization_type === "agency"
      ? t("orgTypeAgency")
      : t("orgTypeEnterprise")
  const displayName =
    `${preview.first_name} ${preview.last_name}`.trim() || preview.email

  const inputBase =
    "w-full rounded-xl border bg-[var(--surface)] px-3 py-2.5 text-[14px] text-[var(--foreground)] placeholder:text-[var(--subtle-foreground)] focus:outline-none focus:ring-2"
  const inputOk =
    "border-[var(--border)] focus:border-[var(--primary)] focus:ring-[var(--primary-soft)]"
  const inputErr = "border-[var(--primary-deep)] focus:ring-[var(--primary-soft)]"

  return (
    <div className="w-full max-w-md rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-8 shadow-[var(--shadow-card)]">
      <div className="flex items-start gap-3">
        <span
          aria-hidden="true"
          className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-[var(--primary-soft)] text-[var(--primary)]"
        >
          <Building2 className="h-5 w-5" strokeWidth={1.8} />
        </span>
        <div className="flex-1">
          <p className="font-mono text-[11px] font-bold uppercase tracking-[0.08em] text-[var(--primary)]">
            {t("eyebrow")}
          </p>
          <h1 className="mt-1 font-serif text-[24px] font-medium tracking-[-0.01em] text-[var(--foreground)]">
            {t("title")}
          </h1>
          <p className="mt-1 font-serif text-[13.5px] italic text-[var(--muted-foreground)]">
            {t("invitedAs", {
              name: displayName,
              role: t(`roles.${preview.role}`),
              orgType: orgLabel,
            })}
          </p>
          {preview.title && (
            <p className="mt-1 font-mono text-[11px] uppercase tracking-[0.05em] text-[var(--subtle-foreground)]">
              {t("invitedTitle", { title: preview.title })}
            </p>
          )}
        </div>
      </div>

      <div className="mt-6 rounded-xl border border-[var(--border)] bg-[var(--background)] p-3">
        <div className="flex items-center gap-2 text-[13px] text-[var(--foreground)]">
          <Mail
            className="h-4 w-4 text-[var(--muted-foreground)]"
            strokeWidth={1.8}
          />
          {preview.email}
        </div>
      </div>

      <form className="mt-6 space-y-4" onSubmit={handleSubmit}>
        <div>
          <label
            htmlFor="accept-password"
            className="mb-1.5 block text-[12px] font-semibold uppercase tracking-[0.04em] text-[var(--muted-foreground)]"
          >
            {t("passwordLabel")}
          </label>
          <input
            id="accept-password"
            type="password"
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder={t("passwordPlaceholder")}
            className={`${inputBase} ${
              passwordTooShort || passwordWeak ? inputErr : inputOk
            }`}
          />
          {passwordTooShort && (
            <p className="mt-1 text-[12px] text-[var(--primary-deep)]">
              {t("errors.passwordTooShort", { min: MIN_PASSWORD_LENGTH })}
            </p>
          )}
          {passwordWeak && !passwordTooShort && (
            <p className="mt-1 text-[12px] text-[var(--primary-deep)]">
              {t("errors.passwordWeak")}
            </p>
          )}
        </div>

        <div>
          <label
            htmlFor="accept-confirm"
            className="mb-1.5 block text-[12px] font-semibold uppercase tracking-[0.04em] text-[var(--muted-foreground)]"
          >
            {t("confirmPasswordLabel")}
          </label>
          <input
            id="accept-confirm"
            type="password"
            autoComplete="new-password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            className={`${inputBase} ${passwordMismatch ? inputErr : inputOk}`}
          />
          {passwordMismatch && (
            <p className="mt-1 text-[12px] text-[var(--primary-deep)]">
              {t("errors.passwordMismatch")}
            </p>
          )}
        </div>

        {acceptError && !hasValidationError && (
          <div className="rounded-xl border border-[var(--primary-soft)] bg-[var(--primary-soft)] p-3 text-[13px] text-[var(--primary-deep)]">
            {acceptError.status === 409
              ? t("errors.emailAlreadyTaken")
              : acceptError.status === 410
                ? t("expiredDescription")
                : t("errors.genericAccept")}
          </div>
        )}

        <button
          type="submit"
          disabled={!canSubmit}
          className="inline-flex w-full items-center justify-center gap-2 rounded-full bg-[var(--primary)] px-4 py-2.5 text-[13px] font-semibold text-[var(--primary-foreground)] shadow-[var(--shadow-message)] transition-colors hover:bg-[var(--primary-deep)] disabled:opacity-50"
        >
          {acceptMutation.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <CheckCircle2 className="h-4 w-4" strokeWidth={2} />
          )}
          {acceptMutation.isPending ? t("submitting") : t("acceptButton")}
        </button>

        <p className="text-center font-mono text-[11px] uppercase tracking-[0.05em] text-[var(--subtle-foreground)]">
          {t("expiresOn", {
            date: new Date(preview.expires_at).toLocaleDateString(),
          })}
        </p>
      </form>
    </div>
  )
}
