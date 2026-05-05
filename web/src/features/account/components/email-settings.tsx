"use client"

import { useTranslations } from "next-intl"
import { useUser } from "@/shared/hooks/use-user"

import { Input } from "@/shared/components/ui/input"

/**
 * EmailSettings — Soleil v2 read-only card showing the current account
 * email plus a placeholder input for the upcoming "change email" flow
 * (still backend-blocked → "comingSoon" pill).
 */
export function EmailSettings() {
  const t = useTranslations("account")
  const { data: user } = useUser()

  return (
    <div className="space-y-6">
      <header>
        <h2 className="font-serif text-[22px] font-semibold tracking-[-0.01em] text-foreground">
          {t("emailTitle")}
        </h2>
        <p className="mt-1.5 text-sm text-muted-foreground">
          {t("emailDesc")}
        </p>
      </header>

      <div className="rounded-2xl border border-border bg-card p-6 shadow-[var(--shadow-card)]">
        <div>
          <span className="text-[11px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
            {t("currentEmail")}
          </span>
          <p className="mt-1.5 font-mono text-[15px] text-foreground">
            {user?.email || "—"}
          </p>
        </div>

        <div className="mt-7">
          <label
            htmlFor="account-new-email"
            className="block text-sm font-medium text-foreground"
          >
            {t("newEmail")}
          </label>
          <Input
            id="account-new-email"
            type="email"
            disabled
            placeholder="new@email.com"
            className="mt-1.5"
          />
          <span className="mt-3 inline-flex items-center rounded-full bg-primary-soft px-3 py-1 text-xs font-medium text-[var(--primary-deep)]">
            {t("comingSoon")}
          </span>
        </div>
      </div>
    </div>
  )
}
