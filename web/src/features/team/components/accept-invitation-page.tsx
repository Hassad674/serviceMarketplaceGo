"use client"

import { useState } from "react"
import { Mail, Building2, Loader2, AlertTriangle, CheckCircle2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { ApiError } from "@/shared/lib/api-client"
import { useInvitationPreview, useAcceptInvitation } from "../hooks/use-team"

// Public landing page reached from the invitation email link.
//
// Flow:
//   1. Extract {token} from the URL.
//   2. Fetch GET /invitations/validate?token=X to preview who's
//      inviting, to which org, and as what role. Also lets the
//      invitee see a 404-ish error early if the link expired or
//      was cancelled.
//   3. Show a password form (with confirmation) — the invitee has
//      no account yet, they're creating one right now.
//   4. On submit, POST /invitations/accept with { token, password }.
//      Backend creates the operator user + membership, sets the
//      session cookie, returns an auth envelope.
//   5. On success, hard-redirect to /team. We could push via next
//      router but a full page reload is cleaner: it forces
//      TanStack Query to reinitialise its ["session"] cache and
//      rehydrates the sidebar/header with the new operator identity.

const MIN_PASSWORD_LENGTH = 8
// Matches the domain password rule on the backend: at least one
// uppercase, one lowercase, one digit, one special character.
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
      <div className="w-full max-w-md rounded-2xl border border-gray-100 dark:border-slate-700 bg-white dark:bg-slate-800 p-8 shadow-sm">
        <div className="flex items-center gap-3 text-sm text-gray-500 dark:text-gray-400">
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
      <div className="w-full max-w-md rounded-2xl border border-rose-200 dark:border-rose-500/30 bg-white dark:bg-slate-800 p-8 shadow-sm">
        <div className="flex items-start gap-3">
          <AlertTriangle className="h-5 w-5 text-rose-500 flex-shrink-0 mt-0.5" />
          <div>
            <h1 className="text-lg font-semibold text-gray-900 dark:text-white">
              {status === 404 || status === 410 ? t("expiredTitle") : t("genericErrorTitle")}
            </h1>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
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
  const passwordWeak = touched && password.length >= MIN_PASSWORD_LENGTH && !PASSWORD_RULE.test(password)
  const passwordMismatch = touched && confirmPassword.length > 0 && password !== confirmPassword
  const hasValidationError = passwordTooShort || passwordWeak || passwordMismatch
  const canSubmit =
    password.length >= MIN_PASSWORD_LENGTH &&
    PASSWORD_RULE.test(password) &&
    password === confirmPassword &&
    !acceptMutation.isPending

  const acceptError = acceptMutation.error instanceof ApiError ? acceptMutation.error : null

  function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setTouched(true)
    if (!canSubmit) return
    acceptMutation.mutate(
      { token, password },
      {
        onSuccess: () => {
          // Hard redirect — the React tree + TanStack cache + cookies
          // need to flip to the new operator session.
          window.location.href = "/team"
        },
      },
    )
  }

  const orgLabel = preview.organization_type === "agency" ? t("orgTypeAgency") : t("orgTypeEnterprise")
  const displayName = `${preview.first_name} ${preview.last_name}`.trim() || preview.email

  return (
    <div className="w-full max-w-md rounded-2xl border border-gray-100 dark:border-slate-700 bg-white dark:bg-slate-800 p-8 shadow-sm">
      <div className="flex items-start gap-3">
        <div className="rounded-xl bg-rose-50 dark:bg-rose-500/10 p-3">
          <Building2 className="h-5 w-5 text-rose-500" />
        </div>
        <div className="flex-1">
          <h1 className="text-xl font-bold text-gray-900 dark:text-white">
            {t("title")}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {t("invitedAs", { name: displayName, role: t(`roles.${preview.role}`), orgType: orgLabel })}
          </p>
          {preview.title && (
            <p className="mt-1 text-xs text-gray-400 dark:text-gray-500">
              {t("invitedTitle", { title: preview.title })}
            </p>
          )}
        </div>
      </div>

      <div className="mt-6 rounded-lg bg-gray-50 dark:bg-slate-900/60 p-3">
        <div className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <Mail className="h-4 w-4 text-gray-400" />
          {preview.email}
        </div>
      </div>

      <form className="mt-6 space-y-4" onSubmit={handleSubmit}>
        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
            {t("passwordLabel")}
          </label>
          <input
            type="password"
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder={t("passwordPlaceholder")}
            className={`w-full rounded-lg border bg-white dark:bg-slate-900 px-3 py-2 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-rose-500/20 ${
              passwordTooShort || passwordWeak
                ? "border-rose-500"
                : "border-gray-200 dark:border-slate-600 focus:border-rose-500"
            }`}
          />
          {passwordTooShort && (
            <p className="mt-1 text-xs text-rose-600 dark:text-rose-400">
              {t("errors.passwordTooShort", { min: MIN_PASSWORD_LENGTH })}
            </p>
          )}
          {passwordWeak && !passwordTooShort && (
            <p className="mt-1 text-xs text-rose-600 dark:text-rose-400">
              {t("errors.passwordWeak")}
            </p>
          )}
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1.5">
            {t("confirmPasswordLabel")}
          </label>
          <input
            type="password"
            autoComplete="new-password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            className={`w-full rounded-lg border bg-white dark:bg-slate-900 px-3 py-2 text-sm text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-rose-500/20 ${
              passwordMismatch
                ? "border-rose-500"
                : "border-gray-200 dark:border-slate-600 focus:border-rose-500"
            }`}
          />
          {passwordMismatch && (
            <p className="mt-1 text-xs text-rose-600 dark:text-rose-400">
              {t("errors.passwordMismatch")}
            </p>
          )}
        </div>

        {acceptError && !hasValidationError && (
          <div className="rounded-lg border border-rose-200 dark:border-rose-500/30 bg-rose-50 dark:bg-rose-500/10 p-3 text-sm text-rose-700 dark:text-rose-300">
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
          className="inline-flex w-full items-center justify-center gap-2 rounded-lg bg-rose-500 px-4 py-2.5 text-sm font-semibold text-white hover:bg-rose-600 disabled:opacity-50"
        >
          {acceptMutation.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <CheckCircle2 className="h-4 w-4" />
          )}
          {acceptMutation.isPending ? t("submitting") : t("acceptButton")}
        </button>

        <p className="text-center text-xs text-gray-400 dark:text-gray-500">
          {t("expiresOn", { date: new Date(preview.expires_at).toLocaleDateString() })}
        </p>
      </form>
    </div>
  )
}
