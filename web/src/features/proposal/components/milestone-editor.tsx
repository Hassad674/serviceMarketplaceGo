"use client"

import { Plus, Trash2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { MilestoneDeadlineErrorKey, MilestoneFormItem } from "../types"
import {
  MAX_MILESTONES_PER_PROPOSAL,
  createEmptyMilestoneItem,
  minDateForMilestone,
  sumMilestoneAmounts,
  validateMilestoneDeadlines,
} from "../types"

import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"

// Soleil v2 — Milestone editor. Repeatable card-per-milestone with
// rounded corail StadiumBorder + / - buttons, sticky total footer in
// corail-soft with Geist Mono numerals.

type MilestoneEditorProps = {
  milestones: MilestoneFormItem[]
  onChange: (milestones: MilestoneFormItem[]) => void
  disabled?: boolean
  /**
   * Optional proposal-level deadline (YYYY-MM-DD). When provided, every
   * milestone deadline must be ≤ projectDeadline — the editor surfaces
   * the violation inline next to the offending row.
   */
  projectDeadline?: string
}

export function MilestoneEditor({
  milestones,
  onChange,
  disabled = false,
  projectDeadline,
}: MilestoneEditorProps) {
  const t = useTranslations("proposal.milestoneEditor")

  function updateAt(index: number, patch: Partial<MilestoneFormItem>) {
    const next = milestones.map((m, i) => (i === index ? { ...m, ...patch } : m))
    onChange(next)
  }

  function addMilestone() {
    if (milestones.length >= MAX_MILESTONES_PER_PROPOSAL) return
    onChange([...milestones, createEmptyMilestoneItem()])
  }

  function removeAt(index: number) {
    if (milestones.length <= 1) return
    onChange(milestones.filter((_, i) => i !== index))
  }

  const totalCents = sumMilestoneAmounts(milestones)
  const totalEuros = (totalCents / 100).toFixed(2)
  const canAddMore = milestones.length < MAX_MILESTONES_PER_PROPOSAL

  // Compute the per-row deadline errors once per render so each
  // MilestoneRow doesn't have to redo the full O(N) walk for every
  // input event. Sparse map: only offending indexes appear.
  const deadlineErrors = validateMilestoneDeadlines(milestones, projectDeadline)
  const todayIso = new Date().toISOString().split("T")[0]

  return (
    <div className="space-y-4" id="payment-mode-panel-milestone">
      <div className="flex items-baseline justify-between">
        <h2 className="font-serif text-[16px] font-medium tracking-[-0.01em] text-foreground">
          {t("label")}
        </h2>
        <span className="font-mono text-[11px] font-bold uppercase tracking-[0.08em] text-subtle-foreground">
          {t("count", { current: milestones.length, max: MAX_MILESTONES_PER_PROPOSAL })}
        </span>
      </div>

      <div className="space-y-3">
        {milestones.map((m, index) => (
          <MilestoneRow
            key={index}
            sequence={index + 1}
            milestone={m}
            disabled={disabled}
            canRemove={milestones.length > 1}
            onChange={(patch) => updateAt(index, patch)}
            onRemove={() => removeAt(index)}
            minDate={minDateForMilestone(milestones, index, todayIso)}
            maxDate={projectDeadline}
            deadlineError={deadlineErrors[index]}
          />
        ))}
      </div>

      <Button
        variant="ghost"
        size="auto"
        type="button"
        onClick={addMilestone}
        disabled={disabled || !canAddMore}
        className={cn(
          "flex w-full items-center justify-center gap-2 rounded-full border-2 border-dashed",
          "px-5 py-3 text-[13.5px] font-bold transition-all duration-200 ease-out",
          canAddMore && !disabled
            ? "border-primary/60 text-primary hover:border-primary hover:bg-primary-soft"
            : "cursor-not-allowed border-border text-subtle-foreground",
        )}
      >
        <Plus className="h-4 w-4" strokeWidth={2} />
        {t("addMilestone")}
      </Button>

      {/* Sticky total footer */}
      <div
        className={cn(
          "flex items-center justify-between rounded-2xl border border-primary/30 bg-primary-soft px-5 py-4",
        )}
      >
        <span className="font-mono text-[11px] font-bold uppercase tracking-[0.1em] text-primary-deep">
          {t("total")}
        </span>
        <span className="font-mono text-[22px] font-bold text-primary-deep">
          {totalEuros}&nbsp;&euro;
        </span>
      </div>
    </div>
  )
}

type MilestoneRowProps = {
  sequence: number
  milestone: MilestoneFormItem
  disabled: boolean
  canRemove: boolean
  onChange: (patch: Partial<MilestoneFormItem>) => void
  onRemove: () => void
  /** YYYY-MM-DD lower bound for the date picker. */
  minDate: string
  /** YYYY-MM-DD upper bound for the date picker (project deadline). */
  maxDate?: string
  /** Inline error for the deadline field, or undefined when valid. */
  deadlineError?: MilestoneDeadlineErrorKey
}

function MilestoneRow({
  sequence,
  milestone,
  disabled,
  canRemove,
  onChange,
  onRemove,
  minDate,
  maxDate,
  deadlineError,
}: MilestoneRowProps) {
  const t = useTranslations("proposal.milestoneEditor")

  return (
    <div
      className={cn(
        "rounded-2xl border border-border bg-card p-5",
      )}
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      <div className="mb-4 flex items-center justify-between">
        <span
          className={cn(
            "inline-flex items-center gap-1.5 rounded-full bg-primary-soft px-3 py-1",
            "font-mono text-[11px] font-bold uppercase tracking-[0.08em] text-primary-deep",
          )}
        >
          {t("milestone")} {sequence}
        </span>
        {canRemove && (
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={onRemove}
            disabled={disabled}
            className={cn(
              "rounded-full p-2 text-subtle-foreground transition-colors duration-150",
              "hover:bg-destructive/10 hover:text-destructive",
              disabled && "cursor-not-allowed opacity-50",
            )}
            aria-label={t("remove")}
          >
            <Trash2 className="h-4 w-4" strokeWidth={1.7} />
          </Button>
        )}
      </div>

      <div className="space-y-3">
        <Input
          type="text"
          value={milestone.title}
          onChange={(e) => onChange({ title: e.target.value })}
          placeholder={t("titlePlaceholder")}
          disabled={disabled}
          className={cn(
            "h-11 w-full rounded-xl border border-border bg-background px-4 text-[14px]",
            "transition-all duration-200 ease-out",
            "placeholder:text-subtle-foreground",
            "focus:border-primary focus:ring-4 focus:ring-primary/15 focus:outline-none",
          )}
          aria-label={`${t("milestone")} ${sequence} ${t("titleAriaLabel")}`}
        />

        <textarea
          value={milestone.description}
          onChange={(e) => onChange({ description: e.target.value })}
          placeholder={t("descriptionPlaceholder")}
          rows={2}
          disabled={disabled}
          className={cn(
            "w-full rounded-xl border border-border bg-background px-4 py-3 text-[14px] resize-none",
            "transition-all duration-200 ease-out",
            "placeholder:text-subtle-foreground",
            "focus:border-primary focus:ring-4 focus:ring-primary/15 focus:outline-none",
          )}
          aria-label={`${t("milestone")} ${sequence} ${t("descriptionAriaLabel")}`}
        />

        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div className="relative">
            <span className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 font-mono text-[14px] font-medium text-subtle-foreground">
              &euro;
            </span>
            <Input
              type="number"
              min="0"
              step="0.01"
              value={milestone.amount}
              onChange={(e) => onChange({ amount: e.target.value })}
              placeholder={t("amountPlaceholder")}
              disabled={disabled}
              className={cn(
                "h-11 w-full rounded-xl border border-border bg-background pl-9 pr-4 text-[14px] font-mono",
                "transition-all duration-200 ease-out",
                "placeholder:text-subtle-foreground",
                "focus:border-primary focus:ring-4 focus:ring-primary/15 focus:outline-none",
                "[appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none",
              )}
              aria-label={`${t("milestone")} ${sequence} ${t("amountAriaLabel")}`}
            />
          </div>
          <div>
            <Input
              type="date"
              value={milestone.deadline}
              min={minDate}
              max={maxDate}
              onChange={(e) => onChange({ deadline: e.target.value })}
              disabled={disabled}
              aria-invalid={deadlineError ? "true" : "false"}
              aria-describedby={
                deadlineError ? `milestone-${sequence}-deadline-error` : undefined
              }
              className={cn(
                "h-11 w-full rounded-xl border bg-background px-4 text-[14px] font-mono",
                "transition-all duration-200 ease-out text-foreground",
                "focus:ring-4 focus:outline-none",
                deadlineError
                  ? "border-destructive focus:border-destructive focus:ring-destructive/15"
                  : "border-border focus:border-primary focus:ring-primary/15",
              )}
              aria-label={`${t("milestone")} ${sequence} ${t("deadlineAriaLabel")}`}
            />
            {deadlineError && (
              <p
                id={`milestone-${sequence}-deadline-error`}
                role="alert"
                className="mt-1.5 text-[12px] text-destructive"
              >
                {t(`error.${deadlineError}`)}
              </p>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
