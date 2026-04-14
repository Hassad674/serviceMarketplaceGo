"use client"

import { useCallback, useState } from "react"
import { Check, Loader2, X } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import {
  LANGUAGE_OPTIONS,
  getLanguageLabel,
} from "@/shared/lib/profile/language-options"
import { useOrganizationShared } from "../hooks/use-organization-shared"
import { useUpdateOrganizationLanguages } from "../hooks/use-update-organization-languages"

type LanguageBucket = "professional" | "conversational"

// SharedLanguagesSection edits the two language arrays on the org row.
// Same pattern as the location section: derives draft during render
// from the cached server state and dispatches the mutation which
// invalidates both persona caches on success.
export function SharedLanguagesSection() {
  const t = useTranslations("profile.languages")
  const locale = useLocale() === "fr" ? "fr" : "en"
  const { data: shared } = useOrganizationShared()
  const mutation = useUpdateOrganizationLanguages()

  const persistedPro = shared?.languages_professional ?? []
  const persistedConv = shared?.languages_conversational ?? []
  const draft = useLanguagesDraft(persistedPro, persistedConv)

  const handleSave = useCallback(() => {
    mutation.mutate({
      professional: draft.professional,
      conversational: draft.conversational,
    })
  }, [draft.conversational, draft.professional, mutation])

  return (
    <section
      aria-labelledby="shared-languages-section-title"
      className="bg-card border border-border rounded-xl p-6 shadow-sm"
    >
      <header className="mb-5 flex flex-col gap-1">
        <h2
          id="shared-languages-section-title"
          className="text-lg font-semibold text-foreground"
        >
          {t("sectionTitle")}
        </h2>
        <p className="text-sm text-muted-foreground">{t("sectionSubtitle")}</p>
      </header>

      <div className="flex flex-col gap-6">
        <LanguageBucketEditor
          bucket="professional"
          selected={draft.professional}
          locale={locale}
          onAdd={(code) => draft.add("professional", code)}
          onRemove={(code) => draft.remove("professional", code)}
        />
        <div className="h-px bg-border" role="presentation" />
        <LanguageBucketEditor
          bucket="conversational"
          selected={draft.conversational}
          locale={locale}
          onAdd={(code) => draft.add("conversational", code)}
          onRemove={(code) => draft.remove("conversational", code)}
        />

        <SaveRow
          isDirty={draft.isDirty}
          isSaving={mutation.isPending}
          onSave={handleSave}
        />
      </div>
    </section>
  )
}

// ----- Bucket editor -----------------------------------------------------

interface LanguageBucketEditorProps {
  bucket: LanguageBucket
  selected: string[]
  locale: "fr" | "en"
  onAdd: (code: string) => void
  onRemove: (code: string) => void
}

function LanguageBucketEditor(props: LanguageBucketEditorProps) {
  const { bucket, selected, locale, onAdd, onRemove } = props
  const t = useTranslations("profile.languages")
  const labelKey =
    bucket === "professional" ? "professionalLabel" : "conversationalLabel"
  const helpKey =
    bucket === "professional" ? "professionalHelp" : "conversationalHelp"
  const available = LANGUAGE_OPTIONS.filter((opt) => !selected.includes(opt.code))

  return (
    <div>
      <h3 className="text-[15px] font-semibold text-foreground tracking-tight">
        {t(labelKey)}
      </h3>
      <p className="mb-3 text-xs text-muted-foreground">{t(helpKey)}</p>

      <SelectedLanguageChips
        codes={selected}
        locale={locale}
        onRemove={onRemove}
        bucketLabel={t(labelKey)}
      />

      <select
        aria-label={t(labelKey)}
        value=""
        onChange={(e) => {
          if (e.target.value) onAdd(e.target.value)
        }}
        className="mt-1 w-full max-w-xs h-9 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10 focus:outline-none"
      >
        <option value="">{t("searchPlaceholder")}</option>
        {available.map((opt) => (
          <option key={opt.code} value={opt.code}>
            {getLanguageLabel(opt.code, locale)}
          </option>
        ))}
      </select>

      <p className="mt-2 text-xs text-muted-foreground tabular-nums">
        {t("selectionCount", { count: selected.length })}
      </p>
    </div>
  )
}

