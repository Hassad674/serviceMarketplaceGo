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
import type { FreelancePricing } from "../api/freelance-profile-api"
import { useFreelancePricing } from "../hooks/use-freelance-pricing"
import { useUpsertFreelancePricing } from "../hooks/use-upsert-freelance-pricing"
import { useDeleteFreelancePricing } from "../hooks/use-delete-freelance-pricing"

// V1 pricing simplification: the freelance persona is narrowed down
// to a single allowed type — `daily` (TJM, the French market standard
// at ~95 %). The form no longer exposes a type dropdown or a currency
// picker; every freelance row is persisted as `daily` in EUR.
//
// Legacy rows (hourly / project_*) created before V1 remain readable
// through formatPricing so existing public profiles keep rendering
// correctly — only the editor is constrained.
const V1_PRICING_TYPE = "daily" as const
const V1_PRICING_CURRENCY = "EUR" as const

interface FreelancePricingSectionProps {
  readOnly?: boolean
}

// FreelancePricingSection owns the editable pricing card on /profile.
// Inline form (no modal) — the freelance persona has a single pricing
// row, so a full editor modal is unnecessary complexity.
export function FreelancePricingSection({
  readOnly = false,
}: FreelancePricingSectionProps) {
  const t = useTranslations("profile.pricing")
  const locale = useLocale() === "fr" ? "fr" : "en"
  const { data: persisted } = useFreelancePricing()
  const [editing, setEditing] = useState(false)

  if (readOnly && !persisted) return null

  return (
    <section
      aria-labelledby="freelance-pricing-section-title"
      className="bg-card border border-border rounded-xl p-6 shadow-sm"
    >
      <header className="mb-4 flex flex-col gap-1">
        <h2
          id="freelance-pricing-section-title"
          className="text-lg font-semibold text-foreground"
        >
          {t("directSectionTitle")}
        </h2>
        {!readOnly ? (
          <p className="text-sm text-muted-foreground">
            {t("directSectionSubtitle")}
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
  persisted: FreelancePricing | null
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
  persisted: FreelancePricing | null
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

// ----- Editable form ----------------------------------------------------

interface PricingFormProps {
  persisted: FreelancePricing | null
  locale: PricingLocale
  onClose: () => void
}

function PricingForm({ persisted, locale, onClose }: PricingFormProps) {
  const t = useTranslations("profile.pricing")
  const upsert = useUpsertFreelancePricing()
  const remove = useDeleteFreelancePricing()
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const draft = usePricingDraft(persisted)

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

  const preview = draft.toRow()

  return (
    <div className="rounded-xl border border-border bg-card p-4">
      {preview ? (
        <PricingPreviewStrip row={preview} locale={locale} />
      ) : null}

      <AmountInput
        id="freelance-pricing-amount"
        label={t("freelanceDailyLabel")}
        hint={t("freelanceDailyHint")}
        value={draft.amountDisplay}
        onChange={draft.setAmountDisplay}
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
  row: FreelancePricing
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

interface AmountInputProps {
  id: string
  label: string
  hint: string
  value: string
  onChange: (next: string) => void
}

function AmountInput({ id, label, hint, value, onChange }: AmountInputProps) {
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
          step={10}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          aria-describedby={hintId}
          className="w-full h-10 rounded-lg border border-border bg-background pl-3 pr-12 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10 focus:outline-none"
        />
        <span
          aria-hidden="true"
          className="absolute right-3 top-1/2 -translate-y-1/2 text-xs font-medium text-muted-foreground"
        >
          €/j
        </span>
      </div>
      <p id={hintId} className="mt-1 text-xs text-muted-foreground">
        {hint}
      </p>
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
    <div className="mt-3">
      <label
        htmlFor="freelance-pricing-note"
        className="block text-xs font-medium text-foreground mb-1"
      >
        {t("noteLabel")}
      </label>
      <input
        id="freelance-pricing-note"
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
  persisted: FreelancePricing | null
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
          data-testid="freelance-pricing-save"
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

// ----- Draft hook -------------------------------------------------------

interface PricingDraft {
  amountDisplay: string
  note: string
  negotiable: boolean
  setAmountDisplay: (next: string) => void
  setNote: (next: string) => void
  setNegotiable: (next: boolean) => void
  toRow: () => FreelancePricing | null
}

function usePricingDraft(persisted: FreelancePricing | null): PricingDraft {
  // V1 collapses the former type+currency picker into a single
  // required amount field. Legacy rows (hourly / project_*) are
  // silently coerced into the daily shape when the user re-edits —
  // the backend whitelist then persists the normalized row.
  const [amountDisplay, setAmountDisplay] = useState(
    persisted ? displayFromStored(persisted.min_amount) : "",
  )
  const [note, setNote] = useState(persisted?.note ?? "")
  const [negotiable, setNegotiable] = useState(persisted?.negotiable ?? false)

  const toRow = useMemo(
    () => (): FreelancePricing | null => {
      const amountNumber = Number(amountDisplay)
      if (!Number.isFinite(amountNumber) || amountNumber < 0) return null
      return {
        type: V1_PRICING_TYPE,
        min_amount: storedFromDisplay(amountNumber),
        max_amount: null,
        currency: V1_PRICING_CURRENCY,
        note: note.trim(),
        negotiable,
      }
    },
    [amountDisplay, note, negotiable],
  )

  return {
    amountDisplay,
    note,
    negotiable,
    setAmountDisplay,
    setNote,
    setNegotiable,
    toRow,
  }
}

// ----- helpers ----------------------------------------------------------

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
    if (err.code === "pricing_type_not_allowed") return t("errorTypeNotAllowed")
    if (err.code === "forbidden") return t("errorKindNotAllowed")
    if (err.code === "validation_error") return t("errorKindNotAllowed")
  }
  return t("errorGeneric")
}
