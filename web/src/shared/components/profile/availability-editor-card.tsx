"use client"

import { useState } from "react"
import { Check, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"

import { Button } from "@/shared/components/ui/button"
export type AvailabilityStatus =
  | "available_now"
  | "available_soon"
  | "not_available"

const STATUS_VALUES: AvailabilityStatus[] = [
  "available_now",
  "available_soon",
  "not_available",
]

// Tailwind pill styles per status — colored idle + colored selected
// so the picker reads the same as the public-profile availability
// strip elsewhere in the app.
const STATUS_STYLES: Record<AvailabilityStatus, string> = {
  available_now:
    "bg-emerald-50 text-emerald-700 border-emerald-200 hover:border-emerald-400 aria-checked:bg-emerald-500 aria-checked:text-white aria-checked:border-emerald-500 dark:bg-emerald-500/10 dark:text-emerald-300 dark:border-emerald-500/30",
  available_soon:
    "bg-amber-50 text-amber-700 border-amber-200 hover:border-amber-400 aria-checked:bg-amber-500 aria-checked:text-white aria-checked:border-amber-500 dark:bg-amber-500/10 dark:text-amber-300 dark:border-amber-500/30",
  not_available:
    "bg-rose-50 text-rose-700 border-rose-200 hover:border-rose-400 aria-checked:bg-rose-500 aria-checked:text-white aria-checked:border-rose-500 dark:bg-rose-500/10 dark:text-rose-300 dark:border-rose-500/30",
}

interface AvailabilityEditorCardProps {
  value: AvailabilityStatus
  onSave: (next: AvailabilityStatus) => Promise<void>
  isSaving: boolean
  titleOverride?: string
  subtitleOverride?: string
}

// Generic availability editor used by both the freelance and
// referrer profile pages. Stateless w.r.t. the data source — the
// feature page wires its own mutation hook and passes `onSave`.
// Keeps the dirty-state sync in-component so each persona's card
// tracks its own draft independently.
export function AvailabilityEditorCard({
  value,
  onSave,
  isSaving,
  titleOverride,
  subtitleOverride,
}: AvailabilityEditorCardProps) {
  const t = useTranslations("profile.availability")
  const [prevValue, setPrevValue] = useState(value)
  const [draft, setDraft] = useState<AvailabilityStatus>(value)
  const [justSaved, setJustSaved] = useState(false)

  if (prevValue !== value) {
    setPrevValue(value)
    setDraft(value)
  }

  const isDirty = draft !== value

  async function handleSave() {
    setJustSaved(false)
    await onSave(draft)
    setJustSaved(true)
  }

  return (
    <section
      aria-labelledby="availability-editor-title"
      className="bg-card border border-border rounded-xl p-6 shadow-sm"
    >
      <header className="mb-4 flex flex-col gap-1">
        <h2
          id="availability-editor-title"
          className="text-lg font-semibold text-foreground"
        >
          {titleOverride ?? t("sectionTitle")}
        </h2>
        <p className="text-sm text-muted-foreground">
          {subtitleOverride ?? t("sectionSubtitle")}
        </p>
      </header>

      <div
        role="radiogroup"
        aria-label={titleOverride ?? t("sectionTitle")}
        className="flex flex-wrap gap-2"
      >
        {STATUS_VALUES.map((status) => {
          const isSelected = status === draft
          return (
            <Button variant="ghost" size="auto"
              key={status}
              type="button"
              role="radio"
              aria-checked={isSelected}
              onClick={() => {
                setDraft(status)
                setJustSaved(false)
              }}
              className={cn(
                "inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm font-medium border transition-all duration-150",
                "focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2",
                STATUS_STYLES[status],
              )}
            >
              {isSelected ? (
                <Check className="w-3.5 h-3.5" aria-hidden="true" />
              ) : null}
              {t(statusKey(status))}
            </Button>
          )
        })}
      </div>

      <div className="mt-4 flex items-center justify-end gap-3">
        {justSaved && !isDirty ? (
          <span
            role="status"
            className="text-xs text-emerald-600 dark:text-emerald-400 inline-flex items-center gap-1"
          >
            <Check className="w-3 h-3" aria-hidden="true" />
            {t("saved")}
          </span>
        ) : null}
        <Button variant="ghost" size="auto"
          type="button"
          onClick={handleSave}
          disabled={!isDirty || isSaving}
          className="bg-primary text-primary-foreground rounded-md h-9 px-4 text-sm font-medium hover:opacity-90 transition-opacity duration-150 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50 disabled:cursor-not-allowed inline-flex items-center gap-2"
        >
          {isSaving ? (
            <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />
          ) : (
            <Check className="w-4 h-4" aria-hidden="true" />
          )}
          {isSaving ? t("saving") : t("save")}
        </Button>
      </div>
    </section>
  )
}

function statusKey(status: AvailabilityStatus): string {
  switch (status) {
    case "available_now":
      return "statusAvailableNow"
    case "available_soon":
      return "statusAvailableSoon"
    case "not_available":
      return "statusNotAvailable"
  }
}