interface SelectedLanguageChipsProps {
  codes: string[]
  locale: "fr" | "en"
  bucketLabel: string
  onRemove: (code: string) => void
}

function SelectedLanguageChips({
  codes,
  locale,
  bucketLabel,
  onRemove,
}: SelectedLanguageChipsProps) {
  const t = useTranslations("profile.languages")
  if (codes.length === 0) return null
  return (
    <ul className="mb-3 flex flex-wrap gap-2" aria-label={bucketLabel}>
      {codes.map((code) => {
        const label = getLanguageLabel(code, locale)
        return (
          <li key={code}>
            <span
              className={cn(
                "inline-flex items-center gap-1.5 rounded-full",
                "bg-primary/10 text-primary border border-primary/20",
                "px-2.5 py-1 text-xs font-medium",
              )}
            >
              {label}
              <button
                type="button"
                onClick={() => onRemove(code)}
                aria-label={`${t("remove")} ${label}`}
                className="inline-flex h-4 w-4 items-center justify-center rounded-full hover:bg-primary/20 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
              >
                <X className="h-3 w-3" aria-hidden="true" />
              </button>
            </span>
          </li>
        )
      })}
    </ul>
  )
}

// ----- Draft hook --------------------------------------------------------

interface LanguagesDraft {
  professional: string[]
  conversational: string[]
  isDirty: boolean
  add: (bucket: LanguageBucket, code: string) => void
  remove: (bucket: LanguageBucket, code: string) => void
}

function useLanguagesDraft(
  persistedPro: string[],
  persistedConv: string[],
): LanguagesDraft {
  const proKey = persistedPro.join(",")
  const convKey = persistedConv.join(",")
  const [prevKey, setPrevKey] = useState(`${proKey}|${convKey}`)
  const [professional, setProfessional] = useState(persistedPro)
  const [conversational, setConversational] = useState(persistedConv)

  const currentKey = `${proKey}|${convKey}`
  if (prevKey !== currentKey) {
    setPrevKey(currentKey)
    setProfessional(persistedPro)
    setConversational(persistedConv)
  }

  const add = (bucket: LanguageBucket, code: string) => {
    if (!code) return
    const addTarget =
      bucket === "professional" ? setProfessional : setConversational
    const removeTarget =
      bucket === "professional" ? setConversational : setProfessional
    addTarget((current) =>
      current.includes(code) ? current : [...current, code],
    )
    removeTarget((current) => current.filter((c) => c !== code))
  }

  const remove = (bucket: LanguageBucket, code: string) => {
    const target =
      bucket === "professional" ? setProfessional : setConversational
    target((current) => current.filter((c) => c !== code))
  }

  const isDirty =
    professional.join(",") !== proKey || conversational.join(",") !== convKey

  return { professional, conversational, isDirty, add, remove }
}

interface SaveRowProps {
  isDirty: boolean
  isSaving: boolean
  onSave: () => void
}

function SaveRow({ isDirty, isSaving, onSave }: SaveRowProps) {
  const t = useTranslations("profile.languages")
  return (
    <div className="flex items-center justify-end">
      <button
        type="button"
        onClick={onSave}
        disabled={!isDirty || isSaving}
        className={cn(
          "bg-primary text-primary-foreground rounded-md h-9 px-4 text-sm font-medium",
          "hover:opacity-90 transition-opacity duration-150",
          "focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2",
          "disabled:opacity-50 disabled:cursor-not-allowed",
          "inline-flex items-center gap-2",
        )}
      >
        {isSaving ? (
          <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />
        ) : (
          <Check className="w-4 h-4" aria-hidden="true" />
        )}
        {isSaving ? t("saving") : t("save")}
      </button>
    </div>
  )
}
