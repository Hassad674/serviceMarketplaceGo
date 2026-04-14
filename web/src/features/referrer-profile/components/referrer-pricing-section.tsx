"use client"

import { useCallback, useMemo, useState } from "react"
import { Check, Loader2, Pencil, Trash2 } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import { ApiError } from "@/shared/lib/api-client"
import { cn } from "@/shared/lib/utils"
import {
  formatPricing,
  SUPPORTED_FIAT_CURRENCIES,
  type PricingLocale,
} from "@/shared/lib/profile/pricing-format"
import type {
  ReferrerPricing,
  ReferrerPricingType,
} from "../api/referrer-profile-api"
import { useReferrerPricing } from "../hooks/use-referrer-pricing"
import { useUpsertReferrerPricing } from "../hooks/use-upsert-referrer-pricing"
import { useDeleteReferrerPricing } from "../hooks/use-delete-referrer-pricing"

const ALLOWED_TYPES: readonly ReferrerPricingType[] = [
  "commission_pct",
  "commission_flat",
] as const

interface ReferrerPricingSectionProps {
  readOnly?: boolean
}

// ReferrerPricingSection owns the commission pricing card on
// /referral. Commission-only — the type union excludes daily/hourly
// and the UI reflects that by forcing the "%" suffix on commission_pct
// and swapping the currency dropdown accordingly.
export function ReferrerPricingSection({
  readOnly = false,
}: ReferrerPricingSectionProps) {
  const t = useTranslations("profile.pricing")
  const locale = useLocale() === "fr" ? "fr" : "en"
  const { data: persisted } = useReferrerPricing()
  const [editing, setEditing] = useState(false)

  if (readOnly && !persisted) return null

  return (
    <section
      aria-labelledby="referrer-pricing-section-title"
      className="bg-card border border-border rounded-xl p-6 shadow-sm"
    >
      <header className="mb-4 flex flex-col gap-1">
        <h2
          id="referrer-pricing-section-title"
          className="text-lg font-semibold text-foreground"
        >
          {t("referralSectionTitle")}
        </h2>
        {!readOnly ? (
          <p className="text-sm text-muted-foreground">
            {t("referralSectionSubtitle")}
          </p>
        ) : null}
      </header>

      {editing && !readOnly ? (
        <PricingForm
          persisted={persisted ?? null}
          locale={locale}
          onClose={() => setEditing(false)}
        />
      ) : (
        <ReadView
          persisted={persisted ?? null}
          locale={locale}
          readOnly={readOnly}
          onEdit={() => setEditing(true)}
        />
      )}
    </section>
  )
}

interface ReadViewProps {
  persisted: ReferrerPricing | null
  locale: PricingLocale
  readOnly: boolean
  onEdit: () => void
}

function ReadView({ persisted, locale, readOnly, onEdit }: ReadViewProps) {
  const t = useTranslations("profile.pricing")
  return (
    <div>
      <PricingDisplayRow persisted={persisted} locale={locale} />
      {!readOnly ? (
        <button
          type="button"
          onClick={onEdit}
          className="mt-4 inline-flex items-center gap-2 rounded-md border border-border h-9 px-4 text-sm font-medium text-foreground hover:bg-muted focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
        >
          <Pencil className="h-4 w-4" aria-hidden="true" />
          {t("editButton")}
        </button>
      ) : null}
    </div>
  )
}

interface DisplayRowProps {
  persisted: ReferrerPricing | null
  locale: PricingLocale
}

function PricingDisplayRow({ persisted, locale }: DisplayRowProps) {
  const t = useTranslations("profile.pricing")
  if (!persisted) {
    return (
      <p className="text-sm italic text-muted-foreground">{t("empty")}</p>
    )
  }
  return (
    <div className="inline-flex flex-col rounded-lg border border-border bg-muted/30 px-3 py-2 text-sm">
      <span className="font-semibold text-foreground">
        {formatPricing(persisted, locale)}
      </span>
      <div className="mt-1 flex flex-wrap items-center gap-2">
        {persisted.negotiable ? (
          <span className="inline-flex items-center rounded-full border border-primary/30 bg-primary/10 px-2 py-0.5 text-[11px] font-medium text-primary">
            {t("negotiableBadge")}
          </span>
        ) : null}
        {persisted.note ? (
          <span className="text-xs text-muted-foreground italic truncate max-w-[220px]">
            {persisted.note}
          </span>
        ) : null}
      </div>
    </div>
  )
}

