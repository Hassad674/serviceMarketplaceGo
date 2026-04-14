"use client"

import { Plus, Trash2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { MilestoneFormItem } from "../types"
import {
  MAX_MILESTONES_PER_PROPOSAL,
  createEmptyMilestoneItem,
  sumMilestoneAmounts,
} from "../types"

type MilestoneEditorProps = {
  milestones: MilestoneFormItem[]
  onChange: (milestones: MilestoneFormItem[]) => void
  disabled?: boolean
}

/**
 * MilestoneEditor is the repeatable form that lets the client define
 * a multi-step project: each row holds a title, description, amount,
 * and optional deadline. The user can add up to 20 milestones and
 * remove any extra ones, but never the last one (a proposal must
 * always have at least one milestone — backend invariant).
 *
 * Sequence numbers are derived from the array index at submit time —
 * the editor itself only stores the form-level fields and reorders
 * are not exposed in V1 (V2 contract-change feature handles edits).
 *
 * The sticky footer shows the running total in EUR so the client
 * sees the full price as they type.
 */
export function MilestoneEditor({
  milestones,
  onChange,
  disabled = false,
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

  return (
    <div className="space-y-4" id="payment-mode-panel-milestone">
      <div className="flex items-baseline justify-between">
        <h2 className="text-sm font-medium text-gray-700 dark:text-gray-300">
          {t("label")}
        </h2>
        <span className="text-xs text-gray-500 dark:text-gray-400">
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
          />
        ))}
      </div>

      <button
        type="button"
        onClick={addMilestone}
        disabled={disabled || !canAddMore}
        className={cn(
          "flex w-full items-center justify-center gap-2 rounded-lg border-2 border-dashed",
          "px-4 py-3 text-sm font-medium transition-all duration-200",
          canAddMore && !disabled
            ? "border-gray-300 text-gray-600 hover:border-rose-400 hover:text-rose-600 dark:border-gray-600 dark:text-gray-400 dark:hover:border-rose-400 dark:hover:text-rose-400"
            : "cursor-not-allowed border-gray-200 text-gray-400 dark:border-gray-700 dark:text-gray-600",
        )}
      >
        <Plus className="h-4 w-4" />
        {t("addMilestone")}
      </button>

      {/* Sticky total footer */}
      <div
        className={cn(
          "flex items-center justify-between rounded-xl border bg-rose-50/50 px-5 py-4",
          "border-rose-200 dark:border-rose-900/50 dark:bg-rose-900/10",
        )}
      >
        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
          {t("total")}
        </span>
        <span className="text-2xl font-bold text-rose-600 dark:text-rose-400">
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
}

function MilestoneRow({
  sequence,
  milestone,
  disabled,
  canRemove,
  onChange,
  onRemove,
}: MilestoneRowProps) {
  const t = useTranslations("proposal.milestoneEditor")
  const todayIso = new Date().toISOString().split("T")[0]

  return (
    <div
      className={cn(
        "rounded-xl border border-gray-200 bg-white p-4 shadow-sm",
        "dark:border-gray-700 dark:bg-gray-800",
      )}
    >
      <div className="mb-3 flex items-center justify-between">
        <span
          className={cn(
            "inline-flex h-7 min-w-[2rem] items-center justify-center rounded-full",
            "bg-rose-100 px-2 text-xs font-semibold text-rose-700",
            "dark:bg-rose-900/40 dark:text-rose-300",
          )}
        >
          {t("milestone")} {sequence}
        </span>
        {canRemove && (
          <button
            type="button"
            onClick={onRemove}
            disabled={disabled}
            className={cn(
              "rounded-lg p-2 text-gray-400 transition-colors",
              "hover:bg-red-50 hover:text-red-600",
              "dark:hover:bg-red-900/20 dark:hover:text-red-400",
              disabled && "cursor-not-allowed opacity-50",
            )}
            aria-label={t("remove")}
          >
            <Trash2 className="h-4 w-4" />
          </button>
        )}
      </div>

      <div className="space-y-3">
        <input
          type="text"
          value={milestone.title}
          onChange={(e) => onChange({ title: e.target.value })}
          placeholder={t("titlePlaceholder")}
          disabled={disabled}
          className={cn(
            "h-10 w-full rounded-lg border border-gray-200 bg-white px-3 text-sm",
            "shadow-xs transition-all duration-200",
            "placeholder:text-gray-400",
            "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
            "dark:border-gray-700 dark:bg-gray-900 dark:text-white dark:placeholder:text-gray-500",
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
            "w-full rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm",
            "shadow-xs transition-all duration-200 resize-none",
            "placeholder:text-gray-400",
            "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
            "dark:border-gray-700 dark:bg-gray-900 dark:text-white dark:placeholder:text-gray-500",
          )}
          aria-label={`${t("milestone")} ${sequence} ${t("descriptionAriaLabel")}`}
        />

        <div className="grid grid-cols-2 gap-3">
          <div className="relative">
            <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-sm font-medium text-gray-500">
              &euro;
            </span>
            <input
              type="number"
              min="0"
              step="0.01"
              value={milestone.amount}
              onChange={(e) => onChange({ amount: e.target.value })}
              placeholder={t("amountPlaceholder")}
              disabled={disabled}
              className={cn(
                "h-10 w-full rounded-lg border border-gray-200 bg-white pl-8 pr-3 text-sm",
                "shadow-xs transition-all duration-200",
                "placeholder:text-gray-400",
                "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
                "dark:border-gray-700 dark:bg-gray-900 dark:text-white dark:placeholder:text-gray-500",
                "[appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none",
              )}
              aria-label={`${t("milestone")} ${sequence} ${t("amountAriaLabel")}`}
            />
          </div>
          <input
            type="date"
            value={milestone.deadline}
            min={todayIso}
            onChange={(e) => onChange({ deadline: e.target.value })}
            disabled={disabled}
            className={cn(
              "h-10 w-full rounded-lg border border-gray-200 bg-white px-3 text-sm",
              "shadow-xs transition-all duration-200",
              "text-gray-700 dark:text-gray-300",
              "focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10 focus:outline-none",
              "dark:border-gray-700 dark:bg-gray-900",
            )}
            aria-label={`${t("milestone")} ${sequence} ${t("deadlineAriaLabel")}`}
          />
        </div>
      </div>
    </div>
  )
}
