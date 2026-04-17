"use client"

import { useCallback, useMemo, useState } from "react"
import { Check, Loader2, Pencil, Trash2 } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import { ApiError } from "@/shared/lib/api-client"
import { cn } from "@/shared/lib/utils"
import {
  formatPricing,
  type PricingLocale,
} from "@/shared/lib/profile/pricing-format"
import type { ReferrerPricing } from "../api/referrer-profile-api"
import { useReferrerPricing } from "../hooks/use-referrer-pricing"
import { useUpsertReferrerPricing } from "../hooks/use-upsert-referrer-pricing"
import { useDeleteReferrerPricing } from "../hooks/use-delete-referrer-pricing"

// V1 pricing simplification: the referrer persona is narrowed down to
// a single allowed type — `commission_pct` (the B2B referral
// convention). The form now asks for ONE percentage in [0..100];
// internally we still persist a range-shaped row (the backend
// validator requires it) by echoing the same value to min and max.
//
// Legacy commission_flat rows remain readable; only writes are
// constrained.
const V1_PRICING_TYPE = "commission_pct" as const
const V1_PRICING_CURRENCY = "pct" as const
const COMMISSION_PCT_MAX = 100

interface ReferrerPricingSectionProps {
  readOnly?: boolean
}

// ReferrerPricingSection owns the commission pricing card on
// /referral. Commission-only — the form asks for ONE percentage and
// the backend persists a commission_pct row with the same value on
// both min and max (validator requires a range).
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

  return (
    <div className="rounded-xl border border-border bg-card p-4">
      {preview ? (
        <PricingPreviewStrip row={preview} locale={locale} />
      ) : null}

      <CommissionInput
        label={t("referrerCommissionLabel")}
        hint={t("referrerCommissionHint")}
        value={draft.pctDisplay}
        onChange={draft.setPctDisplay}
      />

      <NoteField value={draft.note} onChange={draft.setNote} />

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

interface CommissionInputProps {
  label: string
  hint: string
  value: string
  onChange: (next: string) => void
}

function CommissionInput({
  label,
  hint,
  value,
  onChange,
}: CommissionInputProps) {
  const id = "referrer-pricing-pct"
  const hintId = `${id}-hint`
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
          max={COMMISSION_PCT_MAX}
          step={0.5}
          value={value}
          onChange={(e) => onChange(clampPctInput(e.target.value))}
          aria-describedby={hintId}
          className="w-full h-10 rounded-lg border border-border bg-background pl-3 pr-10 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10 focus:outline-none"
        />
        <span
          aria-hidden="true"
          className="absolute right-3 top-1/2 -translate-y-1/2 text-xs font-medium text-muted-foreground"
        >
          %
        </span>
      </div>
      <p id={hintId} className="mt-1 text-xs text-muted-foreground">
        {hint}
      </p>
    </div>
  )
}

// clampPctInput constrains the free-form input to [0..100]. We do this
// on change rather than on blur so the preview strip tracks the final
// value without needing a separate "sanitize" pass.
function clampPctInput(raw: string): string {
  if (raw === "") return ""
  const n = Number(raw)
  if (!Number.isFinite(n)) return ""
  if (n < 0) return "0"
  if (n > COMMISSION_PCT_MAX) return String(COMMISSION_PCT_MAX)
  return raw
}

interface NoteFieldProps {
  value: string
  onChange: (next: string) => void
}

function NoteField({ value, onChange }: NoteFieldProps) {
  const t = useTranslations("profile.pricing")
  return (
    <div className="mt-3">
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
        className="w-full h-10 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10 focus:outline-none"
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
  pctDisplay: string
  note: string
  negotiable: boolean
  setPctDisplay: (next: string) => void
  setNote: (next: string) => void
  setNegotiable: (next: boolean) => void
  toRow: () => ReferrerPricing | null
}

function usePricingDraft(persisted: ReferrerPricing | null): PricingDraft {
  // V1 collapses the former type+currency picker into a single
  // commission percentage. Legacy commission_flat rows are coerced
  // into commission_pct on re-edit — the user enters a fresh
  // percentage, the flat-fee value is discarded. The backend
  // whitelist persists the normalized row.
  const [pctDisplay, setPctDisplay] = useState(initialPctDisplay(persisted))
  const [note, setNote] = useState(persisted?.note ?? "")
  const [negotiable, setNegotiable] = useState(persisted?.negotiable ?? false)

  const toRow = useMemo(
    () => (): ReferrerPricing | null => {
      const pct = Number(pctDisplay)
      if (!Number.isFinite(pct) || pct < 0 || pct > COMMISSION_PCT_MAX) {
        return null
      }
      // basis points = percent * 100 (5.5 % -> 550)
      const basisPoints = Math.round(pct * 100)
      return {
        type: V1_PRICING_TYPE,
        // The commission_pct row shape is a range (min..max) even when
        // the user submits a single value — we echo the value to both
        // bounds so the backend validator accepts the payload and the
        // formatter renders a single "N %" instead of "N – N %".
        min_amount: basisPoints,
        max_amount: basisPoints,
        currency: V1_PRICING_CURRENCY,
        note: note.trim(),
        negotiable,
      }
    },
    [pctDisplay, note, negotiable],
  )

  return {
    pctDisplay,
    note,
    negotiable,
    setPctDisplay,
    setNote,
    setNegotiable,
    toRow,
  }
}

// initialPctDisplay chooses the best single-percent default to surface
// in the input. commission_pct rows use min_amount for the lower
// bound, so we prefer max_amount (the user's "headline" rate) when
// available; legacy commission_flat rows have no percent, so we fall
// back to an empty field and let the user type a fresh value.
function initialPctDisplay(persisted: ReferrerPricing | null): string {
  if (!persisted) return ""
  if (persisted.type !== "commission_pct") return ""
  const basisPoints = persisted.max_amount ?? persisted.min_amount
  return String(basisPoints / 100)
}

function mapErrorMessage(
  err: unknown,
  t: ReturnType<typeof useTranslations>,
): string {
  if (err instanceof ApiError) {
    if (err.code === "pricing_type_not_allowed") return t("errorTypeNotAllowed")
    if (err.code === "forbidden") return t("errorKindNotAllowed")
    if (err.code === "validation_error") return t("errorKindNotAllowed")
  }
  return t("errorGeneric")
}
