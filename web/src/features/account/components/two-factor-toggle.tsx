"use client"

import { useEffect, useState } from "react"
import { useTranslations } from "next-intl"
import { useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { ShieldCheck, ShieldOff } from "lucide-react"

import { Input } from "@/shared/components/ui/input"
import { Button } from "@/shared/components/ui/button"
import { ApiError } from "@/shared/lib/api-client"
import { useUser } from "@/shared/hooks/use-user"
import {
  requestEnableTwoFactor,
  confirmEnableTwoFactor,
  disableTwoFactor,
} from "../api/two-factor-api"

/**
 * TwoFactorToggle — Sécurité tab card that turns email 2FA on or off.
 *
 * The current state derives from `useUser()` (TanStack Query session
 * cache) so the toggle renders the correct initial state on first
 * paint — including after a page reload, which previously defaulted
 * the switch to OFF and left users unable to disable an already-enabled
 * 2FA (FIX-2FA bug). Each successful enable/disable mutation
 * invalidates the ["session"] query so the next /auth/me call refreshes
 * the cache; an `useEffect` syncs the local "enabled" state when the
 * cache value changes (e.g. between login and the first paint).
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
   * Optional initial-paint value, used by the unit tests to seed
   * the toggle state without a full QueryClient setup. In real
   * usage the source of truth is `useUser().two_factor_email_enabled`
   * — the prop only acts as a tie-breaker before the first session
   * payload lands (typically during SSR / initial mount on a public
   * page that does NOT render this component, so the impact is nil).
   */
  initialEnabled?: boolean
}

export function TwoFactorToggle({ initialEnabled = false }: TwoFactorToggleProps) {
  const t = useTranslations("twoFactor")
  const tAccount = useTranslations("account")
  const queryClient = useQueryClient()
  const { data: user } = useUser()
  // FIX-2FA: derive initial state from the session cache, fall back
  // to the prop when the cache is still empty (first paint before
  // /auth/me has resolved). On every subsequent render the useEffect
  // below re-syncs from the cache so a fresh /me refresh after login
  // (or after invalidation post-mutation) is honored.
  const cachedEnabled = user?.two_factor_email_enabled ?? initialEnabled
  const [enabled, setEnabled] = useState<boolean>(cachedEnabled)
  const [mode, setMode] = useState<Mode>("idle")
  const [busy, setBusy] = useState(false)
  const [code, setCode] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState<string | null>(null)

  // Keep the local toggle in sync with the session cache when the
  // user object lands or refreshes. Without this, the toggle stays
  // at its mount-time value (false) even after /auth/me reports the
  // flag is true, which is the FIX-2FA bug we are repairing.
  useEffect(() => {
    if (user?.two_factor_email_enabled !== undefined) {
      setEnabled(user.two_factor_email_enabled)
    }
  }, [user?.two_factor_email_enabled])

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
      // FIX-2FA: invalidate the session cache so any other consumer
      // of useUser() (sidebar badge, account page, etc.) reads the
      // new flag value on its next render instead of showing stale
      // "off". The optimistic setEnabled(true) above keeps THIS card
      // responsive; the invalidate fans the change out to everyone.
      void queryClient.invalidateQueries({ queryKey: ["session"] })
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
      // FIX-2FA: see confirmEnable above — invalidate the session so
      // every consumer of useUser() refreshes its 2FA badge.
      void queryClient.invalidateQueries({ queryKey: ["session"] })
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
