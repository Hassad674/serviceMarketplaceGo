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

// Soleil v2 pill styles per status — calm tinted backgrounds, bolder
// border + font weight on the selected chip. Selected uses semantic
// soft tokens (success-soft / amber-soft / muted) plus the matching
// strong border for clear focus, matching soleil-lotD.jsx.
const STATUS_STYLES: Record<AvailabilityStatus, string> = {
  available_now:
    "bg-card border-border text-muted-foreground hover:border-success/60 aria-checked:bg-success-soft aria-checked:text-success aria-checked:border-success aria-checked:font-semibold",
  available_soon:
    "bg-card border-border text-muted-foreground hover:border-warning/60 aria-checked:bg-amber-soft aria-checked:text-warning aria-checked:border-warning aria-checked:font-semibold",
  not_available:
    "bg-card border-border text-muted-foreground hover:border-border-strong aria-checked:bg-muted aria-checked:text-foreground aria-checked:border-border-strong aria-checked:font-semibold",
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
      className="bg-card border border-border rounded-2xl p-7 shadow-[var(--shadow-card)]"
    >
      <header className="mb-4 flex flex-col gap-1">
        <h2
          id="availability-editor-title"
          className="font-serif text-xl font-medium tracking-[-0.005em] text-foreground"
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
        className="flex flex-wrap gap-1.5"
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
                "inline-flex flex-1 items-center justify-center gap-1.5 rounded-xl px-4 py-2.5 text-sm font-medium border transition-all duration-150",
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
            className="text-xs text-success inline-flex items-center gap-1"
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
