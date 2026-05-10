"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { useQueryClient } from "@tanstack/react-query"
import { useRouter } from "@i18n/navigation"
import { ShieldCheck } from "lucide-react"

import { Input } from "@/shared/components/ui/input"
import { Button } from "@/shared/components/ui/button"
import { ApiError } from "@/shared/lib/api-client"
import { verifyTwoFactor } from "@/features/auth/api/two-factor-api"
import {
  login,
  isTwoFactorChallenge,
  AuthApiError,
} from "@/features/auth/api/auth-api"

/**
 * VerifyTwoFactorForm — second step of the login flow when the backend
 * answered `requires_2fa: true`. Mirrors the LoginForm visual rhythm
 * (eyebrow, single big input, primary CTA) but sits inside the same
 * AuthPageShell — so the user never leaves the editorial split layout.
 *
 * Why a controlled <Input> instead of react-hook-form: the contract is
 * a single 6-digit field with a "resend code" link, not worth wiring
 * resolver + zod for. The ApiError mapping is a flat switch on the
 * 2FA error codes shipped by handleTwoFactorError in the backend.
 */

export type VerifyTwoFactorFormProps = {
  userId: string
  challengeId: string
  email: string
  password: string
  /**
   * Called with a fresh challenge id when the backend issues a new
   * code in response to the "resend" link. The parent owns the
   * challenge id so we can pass the freshest one back to /verify-2fa.
   */
  onChallengeRefreshed?: (challengeId: string) => void
}

const ERROR_CODE_KEYS: Record<string, string> = {
  no_challenge: "errors.noChallenge",
  challenge_expired: "errors.challengeExpired",
  invalid_code: "errors.invalidCode",
  too_many_attempts: "errors.tooManyAttempts",
  session_invalid: "errors.sessionInvalid",
}

export function VerifyTwoFactorForm({
  userId,
  challengeId,
  email,
  password,
  onChallengeRefreshed,
}: VerifyTwoFactorFormProps) {
  const t = useTranslations("twoFactor")
  const router = useRouter()
  const queryClient = useQueryClient()
  const [code, setCode] = useState("")
  const [submitting, setSubmitting] = useState(false)
  const [resending, setResending] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [resendNotice, setResendNotice] = useState<string | null>(null)
  const [activeChallengeId, setActiveChallengeId] = useState(challengeId)

  function mapError(err: unknown): string {
    if (err instanceof ApiError) {
      const key = ERROR_CODE_KEYS[err.code] ?? "errors.generic"
      return t(key)
    }
    if (err instanceof AuthApiError) {
      const key = ERROR_CODE_KEYS[err.code] ?? "errors.generic"
      return t(key)
    }
    if (err && typeof err === "object" && "code" in err) {
      const code = (err as { code?: string }).code
      const key = code ? ERROR_CODE_KEYS[code] : undefined
      return t(key ?? "errors.generic")
    }
    return t("errors.generic")
  }

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (submitting) return
    setError(null)
    setResendNotice(null)
    const trimmed = code.trim()
    if (trimmed.length !== 6) {
      setError(t("errors.codeLength"))
      return
    }
    setSubmitting(true)
    try {
      await verifyTwoFactor({
        user_id: userId,
        challenge_id: activeChallengeId,
        code: trimmed,
      })
      // Same post-login dance as LoginForm — bust the stale cached
      // 401 verdict from /auth/me so the dashboard refetches.
      await queryClient.invalidateQueries({ queryKey: ["session"] })
      router.push("/dashboard")
    } catch (err) {
      setError(mapError(err))
      // Wipe the input on a definitive failure so the user has to
      // re-type — protects against muscle-memory paste of the bad
      // code on every retry.
      setCode("")
    } finally {
      setSubmitting(false)
    }
  }

  async function onResend() {
    if (resending) return
    setError(null)
    setResendNotice(null)
    setResending(true)
    try {
      const resp = await login(email, password)
      if (!isTwoFactorChallenge(resp)) {
        // Defensive — a successful non-2FA login here means the flag
        // flipped off between attempts. Push to dashboard.
        await queryClient.invalidateQueries({ queryKey: ["session"] })
        router.push("/dashboard")
        return
      }
      setActiveChallengeId(resp.challenge_id)
      onChallengeRefreshed?.(resp.challenge_id)
      setResendNotice(t("resendSuccess"))
    } catch (err) {
      setError(mapError(err))
    } finally {
      setResending(false)
    }
  }

  return (
    <form onSubmit={onSubmit} className="space-y-4" noValidate>
      <div
        className="flex items-center gap-2 rounded-xl border border-border bg-primary-soft/40 px-4 py-3 text-sm text-foreground"
        role="status"
      >
        <ShieldCheck className="h-5 w-5 flex-shrink-0 text-primary" aria-hidden="true" />
        <span>{t("emailHint", { email })}</span>
      </div>

      {error && (
        <div
          role="alert"
          className="rounded-xl border border-destructive/30 bg-primary-soft/40 p-3 text-sm text-destructive"
        >
          {error}
        </div>
      )}

      {resendNotice && !error && (
        <div
          role="status"
          className="rounded-xl border border-success/30 bg-amber-soft/40 p-3 text-sm text-foreground"
        >
          {resendNotice}
        </div>
      )}

      <div className="space-y-1.5">
        <label
          htmlFor="two-factor-code"
          className="block text-[13px] font-semibold text-foreground"
        >
          {t("codeLabel")}
        </label>
        <Input
          id="two-factor-code"
          name="two-factor-code"
          type="text"
          inputMode="numeric"
          autoComplete="one-time-code"
          autoFocus
          maxLength={6}
          pattern="[0-9]*"
          placeholder={t("codePlaceholder")}
          value={code}
          onChange={(e) => setCode(e.target.value.replace(/[^0-9]/g, ""))}
          disabled={submitting}
          aria-invalid={Boolean(error) || undefined}
          aria-describedby={error ? "two-factor-error" : undefined}
          className={[
            "block w-full rounded-xl border bg-card px-4 py-[13px] text-center font-mono text-[18px] tracking-[0.4em] text-foreground",
            "transition-colors duration-150 placeholder:text-subtle-foreground placeholder:tracking-normal placeholder:font-sans",
            "focus:border-primary focus:outline-none focus:ring-4 focus:ring-primary/15",
            error
              ? "border-destructive focus:ring-destructive/15"
              : "border-border-strong",
          ].join(" ")}
        />
      </div>

      <Button
        variant="primary"
        size="auto"
        type="submit"
        disabled={submitting || code.length !== 6}
        className={[
          "mt-2 w-full rounded-full px-4 py-3.5 text-[14.5px] font-semibold",
          "active:scale-[0.99]",
          "focus:outline-none focus:ring-4 focus:ring-primary/30",
          "disabled:cursor-not-allowed disabled:opacity-60",
        ].join(" ")}
      >
        {submitting ? t("verifying") : t("verifyCta")}
      </Button>

      <div className="text-center text-[13px] text-muted-foreground">
        {t("resendQuestion")}{" "}
        <button
          type="button"
          onClick={onResend}
          disabled={resending || submitting}
          className="font-semibold text-[var(--text-link)] underline-offset-4 transition-colors hover:text-primary hover:underline disabled:cursor-not-allowed disabled:opacity-60"
        >
          {resending ? t("resending") : t("resend")}
        </button>
      </div>
    </form>
  )
}
