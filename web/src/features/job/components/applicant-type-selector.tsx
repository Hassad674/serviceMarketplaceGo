"use client"

import { Users, User, Building2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { ApplicantType } from "../types"

// W-09 — Soleil v2 segmented pill control for the "Qui peut postuler ?" picker.
// Behaviour and props are unchanged: a single radio-style choice between
// {all, freelancers, agencies}. Visual identity ports to ivoire/corail —
// active option = corail-soft fill + corail border + corail-deep label, off
// option = white surface + sable-light border + tabac label.

type ApplicantTypeSelectorProps = {
  value: ApplicantType
  onChange: (value: ApplicantType) => void
}

const APPLICANT_OPTIONS: ApplicantType[] = ["all", "freelancers", "agencies"]

const OPTION_ICONS: Record<ApplicantType, typeof Users> = {
  all: Users,
  freelancers: User,
  agencies: Building2,
}

export function ApplicantTypeSelector({ value, onChange }: ApplicantTypeSelectorProps) {
  const t = useTranslations("job")

  const labelMap: Record<ApplicantType, string> = {
    all: t("applicantAll"),
    freelancers: t("applicantFreelancers"),
    agencies: t("applicantAgencies"),
  }

  return (
    <div>
      <p className="mb-2 font-mono text-[11px] font-bold uppercase tracking-[0.08em] text-muted-foreground">
        {t("applicantType")}
      </p>
      <div
        className="grid gap-2 sm:grid-cols-3"
        role="radiogroup"
        aria-label={t("applicantType")}
      >
        {APPLICANT_OPTIONS.map((option) => {
          const isActive = value === option
          const Icon = OPTION_ICONS[option]
          return (
            <button
              key={option}
              type="button"
              role="radio"
              aria-checked={isActive}
              onClick={() => onChange(option)}
              className={cn(
                "flex items-center gap-3 rounded-2xl border px-4 py-3 text-left transition-all duration-200",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                isActive
                  ? "border-primary bg-primary-soft"
                  : "border-border bg-surface hover:border-border-strong",
              )}
            >
              <Icon
                className={cn(
                  "h-5 w-5 shrink-0",
                  isActive ? "text-primary-deep" : "text-muted-foreground",
                )}
                strokeWidth={1.6}
              />
              <span
                className={cn(
                  "flex-1 text-[14px] font-semibold leading-snug",
                  isActive ? "text-primary-deep" : "text-foreground",
                )}
              >
                {labelMap[option]}
              </span>
              <span
                aria-hidden="true"
                className={cn(
                  "flex h-[18px] w-[18px] shrink-0 items-center justify-center rounded-full border transition-all duration-200",
                  isActive
                    ? "border-primary bg-primary"
                    : "border-border-strong bg-surface",
                )}
              >
                {isActive && <span className="h-1.5 w-1.5 rounded-full bg-primary-foreground" />}
              </span>
            </button>
          )
        })}
      </div>
    </div>
  )
}
