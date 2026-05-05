"use client"

import { X, Ticket, RefreshCw, Trophy, TrendingUp } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"

// W-09 — Soleil v2 modal explaining how the candidacy credits work.
// Public surface (`open`, `onClose`) is unchanged so the existing caller
// (`opportunity-list.tsx`, Wave A locked) keeps working.

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
      className="fixed inset-0 z-50 flex items-center justify-center bg-foreground/40 p-4 backdrop-blur-sm"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-label={t("creditsHowItWorks")}
    >
      <div
        className={cn(
          "w-full max-w-md rounded-[20px] border border-border bg-surface p-6",
          "shadow-[0_24px_48px_rgba(42,31,21,0.16)]",
        )}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="mb-5 flex items-start justify-between gap-4">
          <div className="min-w-0">
            <p className="mb-1.5 font-mono text-[11px] font-bold uppercase tracking-[0.08em] text-primary">
              {t("creditsHowItWorks")}
            </p>
            <h3 className="font-serif text-[22px] font-medium leading-[1.15] tracking-[-0.01em] text-foreground">
              {t("creditsHowItWorks")}
            </h3>
          </div>
          <button
            type="button"
            onClick={onClose}
            className={cn(
              "rounded-full p-2 text-muted-foreground transition-all duration-200",
              "hover:bg-background hover:text-foreground",
              "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-surface",
            )}
            aria-label={t("close")}
          >
            <X className="h-4 w-4" strokeWidth={2} />
          </button>
        </div>

        {/* Explanation items */}
        <ul className="space-y-4">
          {explanations.map((text, index) => {
            const Icon = EXPLANATION_ICONS[index]
            return (
              <li key={index} className="flex items-start gap-3">
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-soft">
                  <Icon className="h-4 w-4 text-primary-deep" strokeWidth={1.6} />
                </div>
                <p className="pt-1.5 text-[14px] leading-relaxed text-foreground">
                  {text}
                </p>
              </li>
            )
          })}
        </ul>

        {/* Close button */}
        <button
          type="button"
          onClick={onClose}
          className={cn(
            "mt-6 inline-flex w-full items-center justify-center rounded-full",
            "bg-primary px-5 py-2.5 text-[13.5px] font-bold text-primary-foreground",
            "transition-all duration-200 ease-out",
            "hover:bg-primary-deep hover:shadow-[0_4px_14px_rgba(232,93,74,0.28)]",
            "active:scale-[0.98]",
            "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-surface",
          )}
        >
          {t("close")}
        </button>
      </div>
    </div>
  )
}
