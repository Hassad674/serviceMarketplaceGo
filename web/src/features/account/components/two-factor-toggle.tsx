"use client"

import { useState } from "react"
import { useTranslations } from "next-intl"
import { toast } from "sonner"
import { ShieldCheck, ShieldOff } from "lucide-react"

import { Input } from "@/shared/components/ui/input"
import { Button } from "@/shared/components/ui/button"
import { ApiError } from "@/shared/lib/api-client"
import {
  requestEnableTwoFactor,
  confirmEnableTwoFactor,
  disableTwoFactor,
} from "../api/two-factor-api"

/**
 * TwoFactorToggle — Sécurité tab card that turns email 2FA on or off.
 *
 * The current state is owned locally because /auth/me does not yet
 * surface `two_factor_email_enabled`. We accept the prop as the
 * initial value and update it optimistically on each successful
 * mutation. On the next page load the parent re-reads the source of
 * truth (when the backend ships the field) — until then the local
 * state is the only thing the user can rely on.
 *
 * Enable flow is two-step:
 *   1. POST /me/two-factor/enable (no body) → email challenge,
 *      challenge_id returned in 202.
 *   2. POST /me/two-factor/enable { code } → flag flips, 200.
 *
 * Disable flow is one step but requires the current password as
 * defense-in-depth — same posture as ChangePassword.
 */

type Mode = "idle" | "enable-confirm" | "disable-confirm"

const ENABLE_ERROR_KEYS: Record<string, string> = {
  no_challenge: "errors.noChallenge",
  challenge_expired: "errors.challengeExpired",
  invalid_code: "errors.invalidCode",
  too_many_attempts: "errors.tooManyAttempts",
  feature_unavailable: "errors.featureUnavailable",
}

const DISABLE_ERROR_KEYS: Record<string, string> = {
  invalid_credentials: "errors.invalidCredentials",
  feature_unavailable: "errors.featureUnavailable",
}

export type TwoFactorToggleProps = {
  /**
   * Whether 2FA is currently enabled for the user. The flag is not
   * surfaced by /auth/me yet — until then the parent passes `false`
   * by default and we refresh the local copy on every successful
   * mutation.
   */
  initialEnabled?: boolean
}

