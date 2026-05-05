"use client"

import { useTranslations } from "next-intl"

import { Input } from "@/shared/components/ui/input"

/**
 * PasswordSettings — Soleil v2 disabled-state card. The actual password
 * change endpoint isn't shipped yet; the form mirrors what it will look
 * like and surfaces the "comingSoon" pill on submit.
 */
export function PasswordSettings() {
  const t = useTranslations("account")

  const fields = [
    { id: "account-current-password", labelKey: "currentPassword" },
    { id: "account-new-password", labelKey: "newPassword" },
    { id: "account-confirm-password", labelKey: "confirmPassword" },
  ]

  return (
    <div className="space-y-6">
      <header>
        <h2 className="font-serif text-[22px] font-semibold tracking-[-0.01em] text-foreground">
          {t("passwordTitle")}
        </h2>
        <p className="mt-1.5 text-sm text-muted-foreground">
          {t("passwordDesc")}
        </p>
      </header>

      <div className="rounded-2xl border border-border bg-card p-6 shadow-[var(--shadow-card)]">
        <div className="space-y-5">
          {fields.map((field) => (
            <div key={field.id}>
              <label
                htmlFor={field.id}
                className="block text-sm font-medium text-foreground"
              >
                {t(field.labelKey)}
              </label>
              <Input
                id={field.id}
                type="password"
                disabled
                placeholder="••••••••"
                className="mt-1.5"
              />
            </div>
          ))}
        </div>

        <div className="mt-5">
          <span className="inline-flex items-center rounded-full bg-primary-soft px-3 py-1 text-xs font-medium text-[var(--primary-deep)]">
            {t("comingSoon")}
          </span>
        </div>
      </div>
    </div>
  )
}
