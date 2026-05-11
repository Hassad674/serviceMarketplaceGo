"use client"

import { useTranslations } from "next-intl"
import { TwoFactorToggle } from "./two-factor-toggle"
import { SessionsList } from "./sessions-list"
import { useUser } from "@/shared/hooks/use-user"

/**
 * SecuritySettings — Soleil v2 panel for the /account?section=security
 * tab.
 *
 * Surfaces two security primitives:
 *   1. The email-2FA toggle (B.6).
 *   2. The Malt-style active-sessions list (SEC-SESSIONS): one row per
 *      user_sessions audit row, with device label, localisation,
 *      timestamp précis and a "Révoquer" button. The previous
 *      "Activité récente" feed (which read audit_logs) is replaced
 *      because the session view answers the user's real question:
 *      "what's currently logged into my account?" — the audit feed
 *      only listed past events without revocation.
 */
export function SecuritySettings() {
  const t = useTranslations("account.security")
  const { data: user } = useUser()

  return (
    <div className="space-y-6">
      <header>
        <h2 className="font-serif text-[22px] font-semibold tracking-[-0.01em] text-foreground">
          {t("title")}
        </h2>
        <p className="mt-1.5 text-sm text-muted-foreground">{t("subtitle")}</p>
      </header>

      {/* B.6 — Email 2FA toggle. The flag is not exposed on /auth/me yet
          so we default to false; on the next page load the parent will
          re-read once the field is wired (flagged for follow-up). */}
      <TwoFactorToggle initialEnabled={user?.two_factor_email_enabled ?? false} />

      <SessionsList />
    </div>
  )
}