export function TwoFactorToggle({ initialEnabled = false }: TwoFactorToggleProps) {
  const t = useTranslations("twoFactor")
  const tAccount = useTranslations("account")
  const [enabled, setEnabled] = useState(initialEnabled)
  const [mode, setMode] = useState<Mode>("idle")
  const [busy, setBusy] = useState(false)
  const [code, setCode] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState<string | null>(null)

  function reset() {
    setMode("idle")
    setBusy(false)
    setCode("")
    setPassword("")
    setError(null)
  }

  function mapEnableError(err: unknown): string {
    if (err instanceof ApiError) {
      const key = ENABLE_ERROR_KEYS[err.code] ?? "errors.generic"
      return t(key)
    }
    return t("errors.generic")
  }

  function mapDisableError(err: unknown): string {
    if (err instanceof ApiError) {
      const key = DISABLE_ERROR_KEYS[err.code] ?? "errors.generic"
      return t(key)
    }
    return t("errors.generic")
  }

  async function startEnable() {
    setBusy(true)
    setError(null)
    try {
      await requestEnableTwoFactor()
      setMode("enable-confirm")
    } catch (err) {
      setError(mapEnableError(err))
    } finally {
      setBusy(false)
    }
  }

  async function confirmEnable(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (busy) return
    const trimmed = code.trim()
    if (trimmed.length !== 6) {
      setError(t("errors.codeLength"))
      return
    }
    setBusy(true)
    setError(null)
    try {
      await confirmEnableTwoFactor(trimmed)
      setEnabled(true)
      toast.success(t("toasts.enabled"))
      reset()
    } catch (err) {
      setError(mapEnableError(err))
      setCode("")
    } finally {
      setBusy(false)
    }
  }

  async function startDisable() {
    setMode("disable-confirm")
  }

  async function confirmDisable(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (busy) return
    if (!password) {
      setError(t("errors.passwordRequired"))
      return
    }
    setBusy(true)
    setError(null)
    try {
      await disableTwoFactor({ current_password: password })
      setEnabled(false)
      toast.success(t("toasts.disabled"))
      reset()
    } catch (err) {
      setError(mapDisableError(err))
      setPassword("")
    } finally {
      setBusy(false)
    }
  }

  const Icon = enabled ? ShieldCheck : ShieldOff

  return (
    <section
      aria-labelledby="two-factor-heading"
      className="rounded-2xl border border-border bg-card p-6 shadow-[var(--shadow-card)]"
    >
      <div className="flex items-start gap-3">
        <div
          className={[
            "flex h-10 w-10 shrink-0 items-center justify-center rounded-xl",
            enabled ? "bg-success/10 text-success" : "bg-primary-soft text-[var(--primary-deep)]",
          ].join(" ")}
          aria-hidden="true"
        >
          <Icon className="h-5 w-5" strokeWidth={1.6} />
        </div>
        <div className="min-w-0 flex-1">
          <h3
            id="two-factor-heading"
            className="font-serif text-[18px] font-semibold tracking-[-0.01em] text-foreground"
          >
            {t("toggleTitle")}
          </h3>
          <p className="mt-1 text-sm text-muted-foreground">
            {enabled ? t("toggleDescOn") : t("toggleDescOff")}
          </p>
        </div>
        {mode === "idle" && (
          <Button
            type="button"
            variant={enabled ? "outline" : "primary"}
            size="md"
            disabled={busy}
            onClick={enabled ? startDisable : startEnable}
            aria-label={enabled ? t("disableCta") : t("enableCta")}
          >
            {busy
              ? tAccount("saving")
              : enabled
                ? t("disableCta")
                : t("enableCta")}
          </Button>
        )}
      </div>

      {error && mode !== "idle" && (
        <p
          role="alert"
          className="mt-4 rounded-xl border border-destructive/30 bg-primary-soft/30 p-3 text-sm text-destructive"
        >
          {error}
        </p>
      )}

      {mode === "enable-confirm" && (
        <form onSubmit={confirmEnable} className="mt-5 space-y-4" noValidate>
          <p className="text-sm text-foreground">{t("enablePrompt")}</p>
          <div className="space-y-1.5">
            <label
              htmlFor="enable-2fa-code"
              className="block text-sm font-medium text-foreground"
            >
              {t("codeLabel")}
            </label>
            <Input
              id="enable-2fa-code"
              type="text"
              inputMode="numeric"
              autoComplete="one-time-code"
              maxLength={6}
              pattern="[0-9]*"
              autoFocus
              value={code}
              onChange={(e) => setCode(e.target.value.replace(/[^0-9]/g, ""))}
              disabled={busy}
              placeholder={t("codePlaceholder")}
              className="font-mono text-center text-[16px] tracking-[0.4em]"
            />
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              type="submit"
              variant="primary"
              size="md"
              disabled={busy || code.length !== 6}
            >
              {busy ? tAccount("saving") : t("confirmEnableCta")}
            </Button>
            <Button
              type="button"
              variant="outline"
              size="md"
              disabled={busy}
              onClick={reset}
            >
              {t("cancel")}
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="md"
              disabled={busy}
              onClick={startEnable}
            >
              {t("resend")}
            </Button>
          </div>
        </form>
      )}

      {mode === "disable-confirm" && (
        <form onSubmit={confirmDisable} className="mt-5 space-y-4" noValidate>
          <p className="text-sm text-foreground">{t("disablePrompt")}</p>
          <div className="space-y-1.5">
            <label
              htmlFor="disable-2fa-password"
              className="block text-sm font-medium text-foreground"
            >
              {t("currentPasswordLabel")}
            </label>
            <Input
              id="disable-2fa-password"
              type="password"
              autoComplete="current-password"
              autoFocus
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={busy}
              placeholder="••••••••"
            />
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              type="submit"
              variant="primary"
              size="md"
              disabled={busy || !password}
            >
              {busy ? tAccount("saving") : t("confirmDisableCta")}
            </Button>
            <Button
              type="button"
              variant="outline"
              size="md"
              disabled={busy}
              onClick={reset}
            >
              {t("cancel")}
            </Button>
          </div>
        </form>
      )}
    </section>
  )
}
