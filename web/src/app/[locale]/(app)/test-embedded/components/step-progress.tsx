"use client"

import { Check } from "lucide-react"

type StepProgressProps = {
  currentStep: 1 | 2 | 3
}

const STEPS = [
  { id: 1, title: "Informations", subtitle: "Pays et type" },
  { id: 2, title: "Vérification", subtitle: "Identité et banque" },
  { id: 3, title: "Activation", subtitle: "Prêt à recevoir" },
] as const

export function StepProgress({ currentStep }: StepProgressProps) {
  return (
    <nav aria-label="Progression de l'inscription" className="w-full">
      <ol className="flex items-start justify-between gap-2">
        {STEPS.map((step, idx) => {
          const isDone = step.id < currentStep
          const isActive = step.id === currentStep
          const isLast = idx === STEPS.length - 1
          return (
            <li key={step.id} className="relative flex flex-1 items-start gap-3">
              <div className="flex flex-col items-center">
                <span
                  className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-full border-2 text-[13px] font-semibold transition-all ${
                    isDone
                      ? "border-rose-500 bg-rose-500 text-white"
                      : isActive
                        ? "border-rose-500 bg-white text-rose-600 ring-4 ring-rose-500/15"
                        : "border-slate-200 bg-white text-slate-400"
                  }`}
                  aria-current={isActive ? "step" : undefined}
                >
                  {isDone ? <Check className="h-4 w-4" aria-hidden /> : step.id}
                </span>
              </div>
              <div className="flex min-w-0 flex-col pt-0.5">
                <span
                  className={`text-[13px] font-semibold transition-colors ${
                    isDone || isActive ? "text-slate-900" : "text-slate-400"
                  }`}
                >
                  {step.title}
                </span>
                <span className="hidden text-[11px] text-slate-500 sm:block">
                  {step.subtitle}
                </span>
              </div>
              {!isLast ? (
                <div
                  className={`absolute left-[18px] top-10 h-[calc(100%+12px)] w-px -translate-x-1/2 transition-colors sm:left-auto sm:right-0 sm:top-[18px] sm:h-px sm:w-[calc(100%-52px)] sm:translate-x-0 ${
                    isDone ? "bg-rose-500" : "bg-slate-200"
                  }`}
                  aria-hidden
                />
              ) : null}
            </li>
          )
        })}
      </ol>
    </nav>
  )
}
