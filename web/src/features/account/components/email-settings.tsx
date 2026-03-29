"use client"

import { useTranslations } from "next-intl"
import { useUser } from "@/shared/hooks/use-user"

export function EmailSettings() {
  const t = useTranslations("account")
  const { data: user } = useUser()

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
          {t("emailTitle")}
        </h2>
        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
          {t("emailDesc")}
        </p>
      </div>

      <div className="rounded-xl border border-slate-200 bg-white p-5 dark:border-slate-700 dark:bg-slate-800">
        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
          {t("currentEmail")}
        </label>
        <p className="mt-1 text-sm text-slate-900 dark:text-slate-100">
          {user?.email || "—"}
        </p>

        <div className="mt-6">
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
            {t("newEmail")}
          </label>
          <input
            type="email"
            disabled
            placeholder="new@email.com"
            className="mt-1 block w-full rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-400 dark:border-slate-600 dark:bg-slate-700 dark:text-slate-500"
          />
          <span className="mt-2 inline-flex items-center rounded-md bg-amber-50 px-2 py-1 text-xs font-medium text-amber-700 ring-1 ring-inset ring-amber-600/20 dark:bg-amber-900/20 dark:text-amber-400 dark:ring-amber-400/20">
            {t("comingSoon")}
          </span>
        </div>
      </div>
    </div>
  )
}
