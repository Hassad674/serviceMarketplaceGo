"use client"

import { AlertTriangle, Clock } from "lucide-react"
import { useTranslations } from "next-intl"
import { useStripeRequirements } from "../hooks/use-payment-info"
import type { FieldSection, FieldSpec } from "../api/payment-info-api"

export function StripeRequirementsBanner() {
  const t = useTranslations("paymentInfo")
  const { data: reqs } = useStripeRequirements()

  if (!reqs?.has_requirements) return null

  const urgentFields = collectFieldsByUrgency(reqs.sections ?? [], ["past_due", "currently_due"], t)
  const eventualFields = collectFieldsByUrgency(reqs.sections ?? [], ["eventually_due"], t)

  const deadlineDate = reqs.current_deadline
    ? new Date(reqs.current_deadline * 1000).toLocaleDateString("fr-FR", {
        day: "numeric",
        month: "long",
        year: "numeric",
      })
    : null

  return (
    <div className="space-y-3">
      {urgentFields.length > 0 && (
        <div className="rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-500/30 dark:bg-red-500/10">
          <div className="flex items-start gap-3">
            <AlertTriangle className="h-5 w-5 shrink-0 text-red-600 dark:text-red-400 mt-0.5" strokeWidth={1.5} />
            <div className="flex-1 space-y-2">
              <p className="text-sm font-semibold text-red-700 dark:text-red-300">
                {t("requirementsTitle")}
              </p>
              <p className="text-xs text-red-600/80 dark:text-red-400/80">
                {t("requirementsDesc")}
              </p>
              {deadlineDate && (
                <div className="flex items-center gap-1.5 text-xs font-medium text-red-700 dark:text-red-300">
                  <Clock className="h-3.5 w-3.5" strokeWidth={1.5} />
                  <span>Date limite : {deadlineDate}</span>
                </div>
              )}
              <ul className="space-y-1">
                {urgentFields.map((name, i) => (
                  <li key={i} className="text-xs text-red-700 dark:text-red-300">
                    {"\u2022"} {name}
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>
      )}

      {eventualFields.length > 0 && (
        <div className="rounded-xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-500/30 dark:bg-amber-500/10">
          <div className="flex items-start gap-3">
            <AlertTriangle className="h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400 mt-0.5" strokeWidth={1.5} />
            <div className="flex-1 space-y-2">
              <p className="text-sm font-semibold text-amber-700 dark:text-amber-300">
                {t("requirementsEventualTitle")}
              </p>
              <p className="text-xs text-amber-600/80 dark:text-amber-400/80">
                {t("requirementsEventualDesc")}
              </p>
              <ul className="space-y-1">
                {eventualFields.map((name, i) => (
                  <li key={i} className="text-xs text-amber-700 dark:text-amber-300">
                    {"\u2022"} {name}
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

/** Extract human-readable field names from sections matching the given urgency levels. */
function collectFieldsByUrgency(
  sections: FieldSection[],
  urgencies: string[],
  t: (key: string) => string,
): string[] {
  const names: string[] = []
  for (const section of sections) {
    for (const field of section.fields) {
      const fieldUrgency = (field as FieldSpec).urgency ?? "currently_due"
      if (urgencies.includes(fieldUrgency)) {
        names.push(safeTranslate(t, field.label_key))
      }
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