// ----- Editable form -----------------------------------------------------

interface PricingFormProps {
  persisted: ReferrerPricing | null
  locale: PricingLocale
  onClose: () => void
}

function PricingForm({ persisted, locale, onClose }: PricingFormProps) {
  const t = useTranslations("profile.pricing")
  const upsert = useUpsertReferrerPricing()
  const remove = useDeleteReferrerPricing()
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const draft = usePricingDraft(persisted)
  const preview = draft.toRow()

  const handleSave = useCallback(async () => {
    setErrorMessage(null)
    const row = draft.toRow()
    if (!row) {
      setErrorMessage(t("errorGeneric"))
      return
    }
    try {
      await upsert.mutateAsync(row)
      onClose()
    } catch (caught) {
      setErrorMessage(mapErrorMessage(caught, t))
    }
  }, [draft, onClose, t, upsert])

  const handleDelete = useCallback(async () => {
    setErrorMessage(null)
    try {
      await remove.mutateAsync()
      onClose()
    } catch (caught) {
      setErrorMessage(mapErrorMessage(caught, t))
    }
  }, [onClose, remove, t])

  const showMax = draft.type === "commission_pct"
  const isPct = draft.type === "commission_pct"

  return (
    <div className="rounded-xl border border-border bg-card p-4">
      {preview ? (
        <PricingPreviewStrip row={preview} locale={locale} />
      ) : null}

      <PricingTypeRadio
        value={draft.type}
        onChange={(next) => draft.setType(next)}
      />

      <div className="mt-3 grid gap-3 sm:grid-cols-2">
        <AmountInput
          id="referrer-pricing-min"
          label={t("minAmountLabel")}
          value={draft.minDisplay}
          onChange={(next) => draft.setMinDisplay(next)}
          suffix={isPct ? "%" : undefined}
        />
        {showMax ? (
          <AmountInput
            id="referrer-pricing-max"
            label={t("maxAmountLabel")}
            value={draft.maxDisplay}
            onChange={(next) => draft.setMaxDisplay(next)}
            suffix="%"
          />
        ) : null}
      </div>

      <div className="mt-3 grid gap-3 sm:grid-cols-2">
        <CurrencyField
          value={draft.currency}
          disabled={isPct}
          onChange={draft.setCurrency}
        />
        <NoteField value={draft.note} onChange={draft.setNote} />
      </div>

      <NegotiableRow value={draft.negotiable} onChange={draft.setNegotiable} />

      {errorMessage ? (
        <p
          role="alert"
          className="mt-3 rounded-md border border-destructive/20 bg-destructive/5 px-3 py-2 text-sm text-destructive"
        >
          {errorMessage}
        </p>
      ) : null}

      <FormActions
        persisted={persisted}
        isSaving={upsert.isPending}
        isDeleting={remove.isPending}
        onSave={handleSave}
        onDelete={handleDelete}
        onCancel={onClose}
      />
    </div>
  )
}

interface PricingPreviewStripProps {
  row: ReferrerPricing
  locale: PricingLocale
}

function PricingPreviewStrip({ row, locale }: PricingPreviewStripProps) {
  const t = useTranslations("profile.pricing")
  return (
    <div className="mb-4 rounded-md border border-border bg-muted/30 px-3 py-2">
      <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground mb-1">
        {t("previewHeading")}
      </p>
      <span className="font-semibold text-foreground text-sm">
        {formatPricing(row, locale)}
      </span>
    </div>
  )
}

interface PricingTypeRadioProps {
  value: ReferrerPricingType
  onChange: (next: ReferrerPricingType) => void
}

