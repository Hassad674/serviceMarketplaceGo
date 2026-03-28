"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { ProposalStatus } from "../types"

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
      {/* Desktop: horizontal */}
      <div className="hidden sm:flex items-center justify-between">
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
      {/* Mobile: simplified text */}
      <div className="sm:hidden flex items-center gap-2">
        {STEPS.map((step, index) => (
          <MobileStepDot
            key={step}
            index={index}
            currentIndex={currentIndex}
          />
        ))}
        <span className="ml-2 text-sm font-medium text-slate-700 dark:text-slate-300">
          {stepLabels[STEPS[Math.min(currentIndex, STEPS.length - 1)]]}
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
            "flex h-8 w-8 shrink-0 items-center justify-center rounded-full border-2 transition-all duration-200",
            isPast && "border-green-500 bg-green-500",
            isCurrent && "border-rose-500 bg-rose-500 animate-pulse",
            isFuture && "border-slate-300 bg-transparent dark:border-slate-600",
          )}
        >
          {isPast && (
            <svg className="h-4 w-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
            </svg>
          )}
          {isCurrent && (
            <div className="h-2.5 w-2.5 rounded-full bg-white" />
          )}
        </div>
        <span
          className={cn(
            "text-xs font-medium whitespace-nowrap",
            isPast && "text-slate-900 dark:text-slate-100",
            isCurrent && "text-rose-600 dark:text-rose-400 font-semibold",
            isFuture && "text-slate-400 dark:text-slate-500",
          )}
        >
          {label}
        </span>
      </div>
      {!isLast && (
        <div
          className={cn(
            "flex-1 h-0.5 mx-2 mt-[-1.25rem] rounded-full transition-all duration-200",
            index < currentIndex
              ? "bg-green-500"
              : "bg-slate-200 dark:bg-slate-700",
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
        "h-2 w-2 rounded-full transition-all duration-200",
        isPast && "bg-green-500",
        isCurrent && "bg-rose-500 animate-pulse",
        !isPast && !isCurrent && "bg-slate-300 dark:bg-slate-600",
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
          <div className="flex h-8 w-8 items-center justify-center rounded-full border-2 border-green-500 bg-green-500">
            <svg className="h-4 w-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <span className="text-xs font-medium text-slate-900 dark:text-slate-100">
            {stepLabels.created}
          </span>
        </div>
        <div className="h-0.5 w-8 mt-[-1.25rem] rounded-full bg-red-400" />
        <div className="flex flex-col items-center gap-1.5">
          <div className="flex h-8 w-8 items-center justify-center rounded-full border-2 border-red-500 bg-red-500">
            <svg className="h-4 w-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={3}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </div>
          <span className="text-xs font-semibold text-red-600 dark:text-red-400">
            {label}
          </span>
        </div>
      </div>
      <div className="sm:hidden flex items-center gap-2">
        <div className="h-2 w-2 rounded-full bg-green-500" />
        <div className="h-2 w-2 rounded-full bg-red-500" />
        <span className="ml-2 text-sm font-medium text-red-600 dark:text-red-400">
          {label}
        </span>
      </div>
    </div>
  )
}
