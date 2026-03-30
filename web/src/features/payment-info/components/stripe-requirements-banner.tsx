"use client"

import { AlertTriangle, ExternalLink, Loader2 } from "lucide-react"
import { useTranslations, useLocale } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { useStripeRequirements, useCreateAccountLink } from "../hooks/use-payment-info"

export function StripeRequirementsBanner() {
  const t = useTranslations("paymentInfo")
  const locale = useLocale()
  const { data: reqs } = useStripeRequirements(locale)
  const accountLinkMutation = useCreateAccountLink()

  if (!reqs?.has_requirements) return null

  function handleComplete() {
    accountLinkMutation.mutate(undefined, {
      onSuccess: (result) => {
        window.open(result.url, "_blank")
      },
    })
  }

  return (
    <div className="rounded-xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-500/30 dark:bg-amber-500/10">
      <div className="flex items-start gap-3">
        <AlertTriangle className="h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400 mt-0.5" strokeWidth={1.5} />
        <div className="flex-1 space-y-2">
          <p className="text-sm font-semibold text-amber-700 dark:text-amber-300">
            {t("stripeRequirementsTitle")}
          </p>
          <p className="text-xs text-amber-600/80 dark:text-amber-400/80">
            {t("stripeRequirementsDesc")}
          </p>
          {reqs.labels && reqs.labels.length > 0 && (
            <ul className="space-y-1">
              {reqs.labels.map((label) => (
                <li key={label.code} className="text-xs text-amber-700 dark:text-amber-300">
                  • {label.label}
                </li>
              ))}
            </ul>
          )}
          <button
            type="button"
            onClick={handleComplete}
            disabled={accountLinkMutation.isPending}
            className={cn(
              "inline-flex items-center gap-1.5 rounded-lg px-4 py-2 text-sm font-medium transition-all",
              "bg-amber-600 text-white hover:bg-amber-700 active:scale-[0.98]",
              "disabled:opacity-50 disabled:cursor-not-allowed",
            )}
          >
            {accountLinkMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <ExternalLink className="h-4 w-4" />
            )}
            {t("completeOnStripe")}
          </button>
        </div>
      </div>
    </div>
  )
}
