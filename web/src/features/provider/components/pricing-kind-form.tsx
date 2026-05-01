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
import { Button } from "@/shared/components/ui/button"

interface PricingKindFormProps {
  kind: PricingKind
  orgType: string | undefined
  referrerEnabled: boolean | undefined
  persisted: Pricing | null
  onSave: (row: Pricing) => Promise<void>
  onDelete: () => Promise<void>
  onPreviewChange: (row: Pricing | null) => void
}

// Form for one pricing kind (direct OR referral). Kind and org
// type together determine the allowed pricing types — e.g. an
// agency on the direct variant only has project_from and
// project_range (no daily / hourly).
export function PricingKindForm(props: PricingKindFormProps) {
  const {
    kind,
    orgType,
    referrerEnabled,
    persisted,
    onSave,
    onDelete,
    onPreviewChange,
  } = props

  const t = useTranslations("profile.pricing")
  const allowedTypes = useMemo(
    () => computeAllowedTypes(kind, orgType, referrerEnabled),
    [kind, orgType, referrerEnabled],
  )

  const initialType = pickInitialType(persisted?.type, allowedTypes)
  const initialCurrency = persisted?.currency ?? defaultCurrencyForType(initialType)

  const [type, setType] = useState<PricingType>(initialType)
  const [currencyRaw, setCurrencyRaw] = useState<string>(initialCurrency)
  const currency =
    type === "commission_pct" ? "pct" : currencyRaw === "pct" ? "EUR" : currencyRaw
  const [minDisplay, setMinDisplay] = useState<string>(
    persisted ? displayFromStored(persisted.min_amount) : "",
  )
  const [maxDisplay, setMaxDisplay] = useState<string>(
    persisted && persisted.max_amount !== null
      ? displayFromStored(persisted.max_amount)
      : "",
  )
  const [note, setNote] = useState<string>(persisted?.note ?? "")
  const [negotiable, setNegotiable] = useState<boolean>(
    persisted?.negotiable ?? false,
  )
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
      min_amount: storedFromDisplay(minNumber),
      max_amount:
        hasRange && maxNumber !== null ? storedFromDisplay(maxNumber) : null,
      currency,
      note: note.trim(),
      negotiable,
    }
  }, [currency, kind, maxDisplay, minDisplay, note, negotiable, type])

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
      await onDelete()
      setMinDisplay("")
      setMaxDisplay("")
      setNote("")
      setNegotiable(false)
    } catch (caught) {
      setErrorMessage(mapErrorToMessage(caught, t))
    } finally {
      setIsDeleting(false)
    }
  }, [onDelete, t])

  const showMax = type === "project_range" || type === "commission_pct"
  const isPct = type === "commission_pct"

  return (
    <section
      aria-label={kind === "direct" ? t("kindDirect") : t("kindReferral")}
      className="rounded-xl border border-border bg-card p-4"
    >
      <div className="mb-3 flex items-center justify-between">
        <h3 className="text-base font-semibold text-foreground">
          {kind === "direct" ? t("kindDirect") : t("kindReferral")}
        </h3>
        {persisted ? (
          <Button variant="ghost" size="auto"
            type="button"
            onClick={handleDelete}
            disabled={isDeleting}
            className="inline-flex items-center gap-1.5 rounded-md h-8 px-3 text-xs font-medium text-destructive hover:bg-destructive/10 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50"
          >
            <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
            {t("delete")}
          </Button>
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
            onChange={(e) => setCurrencyRaw(e.target.value)}
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

      <NegotiableRow kind={kind} value={negotiable} onChange={setNegotiable} />

      {errorMessage ? (
        <p
          role="alert"
          className="mt-3 rounded-md border border-destructive/20 bg-destructive/5 px-3 py-2 text-sm text-destructive"
        >
          {errorMessage}
        </p>
      ) : null}

      <div className="mt-3 flex justify-end">
        <Button variant="ghost" size="auto"
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
        </Button>
      </div>
    </section>
  )
}

// ----- Sub-components ---------------------------------------------------

// V1 pricing simplification: every persona is narrowed to a single
// allowed type, so each bucket below is now length 1. The type radio
// hides itself when only one option is present (see TypeRadioRow),
// turning the form into a single-field experience.
//
// The six-value PricingType union is retained — legacy rows still
// round-trip through this form as-is, and we never want the editor
// to crash on a persona/kind combination we decommissioned.
const DIRECT_TYPES: readonly PricingType[] = ["project_from"] as const

const AGENCY_DIRECT_TYPES: readonly PricingType[] = ["project_from"] as const

const REFERRAL_TYPES: readonly PricingType[] = ["commission_pct"] as const

// computeAllowedTypes mirrors the backend V1 whitelist: each
// (kind, org) triplet narrows to the single allowed type. The
// pricing-kind-form is shared across agency + provider_personal
// (for the direct kind) and any org with referrer_enabled (for the
// referral kind), so the single-type rule applies uniformly.
function computeAllowedTypes(
  kind: PricingKind,
  orgType: string | undefined,
  _referrerEnabled: boolean | undefined,
): readonly PricingType[] {
  if (kind === "referral") return REFERRAL_TYPES
  if (orgType === "agency") return AGENCY_DIRECT_TYPES
  return DIRECT_TYPES
}

// pickInitialType falls back to the first allowed type when the
// persisted type was removed (e.g. an agency migrating from a
// legacy daily row to the new outcome-only policy).
function pickInitialType(
  persistedType: PricingType | undefined,
  allowed: readonly PricingType[],
): PricingType {
  if (persistedType && allowed.includes(persistedType)) return persistedType
  return allowed[0] ?? "daily"
}

interface TypeRadioRowProps {
  allowedTypes: readonly PricingType[]
  value: PricingType
  onChange: (next: PricingType) => void
}

function TypeRadioRow({ allowedTypes, value, onChange }: TypeRadioRowProps) {
  const t = useTranslations("profile.pricing")
  // V1: when only one type is allowed, the radio is pure noise — hide
  // it and let the single-field form speak for itself. The parent
  // already initialises `value` to the single allowed type, so
  // skipping the picker does not change the submitted payload.
  if (allowedTypes.length <= 1) return null
  return (
    <div
      role="radiogroup"
      aria-label={t("typeGroupLabel")}
      className="flex flex-wrap gap-2"
    >
      {allowedTypes.map((type) => {
        const isSelected = type === value
        return (
          <Button variant="ghost" size="auto"
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
          </Button>
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

interface NegotiableRowProps {
  kind: PricingKind
  value: boolean
  onChange: (next: boolean) => void
}

function NegotiableRow({ kind, value, onChange }: NegotiableRowProps) {
  const t = useTranslations("profile.pricing")
  return (
    <div className="mt-3">
      <p className="text-xs font-medium text-foreground mb-1">
        {t("negotiableLabel")}
      </p>
      <div
        role="radiogroup"
        aria-label={t("negotiableLabel")}
        className="flex flex-wrap gap-2"
      >
        {([true, false] as const).map((option) => {
          const isSelected = option === value
          const labelKey = option ? "negotiableYes" : "negotiableNo"
          return (
            <Button variant="ghost" size="auto"
              key={String(option)}
              type="button"
              role="radio"
              aria-checked={isSelected}
              onClick={() => onChange(option)}
              data-testid={`pricing-${kind}-negotiable-${option ? "yes" : "no"}`}
              className={cn(
                "inline-flex items-center gap-1 rounded-full border px-3 py-1 text-xs font-medium transition-all duration-150",
                "focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2",
                isSelected
                  ? "bg-primary text-primary-foreground border-primary"
                  : "bg-background text-foreground border-border hover:border-primary/60 hover:bg-muted",
              )}
            >
              {t(labelKey)}
            </Button>
          )
        })}
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
// The mapping is identical for every type we currently support, so
// the signature no longer needs the pricing type.
function storedFromDisplay(display: number): number {
  return Math.round(display * 100)
}

function displayFromStored(stored: number): string {
  return String(stored / 100)
}

function mapErrorToMessage(
  error: unknown,
  t: ReturnType<typeof useTranslations>,
): string {
  if (error instanceof ApiError) {
    if (error.code === "kind_not_allowed") return t("errorKindNotAllowed")
    if (error.code === "forbidden") return t("errorKindNotAllowed")
    if (error.code === "validation_error") return t("errorKindNotAllowed")
  }
  return t("errorGeneric")
}
