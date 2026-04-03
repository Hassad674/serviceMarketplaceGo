"use client"

import { X, Ticket, RefreshCw, Trophy, TrendingUp } from "lucide-react"
import { useTranslations } from "next-intl"

interface CreditsInfoModalProps {
  open: boolean
  onClose: () => void
}

const EXPLANATION_ICONS = [Ticket, RefreshCw, Trophy, TrendingUp] as const

export function CreditsInfoModal({ open, onClose }: CreditsInfoModalProps) {
  const t = useTranslations("opportunity")

  if (!open) return null

  const explanations = [
    t("creditsExplanation1"),
    t("creditsExplanation2"),
    t("creditsExplanation3"),
    t("creditsExplanation4"),
  ]

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-label={t("creditsHowItWorks")}
    >
      <div
        className="bg-white dark:bg-slate-800 rounded-2xl shadow-xl p-6 w-full max-w-md mx-4 animate-scale-in"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between mb-5">
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
            {t("creditsHowItWorks")}
          </h3>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
            aria-label={t("close")}
          >
            <X className="h-5 w-5 text-slate-400" />
          </button>
        </div>

        {/* Explanation items */}
        <ul className="space-y-4">
          {explanations.map((text, index) => {
            const Icon = EXPLANATION_ICONS[index]
            return (
              <li key={index} className="flex items-start gap-3">
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-rose-100 dark:bg-rose-500/20">
                  <Icon className="h-4.5 w-4.5 text-rose-600 dark:text-rose-400" />
                </div>
                <p className="text-sm text-slate-600 dark:text-slate-300 pt-1.5">{text}</p>
              </li>
            )
          })}
        </ul>

        {/* Close button */}
        <button
          type="button"
          onClick={onClose}
          className="mt-6 w-full rounded-xl bg-slate-100 px-4 py-2.5 text-sm font-medium text-slate-700 hover:bg-slate-200 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-slate-600 transition-all"
        >
          {t("close")}
        </button>
      </div>
    </div>
  )
}
