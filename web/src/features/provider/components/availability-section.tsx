"use client"

import { useCallback, useMemo, useState } from "react"
import { Check, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { AvailabilityStatus } from "../api/profile-api"
import { useProfile } from "../hooks/use-profile"
import { useUpdateAvailability } from "../hooks/use-update-availability"
import { Button } from "@/shared/components/ui/button"

const STATUS_VALUES: AvailabilityStatus[] = [
  "available_now",
  "available_soon",
  "not_available",
]

// Tailwind pill colors for each status. Extracted into a constant so
// the component body stays skimmable and so the palette matches the
// hero strip / listing card without duplication.
const STATUS_STYLES: Record<AvailabilityStatus, string> = {
  available_now:
    "bg-emerald-50 text-emerald-700 border-emerald-200 hover:border-emerald-400 aria-checked:bg-emerald-500 aria-checked:text-white aria-checked:border-emerald-500 dark:bg-emerald-500/10 dark:text-emerald-300 dark:border-emerald-500/30",
  available_soon:
    "bg-amber-50 text-amber-700 border-amber-200 hover:border-amber-400 aria-checked:bg-amber-500 aria-checked:text-white aria-checked:border-amber-500 dark:bg-amber-500/10 dark:text-amber-300 dark:border-amber-500/30",
  not_available:
    "bg-rose-50 text-rose-700 border-rose-200 hover:border-rose-400 aria-checked:bg-rose-500 aria-checked:text-white aria-checked:border-rose-500 dark:bg-rose-500/10 dark:text-rose-300 dark:border-rose-500/30",
}

type AvailabilityVariant = "direct" | "referrer"

interface AvailabilitySectionProps {
  orgType: string | undefined
  referrerEnabled: boolean | undefined
  variant: AvailabilityVariant
  readOnly?: boolean
}

export function AvailabilitySection({
  orgType,
  referrerEnabled,
  variant,
  readOnly = false,
}: AvailabilitySectionProps) {
  const t = useTranslations("profile.availability")
  const { data: profile } = useProfile()
  const mutation = useUpdateAvailability()

  const persistedDirect = profile?.availability_status ?? "available_now"
  const persistedReferrer = profile?.referrer_availability_status ?? null

  // Sync local draft to persisted values through the "derive state
  // during render" pattern instead of an effect — this mirrors
  // React's official recommendation for resetting state on prop
  // changes without an extra render pass.
  const [prevDirect, setPrevDirect] = useState(persistedDirect)
  const [prevReferrer, setPrevReferrer] = useState(persistedReferrer)
  const [direct, setDirect] = useState<AvailabilityStatus>(persistedDirect)
  const [referrer, setReferrer] = useState<AvailabilityStatus>(
    persistedReferrer ?? "available_now",
  )
  if (prevDirect !== persistedDirect || prevReferrer !== persistedReferrer) {
    setPrevDirect(persistedDirect)
    setPrevReferrer(persistedReferrer)
    setDirect(persistedDirect)
    setReferrer(persistedReferrer ?? "available_now")
  }

  const referrerAllowed = useMemo(
    () => orgType === "provider_personal" && referrerEnabled === true,
    [orgType, referrerEnabled],
  )

  const isDirect = variant === "direct"
  const isDirty = isDirect
    ? direct !== persistedDirect
    : referrer !== (persistedReferrer ?? "available_now")

  const handleSave = useCallback(() => {
    mutation.mutate(
      isDirect
        ? { availability_status: direct }
        : { referrer_availability_status: referrer },
    )
  }, [isDirect, direct, referrer, mutation])

  if (orgType === "enterprise") return null
  if (readOnly) return null
  if (!isDirect && !referrerAllowed) return null

  return (
    <section
      aria-labelledby="availability-section-title"
      className="bg-card border border-border rounded-xl p-6 shadow-sm"
    >
      <header className="mb-4 flex flex-col gap-1">
        <h2
          id="availability-section-title"
          className="text-lg font-semibold text-foreground"
        >
          {t("sectionTitle")}
        </h2>
        <p className="text-sm text-muted-foreground">{t("sectionSubtitle")}</p>
      </header>

      <div className="space-y-4">
        {isDirect ? (
          <StatusRadioGroup
            label={t("directTitle")}
            value={direct}
            onChange={setDirect}
          />
        ) : (
          <StatusRadioGroup
            label={t("referrerTitle")}
            value={referrer}
            onChange={setReferrer}
          />
        )}
        <SaveRow
          isDirty={isDirty}
          isSaving={mutation.isPending}
          onSave={handleSave}
          isSuccess={mutation.isSuccess && !isDirty}
        />
      </div>
    </section>
  )
}

interface StatusRadioGroupProps {
  label: string
  value: AvailabilityStatus
  onChange: (next: AvailabilityStatus) => void
}

function StatusRadioGroup({ label, value, onChange }: StatusRadioGroupProps) {
  const t = useTranslations("profile.availability")
  return (
    <div>
      <p className="text-sm font-medium text-foreground mb-2">{label}</p>
      <div
        role="radiogroup"
        aria-label={label}
        className="flex flex-wrap gap-2"
      >
        {STATUS_VALUES.map((status) => {
          const isSelected = status === value
          return (
            <Button variant="ghost" size="auto"
              key={status}
              type="button"
              role="radio"
              aria-checked={isSelected}
              onClick={() => onChange(status)}
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
    </div>
  )
}

interface SaveRowProps {
  isDirty: boolean
  isSaving: boolean
  isSuccess: boolean
  onSave: () => void
}

function SaveRow({ isDirty, isSaving, isSuccess, onSave }: SaveRowProps) {
  const t = useTranslations("profile.availability")
  return (
    <div className="flex items-center justify-end gap-3">
      {isSuccess ? (
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
        onClick={onSave}
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
