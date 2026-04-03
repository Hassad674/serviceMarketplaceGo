"use client"

import { AlertTriangle } from "lucide-react"
import { useTranslations } from "next-intl"
import { useStripeRequirements } from "../hooks/use-payment-info"
import type { FieldSection } from "../api/payment-info-api"

export function StripeRequirementsBanner() {
  const t = useTranslations("paymentInfo")
  const { data: reqs } = useStripeRequirements()

  if (!reqs?.has_requirements) return null

  const fieldNames = collectFieldNames(reqs.sections ?? [], t)

  return (
    <div className="rounded-xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-500/30 dark:bg-amber-500/10">
      <div className="flex items-start gap-3">
        <AlertTriangle className="h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400 mt-0.5" strokeWidth={1.5} />
        <div className="flex-1 space-y-2">
          <p className="text-sm font-semibold text-amber-700 dark:text-amber-300">
            {t("requirementsTitle")}
          </p>
          <p className="text-xs text-amber-600/80 dark:text-amber-400/80">
            {t("requirementsDesc")}
          </p>
          {fieldNames.length > 0 && (
            <ul className="space-y-1">
              {fieldNames.map((name, i) => (
                <li key={i} className="text-xs text-amber-700 dark:text-amber-300">
                  {"\u2022"} {name}
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}

/** Extract human-readable field names from requirement sections. */
function collectFieldNames(
  sections: FieldSection[],
  t: (key: string) => string,
): string[] {
  const names: string[] = []
  for (const section of sections) {
    for (const field of section.fields) {
      names.push(safeTranslate(t, field.label_key))
    }
  }
  return names
}

/** Safely translate a key, falling back to a humanized version. */
function safeTranslate(t: (key: string) => string, key: string): string {
  try {
    const result = t(key)
    if (result.startsWith("paymentInfo.") || result === key) {
      return humanizeKey(key)
    }
    return result
  } catch {
    return humanizeKey(key)
  }
}

function humanizeKey(key: string): string {
  return key
    .replace(/([A-Z])/g, " $1")
    .replace(/_/g, " ")
    .replace(/^\s+/, "")
    .replace(/\b\w/g, (c) => c.toUpperCase())
    .trim()
}