function PricingTypeRadio({ value, onChange }: PricingTypeRadioProps) {
  const t = useTranslations("profile.pricing")
  return (
    <div
      role="radiogroup"
      aria-label={t("typeGroupLabel")}
      className="flex flex-wrap gap-2"
    >
      {ALLOWED_TYPES.map((type) => {
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
            {t(typeLabelKey(type))}
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

function AmountInput({
  id,
  label,
  value,
  suffix,
  onChange,
}: AmountInputProps) {
  return (
    <div>
      <label
        htmlFor={id}
        className="block text-xs font-medium text-foreground mb-1"
      >
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

interface CurrencyFieldProps {
  value: string
  disabled: boolean
  onChange: (next: string) => void
}

function CurrencyField({ value, disabled, onChange }: CurrencyFieldProps) {
  const t = useTranslations("profile.pricing")
  return (
    <div>
      <label
        htmlFor="referrer-pricing-currency"
        className="block text-xs font-medium text-foreground mb-1"
      >
        {t("currencyLabel")}
      </label>
      <select
        id="referrer-pricing-currency"
        value={disabled ? "pct" : value}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
        className="w-full h-9 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10 focus:outline-none disabled:opacity-60"
      >
        {disabled ? (
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
  )
}

interface NoteFieldProps {
  value: string
  onChange: (next: string) => void
}

function NoteField({ value, onChange }: NoteFieldProps) {
  const t = useTranslations("profile.pricing")
  return (
    <div>
      <label
        htmlFor="referrer-pricing-note"
        className="block text-xs font-medium text-foreground mb-1"
      >
        {t("noteLabel")}
      </label>
      <input
        id="referrer-pricing-note"
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={t("notePlaceholder")}
        maxLength={120}
        className="w-full h-9 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10 focus:outline-none"
      />
    </div>
  )
}

interface NegotiableRowProps {
  value: boolean
  onChange: (next: boolean) => void
}

function NegotiableRow({ value, onChange }: NegotiableRowProps) {
  const t = useTranslations("profile.pricing")
  return (
    <div className="mt-3">
      <p className="text-xs font-medium text-foreground mb-1">
        {t("negotiableLabel")}
      </p>
      <div role="radiogroup" className="flex flex-wrap gap-2">
        {([true, false] as const).map((option) => {
          const isSelected = option === value
          const labelKey = option ? "negotiableYes" : "negotiableNo"
          return (
            <button
              key={String(option)}
              type="button"
              role="radio"
              aria-checked={isSelected}
              onClick={() => onChange(option)}
              className={cn(
                "inline-flex items-center gap-1 rounded-full border px-3 py-1 text-xs font-medium transition-all duration-150",
                isSelected
                  ? "bg-primary text-primary-foreground border-primary"
                  : "bg-background text-foreground border-border hover:border-primary/60 hover:bg-muted",
              )}
            >
              {t(labelKey)}
            </button>
          )
        })}
      </div>
    </div>
  )
}

interface FormActionsProps {
  persisted: ReferrerPricing | null
  isSaving: boolean
  isDeleting: boolean
  onSave: () => void
  onDelete: () => void
  onCancel: () => void
}

function FormActions(props: FormActionsProps) {
  const {
    persisted,
    isSaving,
    isDeleting,
    onSave,
    onDelete,
    onCancel,
  } = props
  const t = useTranslations("profile.pricing")
  const tCommon = useTranslations("common")
  return (
    <div className="mt-4 flex items-center justify-between gap-2">
      {persisted ? (
        <button
          type="button"
          onClick={onDelete}
          disabled={isDeleting}
          className="inline-flex items-center gap-1.5 rounded-md h-9 px-3 text-xs font-medium text-destructive hover:bg-destructive/10 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50"
        >
          <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
          {t("delete")}
        </button>
      ) : (
        <span />
      )}
      <div className="flex items-center gap-2">
        <button
          type="button"
          onClick={onCancel}
          className="rounded-md h-9 px-4 text-sm font-medium text-foreground hover:bg-muted focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
        >
          {tCommon("cancel")}
        </button>
        <button
          type="button"
          onClick={onSave}
          disabled={isSaving}
          data-testid="referrer-pricing-save"
          className="bg-primary text-primary-foreground rounded-md h-9 px-4 text-sm font-medium hover:opacity-90 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50 inline-flex items-center gap-2"
        >
          {isSaving ? (
            <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />
          ) : (
            <Check className="w-4 h-4" aria-hidden="true" />
          )}
          {isSaving ? t("saving") : t("save")}
        </button>
      </div>
    </div>
  )
}

// ----- Draft hook ------------------------------------------------------

interface PricingDraft {
  type: ReferrerPricingType
  minDisplay: string
  maxDisplay: string
  currency: string
  note: string
  negotiable: boolean
  setType: (next: ReferrerPricingType) => void
  setMinDisplay: (next: string) => void
  setMaxDisplay: (next: string) => void
  setCurrency: (next: string) => void
  setNote: (next: string) => void
  setNegotiable: (next: boolean) => void
  toRow: () => ReferrerPricing | null
}

function usePricingDraft(persisted: ReferrerPricing | null): PricingDraft {
  const [type, setType] = useState<ReferrerPricingType>(
    (persisted?.type as ReferrerPricingType) ?? "commission_pct",
  )
  const [minDisplay, setMinDisplay] = useState(
    persisted ? displayFromStored(persisted.min_amount) : "",
  )
  const [maxDisplay, setMaxDisplay] = useState(
    persisted && persisted.max_amount !== null
      ? displayFromStored(persisted.max_amount)
      : "",
  )
  const defaultCurrency = persisted?.currency ?? defaultCurrencyForType(type)
  const [currencyRaw, setCurrencyRaw] = useState(defaultCurrency)
  const currency = type === "commission_pct" ? "pct" : currencyRaw === "pct" ? "EUR" : currencyRaw
  const [note, setNote] = useState(persisted?.note ?? "")
  const [negotiable, setNegotiable] = useState(persisted?.negotiable ?? false)

  const toRow = useMemo(
    () => (): ReferrerPricing | null => {
      const minNumber = Number(minDisplay)
      if (!Number.isFinite(minNumber) || minNumber < 0) return null
      const hasRange = type === "commission_pct"
      const maxNumber = maxDisplay.trim() === "" ? null : Number(maxDisplay)
      if (hasRange && (maxNumber === null || !Number.isFinite(maxNumber))) {
        return null
      }
      return {
        type,
        min_amount: storedFromDisplay(minNumber),
        max_amount:
          hasRange && maxNumber !== null ? storedFromDisplay(maxNumber) : null,
        currency,
        note: note.trim(),
        negotiable,
      }
    },
    [currency, maxDisplay, minDisplay, note, negotiable, type],
  )

  return {
    type,
    minDisplay,
    maxDisplay,
    currency,
    note,
    negotiable,
    setType,
    setMinDisplay,
    setMaxDisplay,
    setCurrency: setCurrencyRaw,
    setNote,
    setNegotiable,
    toRow,
  }
}

// ----- helpers ---------------------------------------------------------

function typeLabelKey(type: ReferrerPricingType): string {
  switch (type) {
    case "commission_pct":
      return "typeCommissionPct"
    case "commission_flat":
      return "typeCommissionFlat"
  }
}

function defaultCurrencyForType(type: ReferrerPricingType): string {
  return type === "commission_pct" ? "pct" : "EUR"
}

function storedFromDisplay(display: number): number {
  return Math.round(display * 100)
}

function displayFromStored(stored: number): string {
  return String(stored / 100)
}

function mapErrorMessage(
  err: unknown,
  t: ReturnType<typeof useTranslations>,
): string {
  if (err instanceof ApiError) {
    if (err.code === "forbidden") return t("errorKindNotAllowed")
    if (err.code === "validation_error") return t("errorKindNotAllowed")
  }
  return t("errorGeneric")
}
