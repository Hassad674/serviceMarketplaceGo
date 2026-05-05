"use client"

import { Check, X } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { ProposalStatus } from "../types"

// Soleil v2 — Proposal stepper.
// Soleil pill steps (corail-soft active, sable off, corail border current).
// Active step animation kept calm: no pulse, simple corail dot inside the pill.

const STEPS = [
  "created",
  "accepted",
  "paid",
  "active",
  "completed",
] as const

type StepKey = (typeof STEPS)[number]

function getStepIndex(status: ProposalStatus): number {
  const mapping: Record<ProposalStatus, number> = {
    pending: 0,
    accepted: 1,
    paid: 2,
    active: 3,
    completed: 4,
    completion_requested: 3,
    declined: -1,
    withdrawn: -1,
    disputed: 3,
  }
  return mapping[status]
}

function isTerminalNegative(status: ProposalStatus): boolean {
  return status === "declined" || status === "withdrawn"
}

interface ProposalStepperProps {
  status: ProposalStatus
}

export function ProposalStepper({ status }: ProposalStepperProps) {
  const t = useTranslations("proposal")
  const currentIndex = getStepIndex(status)
  const isNegative = isTerminalNegative(status)

  const stepLabels: Record<StepKey, string> = {
    created: t("stepCreated"),
    accepted: t("stepAccepted"),
    paid: t("stepPaid"),
    active: t("stepActive"),
    completed: t("stepCompleted"),
  }

  if (isNegative) {
    return <NegativeStepper status={status} stepLabels={stepLabels} />
  }

  return (
    <div className="w-full">
      {/* Desktop: horizontal Soleil pills */}
      <div className="hidden sm:flex items-center justify-between gap-2">
        {STEPS.map((step, index) => (
          <StepItem
            key={step}
            label={stepLabels[step]}
            index={index}
            currentIndex={currentIndex}
            isLast={index === STEPS.length - 1}
          />
        ))}
      </div>
      {/* Mobile: simplified text + soft dots */}
      <div className="sm:hidden flex items-center gap-2">
        {STEPS.map((step, index) => (
          <MobileStepDot
            key={step}
            index={index}
            currentIndex={currentIndex}
          />
        ))}
        <span className="ml-2 font-mono text-[11px] font-bold uppercase tracking-[0.1em] text-primary">
          {stepLabels[STEPS[Math.min(Math.max(currentIndex, 0), STEPS.length - 1)]]}
        </span>
      </div>
    </div>
  )
}

interface StepItemProps {
  label: string
  index: number
  currentIndex: number
  isLast: boolean
}

function StepItem({ label, index, currentIndex, isLast }: StepItemProps) {
  const isPast = index < currentIndex
  const isCurrent = index === currentIndex
  const isFuture = index > currentIndex

  return (
    <div className="flex items-center flex-1 last:flex-none">
      <div className="flex flex-col items-center gap-1.5">
        <div
          className={cn(
            "flex h-8 w-8 shrink-0 items-center justify-center rounded-full border transition-colors duration-200",
            isPast && "border-success bg-success-soft text-success",
            isCurrent && "border-primary bg-primary-soft text-primary",
            isFuture && "border-border bg-background text-subtle-foreground",
          )}
        >
          {isPast && <Check className="h-4 w-4" strokeWidth={2.4} />}
          {isCurrent && (
            <span className="block h-2 w-2 rounded-full bg-primary" aria-hidden="true" />
          )}
        </div>
        <span
          className={cn(
            "text-[11.5px] font-medium whitespace-nowrap",
            isPast && "text-foreground",
            isCurrent && "font-bold text-primary",
            isFuture && "text-subtle-foreground",
          )}
        >
          {label}
        </span>
      </div>
      {!isLast && (
        <div
          className={cn(
            "flex-1 h-px mx-2 mt-[-1.25rem] transition-colors duration-200",
            index < currentIndex ? "bg-success" : "bg-border",
          )}
        />
      )}
    </div>
  )
}

interface MobileStepDotProps {
  index: number
  currentIndex: number
}

function MobileStepDot({ index, currentIndex }: MobileStepDotProps) {
  const isPast = index < currentIndex
  const isCurrent = index === currentIndex

  return (
    <div
      className={cn(
        "h-2 w-2 rounded-full transition-colors duration-200",
        isPast && "bg-success",
        isCurrent && "bg-primary",
        !isPast && !isCurrent && "bg-border-strong",
      )}
    />
  )
}

interface NegativeStepperProps {
  status: ProposalStatus
  stepLabels: Record<StepKey, string>
}

function NegativeStepper({ status, stepLabels }: NegativeStepperProps) {
  const t = useTranslations("proposal")
  const label = status === "declined" ? t("declined") : t("withdrawn")

  return (
    <div className="w-full">
      <div className="hidden sm:flex items-center gap-3">
        <div className="flex flex-col items-center gap-1.5">
          <div className="flex h-8 w-8 items-center justify-center rounded-full border border-success bg-success-soft text-success">
            <Check className="h-4 w-4" strokeWidth={2.4} />
          </div>
          <span className="text-[11.5px] font-medium text-foreground">
            {stepLabels.created}
          </span>
        </div>
        <div className="h-px w-8 mt-[-1.25rem] rounded-full bg-destructive/40" />
        <div className="flex flex-col items-center gap-1.5">
          <div className="flex h-8 w-8 items-center justify-center rounded-full border border-destructive bg-destructive text-destructive-foreground">
            <X className="h-4 w-4" strokeWidth={2.4} />
          </div>
          <span className="text-[11.5px] font-bold text-destructive">
            {label}
          </span>
        </div>
      </div>
      <div className="sm:hidden flex items-center gap-2">
        <div className="h-2 w-2 rounded-full bg-success" />
        <div className="h-2 w-2 rounded-full bg-destructive" />
        <span className="ml-2 font-mono text-[11px] font-bold uppercase tracking-[0.1em] text-destructive">
          {label}
        </span>
      </div>
    </div>
  )
}
