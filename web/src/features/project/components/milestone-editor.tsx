"use client"

import { GripVertical, Plus, Trash2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { Milestone } from "../types"
import { createEmptyMilestone } from "../types"

type MilestoneEditorProps = {
  milestones: Milestone[]
  onChange: (milestones: Milestone[]) => void
}

export function MilestoneEditor({ milestones, onChange }: MilestoneEditorProps) {
  const t = useTranslations("projects")

  function updateMilestone(id: string, field: keyof Milestone, value: string) {
    onChange(
      milestones.map((m) =>
        m.id === id ? { ...m, [field]: value } : m,
      ),
    )
  }

  function removeMilestone(id: string) {
    if (milestones.length <= 1) return
    onChange(milestones.filter((m) => m.id !== id))
  }

  function addMilestone() {
    onChange([...milestones, createEmptyMilestone()])
  }

  return (
    <div className="space-y-3">
      {milestones.map((milestone, index) => (
        <MilestoneItem
          key={milestone.id}
          milestone={milestone}
          index={index}
          canDelete={milestones.length > 1}
          onUpdate={(field, value) => updateMilestone(milestone.id, field, value)}
          onDelete={() => removeMilestone(milestone.id)}
        />
      ))}
      <button
        type="button"
        onClick={addMilestone}
        className={cn(
          "flex w-full items-center justify-center gap-2 rounded-xl border-2 border-dashed",
          "border-gray-200 dark:border-gray-700 py-3 text-sm font-medium",
          "text-gray-500 dark:text-gray-400 transition-all duration-200",
          "hover:border-rose-300 hover:text-rose-600 dark:hover:border-rose-500/40 dark:hover:text-rose-400",
        )}
      >
        <Plus className="h-4 w-4" strokeWidth={2} />
        {t("addMilestone")}
      </button>
    </div>
  )
}

type MilestoneItemProps = {
  milestone: Milestone
  index: number
  canDelete: boolean
  onUpdate: (field: keyof Milestone, value: string) => void
  onDelete: () => void
}

function MilestoneItem({ milestone, index, canDelete, onUpdate, onDelete }: MilestoneItemProps) {
  const t = useTranslations("projects")

  return (
    <div
      className={cn(
        "group rounded-xl border border-gray-200 dark:border-gray-700",
        "bg-white dark:bg-gray-900 p-4 transition-all duration-200",
        "hover:border-gray-300 dark:hover:border-gray-600",
      )}
    >
      <div className="flex items-start gap-3">
        {/* Drag handle (visual only) */}
        <div className="mt-2 cursor-grab text-gray-300 dark:text-gray-600">
          <GripVertical className="h-5 w-5" strokeWidth={1.5} />
        </div>

        <div className="flex-1 space-y-3">
          {/* Title row */}
          <div className="flex items-center gap-3">
            <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-rose-100 dark:bg-rose-500/20 text-xs font-semibold text-rose-600 dark:text-rose-400">
              {index + 1}
            </span>
            <input
              type="text"
              value={milestone.title}
              onChange={(e) => onUpdate("title", e.target.value)}
              placeholder={t("milestoneTitle")}
              className={cn(
                "h-10 flex-1 rounded-lg border border-gray-200 dark:border-gray-700",
                "bg-gray-50 dark:bg-gray-800 px-3 text-sm",
                "text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500",
                "transition-all duration-200",
                "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
              )}
            />
          </div>

          {/* Description */}
          <textarea
            value={milestone.description}
            onChange={(e) => onUpdate("description", e.target.value)}
            placeholder={t("milestoneDesc")}
            rows={2}
            className={cn(
              "w-full rounded-lg border border-gray-200 dark:border-gray-700",
              "bg-gray-50 dark:bg-gray-800 px-3 py-2 text-sm",
              "text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500",
              "resize-none transition-all duration-200",
              "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
            )}
          />

          {/* Deadline + Amount row */}
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
                {t("milestoneDeadline")}
              </label>
              <input
                type="date"
                value={milestone.deadline}
                onChange={(e) => onUpdate("deadline", e.target.value)}
                className={cn(
                  "h-10 w-full rounded-lg border border-gray-200 dark:border-gray-700",
                  "bg-gray-50 dark:bg-gray-800 px-3 text-sm",
                  "text-gray-900 dark:text-white",
                  "transition-all duration-200",
                  "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
                )}
              />
            </div>
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">
                {t("milestoneAmount")}
              </label>
              <div className="relative">
                <input
                  type="number"
                  min="0"
                  step="0.01"
                  value={milestone.amount}
                  onChange={(e) => onUpdate("amount", e.target.value)}
                  placeholder="0.00"
                  className={cn(
                    "h-10 w-full rounded-lg border border-gray-200 dark:border-gray-700",
                    "bg-gray-50 dark:bg-gray-800 pl-3 pr-8 text-sm",
                    "text-gray-900 dark:text-white placeholder:text-gray-400 dark:placeholder:text-gray-500",
                    "transition-all duration-200",
                    "focus:border-rose-500 focus:bg-white dark:focus:bg-gray-900 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
                  )}
                />
                <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-sm text-gray-400">
                  &euro;
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* Delete button */}
        {canDelete && (
          <button
            type="button"
            onClick={onDelete}
            className={cn(
              "mt-2 rounded-lg p-1.5 text-gray-300 dark:text-gray-600",
              "transition-all duration-200",
              "hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-500/10 dark:hover:text-red-400",
            )}
            aria-label="Delete milestone"
          >
            <Trash2 className="h-4 w-4" strokeWidth={1.5} />
          </button>
        )}
      </div>
    </div>
  )
}
