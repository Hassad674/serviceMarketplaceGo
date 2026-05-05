"use client"

import { Building2, User } from "lucide-react"

import { cn } from "@/shared/lib/utils"
import { Button } from "@/shared/components/ui/button"
type BusinessType = "individual" | "company"

type BusinessTypeCardProps = {
  value: BusinessType | null
  onChange: (type: BusinessType) => void
  disabled?: boolean
  /** When true, the "individual" option is disabled with a tooltip. */
  individualBlocked?: boolean
  /** Reason shown as tooltip when an option is blocked. */
  blockedReason?: string
}

const OPTIONS: {
  type: BusinessType
  title: string
  description: string
  icon: typeof User
  details: string[]
}[] = [
  {
    type: "individual",
    title: "Individual",
    description: "Freelance or independent professional",
    icon: User,
    details: ["Identity document", "Personal address", "Bank details"],
  },
  {
    type: "company",
    title: "Registered business",
    description: "Company, partnership or other legal entity",
    icon: Building2,
    details: ["Business registration", "Legal representative", "Beneficial owners"],
  },
]

export function BusinessTypeCard({
  value,
  onChange,
  disabled,
  individualBlocked,
  blockedReason,
}: BusinessTypeCardProps) {
  return (
    <div role="radiogroup" aria-label="Type de compte" className="grid gap-3 sm:grid-cols-2">
      {OPTIONS.map((opt) => {
        const Icon = opt.icon
        const selected = value === opt.type
        const isBlocked = individualBlocked && opt.type === "individual"
        const isDisabled = disabled || isBlocked
        return (
          <Button
            variant="ghost"
            size="auto"
            key={opt.type}
            type="button"
            role="radio"
            aria-checked={selected}
            disabled={isDisabled}
            title={isBlocked ? blockedReason : undefined}
            onClick={() => !isBlocked && onChange(opt.type)}
            className={cn(
              "group relative flex flex-col gap-3 rounded-2xl border bg-card p-5 text-left transition-all",
              isDisabled
                ? "cursor-not-allowed opacity-60"
                : selected
                  ? "border-primary shadow-card ring-4 ring-primary/15"
                  : "border-border-strong hover:-translate-y-0.5 hover:border-primary/60 hover:shadow-card",
            )}
          >
            {isBlocked ? (
              <span className="absolute right-3 top-3 rounded-full border border-warning/30 bg-amber-soft px-2 py-0.5 text-[10px] font-semibold text-warning">
                Non disponible
              </span>
            ) : null}
            <div className="flex items-center justify-between">
              <div
                className={cn(
                  "flex h-11 w-11 items-center justify-center rounded-xl transition-colors",
                  selected
                    ? "bg-primary text-primary-foreground"
                    : "bg-primary-soft text-primary group-hover:bg-primary-soft/80",
                )}
              >
                <Icon className="h-5 w-5" aria-hidden />
              </div>
              <span
                className={cn(
                  "flex h-5 w-5 items-center justify-center rounded-full border-2 transition-all",
                  selected
                    ? "border-primary bg-primary"
                    : "border-border-strong bg-card",
                )}
                aria-hidden
              >
                {selected ? <span className="h-2 w-2 rounded-full bg-card" /> : null}
              </span>
            </div>
            <div>
              <div className="text-[15px] font-semibold text-foreground">{opt.title}</div>
              <p className="mt-0.5 text-[13px] leading-snug text-muted-foreground">
                {opt.description}
              </p>
            </div>
            <ul className="mt-1 space-y-1">
              {opt.details.map((detail) => (
                <li
                  key={detail}
                  className="flex items-center gap-1.5 text-[12px] text-muted-foreground"
                >
                  <span className="h-1 w-1 rounded-full bg-subtle-foreground" aria-hidden />
                  {detail}
                </li>
              ))}
            </ul>
          </Button>
        )
      })}
    </div>
  )
}
