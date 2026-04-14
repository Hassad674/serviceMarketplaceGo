"use client"

import { useCallback, useEffect, useMemo, useState } from "react"
import { Check, Loader2, Trash2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { ApiError } from "@/shared/lib/api-client"
import type {
  Pricing,
  PricingKind,
  PricingType,
} from "../api/profile-api"
import { SUPPORTED_FIAT_CURRENCIES } from "../lib/pricing-format"

interface PricingKindFormProps {
  kind: PricingKind
  persisted: Pricing | null
  onSave: (row: Pricing) => Promise<void>
  onDelete: (kind: PricingKind) => Promise<void>
  onPreviewChange: (row: Pricing | null) => void
}

// Form for one pricing kind (direct OR referral). Kind determines the
// allowed pricing types — direct supports hourly/daily/project_*, while
// referral only supports the two commission types.
export function PricingKindForm({
  kind,
  persisted,
  onSave,
  onDelete,
  onPreviewChange,
}: PricingKindFormProps) {
  const t = useTranslations("profile.pricing")
  const allowedTypes = useMemo(
    () => (kind === "direct" ? DIRECT_TYPES : REFERRAL_TYPES),
    [kind],
  )

  const initialType = persisted?.type ?? allowedTypes[0]
  const initialCurrency = persisted?.currency ?? defaultCurrencyForType(initialType)

  const [type, setType] = useState<PricingType>(initialType)
  const [currencyRaw, setCurrencyRaw] = useState<string>(initialCurrency)
  // Derive the effective currency during render so we don't need an
  // effect to keep it in sync with the pricing type.
  const currency =
    type === "commission_pct" ? "pct" : currencyRaw === "pct" ? "EUR" : currencyRaw
  const setCurrency = (next: string) => setCurrencyRaw(next)
  const [minDisplay, setMinDisplay] = useState<string>(
    persisted ? displayFromStored(persisted.min_amount, persisted.type) : "",
  )
  const [maxDisplay, setMaxDisplay] = useState<string>(
    persisted && persisted.max_amount !== null
      ? displayFromStored(persisted.max_amount, persisted.type)
      : "",
  )
  const [note, setNote] = useState<string>(persisted?.note ?? "")
  const [isSaving, setIsSaving] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  const toRow = useCallback((): Pricing | null => {
    const minNumber = Number(minDisplay)
    if (!Number.isFinite(minNumber) || minNumber < 0) return null
    const hasRange = type === "project_range" || type === "commission_pct"
    const maxNumber = maxDisplay.trim() === "" ? null : Number(maxDisplay)
    if (hasRange && (maxNumber === null || !Number.isFinite(maxNumber))) {
      return null
    }
    return {
      kind,
      type,
      min_amount: storedFromDisplay(minNumber, type),
      max_amount:
        hasRange && maxNumber !== null
          ? storedFromDisplay(maxNumber, type)
          : null,
      currency,
      note: note.trim(),
    }
  }, [currency, kind, maxDisplay, minDisplay, note, type])

  // Propagate preview upward whenever the form changes so the parent
  // modal can render the preview strip live.
  useEffect(() => {
    onPreviewChange(toRow())
  }, [onPreviewChange, toRow])

  const handleSave = useCallback(async () => {
    setErrorMessage(null)
    const row = toRow()
    if (!row) {
      setErrorMessage(t("errorGeneric"))
      return
    }
    setIsSaving(true)
    try {
      await onSave(row)
    } catch (caught) {
      setErrorMessage(mapErrorToMessage(caught, t))
    } finally {
      setIsSaving(false)
    }
  }, [onSave, t, toRow])

  const handleDelete = useCallback(async () => {
    setErrorMessage(null)
    setIsDeleting(true)
    try {
      await onDelete(kind)
      setMinDisplay("")
      setMaxDisplay("")
      setNote("")
    } catch (caught) {
      setErrorMessage(mapErrorToMessage(caught, t))
    } finally {
      setIsDeleting(false)
    }
  }, [kind, onDelete, t])

  const showMax = type === "project_range" || type === "commission_pct"
  const isPct = type === "commission_pct"
  const title = kind === "direct" ? t("kindDirect") : t("kindReferral")

  return (
    <section
      aria-label={title}
      className="rounded-xl border border-border bg-card p-4"
    >
      <div className="mb-3 flex items-center justify-between">
        <h3 className="text-base font-semibold text-foreground">{title}</h3>
        {persisted ? (
          <button
            type="button"
            onClick={handleDelete}
            disabled={isDeleting}
            className="inline-flex items-center gap-1.5 rounded-md h-8 px-3 text-xs font-medium text-destructive hover:bg-destructive/10 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50"
          >
            <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
            {t("delete")}
          </button>
        ) : null}
      </div>

      <TypeRadioRow
        allowedTypes={allowedTypes}
        value={type}
        onChange={(next) => setType(next)}
      />

      <div className="mt-3 grid gap-3 sm:grid-cols-2">
        <AmountInput
          id={`pricing-${kind}-min`}
          label={t("minAmountLabel")}
          value={minDisplay}
          onChange={setMinDisplay}
          suffix={isPct ? "%" : undefined}
        />
        {showMax ? (
          <AmountInput
            id={`pricing-${kind}-max`}
            label={t("maxAmountLabel")}
            value={maxDisplay}
            onChange={setMaxDisplay}
            suffix={isPct ? "%" : undefined}
          />
        ) : null}
      </div>

      <div className="mt-3 grid gap-3 sm:grid-cols-2">
        <div>
          <label
            htmlFor={`pricing-${kind}-currency`}
            className="block text-xs font-medium text-foreground mb-1"
          >
            {t("currencyLabel")}
          </label>
          <select
            id={`pricing-${kind}-currency`}
            value={currency}
            onChange={(e) => setCurrency(e.target.value)}
            disabled={isPct}
            className="w-full h-9 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10 focus:outline-none disabled:opacity-60"
          >
            {isPct ? (
              <option value="pct">%</option>
            ) : (
              SUPPORTED_FIAT_CURRENCIES.map((code) => (
                <option key={code} value={code}>
                  {code}
                </option>
              ))
            )}
          </select>
        </div>
        <div>
          <label
            htmlFor={`pricing-${kind}-note`}
            className="block text-xs font-medium text-foreground mb-1"
          >
            {t("noteLabel")}
          </label>
          <input
            id={`pricing-${kind}-note`}
            type="text"
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder={t("notePlaceholder")}
            maxLength={120}
            className="w-full h-9 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10 focus:outline-none"
          />
        </div>
      </div>

      {errorMessage ? (
        <p
          role="alert"
          className="mt-3 rounded-md border border-destructive/20 bg-destructive/5 px-3 py-2 text-sm text-destructive"
        >
          {errorMessage}
        </p>
      ) : null}

      <div className="mt-3 flex justify-end">
        <button
          type="button"
          onClick={handleSave}
          disabled={isSaving}
          className="inline-flex items-center gap-2 rounded-md bg-primary h-9 px-4 text-sm font-medium text-primary-foreground hover:opacity-90 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50"
        >
          {isSaving ? (
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          ) : (
            <Check className="h-4 w-4" aria-hidden="true" />
          )}
          {isSaving ? t("saving") : t("save")}
        </button>
      </div>
    </section>
  )
}

// ----- Sub-components ---------------------------------------------------

const DIRECT_TYPES: readonly PricingType[] = [
  "daily",
  "hourly",
  "project_from",
  "project_range",
] as const

const REFERRAL_TYPES: readonly PricingType[] = [
  "commission_pct",
  "commission_flat",
] as const

interface TypeRadioRowProps {
  allowedTypes: readonly PricingType[]
  value: PricingType
  onChange: (next: PricingType) => void
}

function TypeRadioRow({ allowedTypes, value, onChange }: TypeRadioRowProps) {
  const t = useTranslations("profile.pricing")
  return (
    <div
      role="radiogroup"
      aria-label={t("sectionTitle")}
      className="flex flex-wrap gap-2"
    >
      {allowedTypes.map((type) => {
        const isSelected = type === value
        return (
          <button
            key={type}
            type="button"
            role="radio"
            aria-checked={isSelected}
            onClick={() => onChange(type)}
            className={cn(
              "inline-flex items-center gap-1 rounded-full border px-3 py-1 text-xs font-medium transition-all duration-150",
              "focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2",
              isSelected
                ? "bg-primary text-primary-foreground border-primary"
                : "bg-background text-foreground border-border hover:border-primary/60 hover:bg-muted",
            )}
          >
            {t(typeKey(type))}
          </button>
        )
      })}
    </div>
  )
}

interface AmountInputProps {
  id: string
  label: string
  value: string
  suffix?: string
  onChange: (next: string) => void
}

function AmountInput({ id, label, value, suffix, onChange }: AmountInputProps) {
  return (
    <div>
      <label htmlFor={id} className="block text-xs font-medium text-foreground mb-1">
        {label}
      </label>
      <div className="relative">
        <input
          id={id}
          type="number"
          inputMode="decimal"
          min={0}
          step={0.01}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="w-full h-9 rounded-lg border border-border bg-background px-3 pr-8 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10 focus:outline-none"
        />
        {suffix ? (
          <span
            aria-hidden="true"
            className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-muted-foreground"
          >
            {suffix}
          </span>
        ) : null}
      </div>
    </div>
  )
}

// ----- Helpers ---------------------------------------------------------

function typeKey(type: PricingType): string {
  switch (type) {
    case "daily":
      return "typeDaily"
    case "hourly":
      return "typeHourly"
    case "project_from":
      return "typeProjectFrom"
    case "project_range":
      return "typeProjectRange"
    case "commission_pct":
      return "typeCommissionPct"
    case "commission_flat":
      return "typeCommissionFlat"
  }
}

function defaultCurrencyForType(type: PricingType): string {
  return type === "commission_pct" ? "pct" : "EUR"
}

// Converts a user-entered display value (e.g. "500" for 500€ or "5.5"
// for 5.5%) to the backend's canonical smallest unit (50000 or 550).
function storedFromDisplay(display: number, type: PricingType): number {
  if (type === "commission_pct") {
    return Math.round(display * 100)
  }
  return Math.round(display * 100)
}

function displayFromStored(stored: number, type: PricingType): string {
  if (type === "commission_pct") {
    return String(stored / 100)
  }
  return String(stored / 100)
}

function mapErrorToMessage(
  error: unknown,
  t: ReturnType<typeof useTranslations>,
): string {
  if (error instanceof ApiError) {
    if (error.code === "kind_not_allowed") return t("errorKindNotAllowed")
  }
  return t("errorGeneric")
}
