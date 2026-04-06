"use client"

import { Building2, User } from "lucide-react"

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
          <button
            key={opt.type}
            type="button"
            role="radio"
            aria-checked={selected}
            disabled={isDisabled}
            title={isBlocked ? blockedReason : undefined}
            onClick={() => !isBlocked && onChange(opt.type)}
            className={`group relative flex flex-col gap-3 rounded-2xl border-2 bg-white p-5 text-left transition-all ${
              isDisabled
                ? "cursor-not-allowed opacity-60"
                : selected
                  ? "border-rose-500 shadow-md ring-4 ring-rose-500/10"
                  : "border-slate-200 hover:-translate-y-0.5 hover:border-slate-300 hover:shadow-md"
            }`}
          >
            {isBlocked ? (
              <span className="absolute right-3 top-3 rounded-full border border-amber-200 bg-amber-50 px-2 py-0.5 text-[10px] font-semibold text-amber-800">
                Non disponible
              </span>
            ) : null}
            <div className="flex items-center justify-between">
              <div
                className={`flex h-11 w-11 items-center justify-center rounded-xl transition-colors ${
                  selected
                    ? "bg-gradient-to-br from-rose-500 to-rose-600 text-white"
                    : "bg-slate-100 text-slate-600 group-hover:bg-slate-200"
                }`}
              >
                <Icon className="h-5 w-5" aria-hidden />
              </div>
              <span
                className={`flex h-5 w-5 items-center justify-center rounded-full border-2 transition-all ${
                  selected ? "border-rose-500 bg-rose-500" : "border-slate-300 bg-white"
                }`}
                aria-hidden
              >
                {selected ? <span className="h-2 w-2 rounded-full bg-white" /> : null}
              </span>
            </div>
            <div>
              <div className="text-[15px] font-semibold text-slate-900">{opt.title}</div>
              <p className="mt-0.5 text-[13px] leading-snug text-slate-500">{opt.description}</p>
            </div>
            <ul className="mt-1 space-y-1">
              {opt.details.map((detail) => (
                <li
                  key={detail}
                  className="flex items-center gap-1.5 text-[12px] text-slate-500"
                >
                  <span className="h-1 w-1 rounded-full bg-slate-300" aria-hidden />
                  {detail}
                </li>
              ))}
            </ul>
          </button>
        )
      })}
    </div>
  )
}
