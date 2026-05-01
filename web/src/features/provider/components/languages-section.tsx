"use client"

import { useCallback, useId, useState } from "react"
import { Check, Globe, Loader2, X } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { getLanguageLabel } from "../lib/language-options"
import { useProfile } from "../hooks/use-profile"
import { useUpdateLanguages } from "../hooks/use-update-languages"
import {
  LanguageCombobox,
  type LanguageComboboxLocale,
} from "./language-combobox"

import { Button } from "@/shared/components/ui/button"
type LanguageBucket = "professional" | "conversational"

interface LanguagesSectionProps {
  orgType: string | undefined
  readOnly?: boolean
}

export function LanguagesSection({
  orgType,
  readOnly = false,
}: LanguagesSectionProps) {
  const t = useTranslations("profile.languages")
  const locale: LanguageComboboxLocale = useLocale() === "fr" ? "fr" : "en"
  const draft = useLanguagesDraft(locale)

  if (orgType === "enterprise") return null
  if (readOnly) return null

  return (
    <section
      aria-labelledby="languages-section-title"
      className="bg-card border border-border rounded-xl p-6 shadow-sm"
    >
      <header className="mb-5 flex flex-col gap-1">
        <h2
          id="languages-section-title"
          className="text-lg font-semibold text-foreground"
        >
          {t("sectionTitle")}
        </h2>
        <p className="text-sm text-muted-foreground">{t("sectionSubtitle")}</p>
      </header>
      <div
        aria-live="polite"
        aria-atomic="true"
        className="sr-only"
        role="status"
      >
        {draft.liveMessage}
      </div>
      <div className="flex flex-col gap-6">
        <LanguageBucketEditor bucket="professional" locale={locale} draft={draft} />
        <div className="h-px bg-border" role="presentation" />
        <LanguageBucketEditor
          bucket="conversational"
          locale={locale}
          draft={draft}
        />
        <LanguagesSaveButton
          isDirty={draft.isDirty}
          isPending={draft.isSaving}
          onSave={draft.save}
        />
      </div>
    </section>
  )
}

interface LanguagesSaveButtonProps {
  isDirty: boolean
  isPending: boolean
  onSave: () => void
}

function LanguagesSaveButton({
  isDirty,
  isPending,
  onSave,
}: LanguagesSaveButtonProps) {
  const t = useTranslations("profile.languages")
  return (
    <div className="flex items-center justify-end">
      <Button variant="ghost" size="auto"
        type="button"
        onClick={onSave}
        disabled={!isDirty || isPending}
        className={cn(
          "bg-primary text-primary-foreground rounded-md h-9 px-4 text-sm font-medium",
          "hover:opacity-90 transition-opacity duration-150",
          "focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2",
          "disabled:opacity-50 disabled:cursor-not-allowed",
          "inline-flex items-center gap-2",
        )}
      >
        {isPending ? (
          <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />
        ) : (
          <Check className="w-4 h-4" aria-hidden="true" />
        )}
        {isPending ? t("saving") : t("save")}
      </Button>
    </div>
  )
}

interface LanguagesDraft {
  professional: string[]
  conversational: string[]
  liveMessage: string
  isDirty: boolean
  isSaving: boolean
  add: (bucket: LanguageBucket, code: string) => void
  remove: (bucket: LanguageBucket, code: string) => void
  clear: (bucket: LanguageBucket) => void
  save: () => void
}

function useLanguagesDraft(locale: LanguageComboboxLocale): LanguagesDraft {
  const t = useTranslations("profile.languages")
  const { data: profile } = useProfile()
  const mutation = useUpdateLanguages()
  const buckets = useBucketsSyncedWithProfile(profile)
  const [liveMessage, setLiveMessage] = useState("")

  const announce = useCallback(
    (code: string, added: boolean) => {
      const key = added ? "addedAnnounce" : "removedAnnounce"
      setLiveMessage(t(key, { language: getLanguageLabel(code, locale) }))
    },
    [locale, t],
  )
  const add = useCallback(
    (bucket: LanguageBucket, code: string) => {
      if (!code) return
      buckets.moveTo(bucket, code)
      announce(code, true)
    },
    [announce, buckets],
  )
  const remove = useCallback(
    (bucket: LanguageBucket, code: string) => {
      buckets.removeFrom(bucket, code)
      announce(code, false)
    },
    [announce, buckets],
  )
  const save = useCallback(
    () =>
      mutation.mutate({
        professional: buckets.professional,
        conversational: buckets.conversational,
      }),
    [buckets.conversational, buckets.professional, mutation],
  )

  return {
    professional: buckets.professional,
    conversational: buckets.conversational,
    liveMessage,
    isDirty: buckets.isDirty,
    isSaving: mutation.isPending,
    add,
    remove,
    clear: buckets.clear,
    save,
  }
}

interface BucketsState {
  professional: string[]
  conversational: string[]
  isDirty: boolean
  moveTo: (bucket: LanguageBucket, code: string) => void
  removeFrom: (bucket: LanguageBucket, code: string) => void
  clear: (bucket: LanguageBucket) => void
}

function useBucketsSyncedWithProfile(
  profile: { languages_professional?: string[]; languages_conversational?: string[] } | undefined,
): BucketsState {
  // Sync draft to server state during render, not in an effect: the
  // React-recommended pattern. We key on a stable joined string so
  // we only reset when the persisted arrays change by value.
  const persistedProKey = (profile?.languages_professional ?? []).join(",")
  const persistedConvKey = (profile?.languages_conversational ?? []).join(",")

  const [prevProKey, setPrevProKey] = useState(persistedProKey)
  const [prevConvKey, setPrevConvKey] = useState(persistedConvKey)
  const [professional, setProfessional] = useState<string[]>(
    profile?.languages_professional ?? [],
  )
  const [conversational, setConversational] = useState<string[]>(
    profile?.languages_conversational ?? [],
  )

  if (prevProKey !== persistedProKey || prevConvKey !== persistedConvKey) {
    setPrevProKey(persistedProKey)
    setPrevConvKey(persistedConvKey)
    setProfessional(persistedProKey ? persistedProKey.split(",") : [])
    setConversational(persistedConvKey ? persistedConvKey.split(",") : [])
  }

  const isDirty =
    professional.join(",") !== persistedProKey ||
    conversational.join(",") !== persistedConvKey

  const moveTo = useCallback((bucket: LanguageBucket, code: string) => {
    const addTo = bucket === "professional" ? setProfessional : setConversational
    const removeFrom =
      bucket === "professional" ? setConversational : setProfessional
    addTo((current) => (current.includes(code) ? current : [...current, code]))
    removeFrom((current) => current.filter((c) => c !== code))
  }, [])

  const removeFrom = useCallback((bucket: LanguageBucket, code: string) => {
    const target = bucket === "professional" ? setProfessional : setConversational
    target((current) => current.filter((c) => c !== code))
  }, [])

  const clear = useCallback((bucket: LanguageBucket) => {
    if (bucket === "professional") setProfessional([])
    else setConversational([])
  }, [])

  return { professional, conversational, isDirty, moveTo, removeFrom, clear }
}

interface LanguageBucketEditorProps {
  bucket: LanguageBucket
  locale: LanguageComboboxLocale
  draft: LanguagesDraft
}

function LanguageBucketEditor({
  bucket,
  locale,
  draft,
}: LanguageBucketEditorProps) {
  const t = useTranslations("profile.languages")
  const labelKey =
    bucket === "professional" ? "professionalLabel" : "conversationalLabel"
  const helpKey =
    bucket === "professional" ? "professionalHelp" : "conversationalHelp"
  const selected =
    bucket === "professional" ? draft.professional : draft.conversational
  const labelId = useId()

  return (
    <div aria-labelledby={labelId} role="group">
      <div className="mb-1 flex items-baseline justify-between gap-3">
        <h3
          id={labelId}
          className="text-[15px] font-semibold text-foreground tracking-tight"
        >
          {t(labelKey)}
        </h3>
        {selected.length > 0 ? (
          <Button variant="ghost" size="auto"
            type="button"
            onClick={() => draft.clear(bucket)}
            className={cn(
              "text-xs font-medium text-muted-foreground rounded",
              "hover:text-foreground transition-colors duration-150",
              "focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2",
            )}
          >
            {t("clearAll")}
          </Button>
        ) : null}
      </div>
      <p className="mb-3 text-xs text-muted-foreground">{t(helpKey)}</p>
      <SelectedLanguageList bucket={bucket} locale={locale} draft={draft} />
      <LanguageCombobox
        selectedCodes={selected}
        locale={locale}
        onPick={(code) => draft.add(bucket, code)}
      />
      <p className="mt-2 text-xs text-muted-foreground tabular-nums">
        {t("selectionCount", { count: selected.length })}
      </p>
    </div>
  )
}

interface SelectedLanguageListProps {
  bucket: LanguageBucket
  locale: LanguageComboboxLocale
  draft: LanguagesDraft
}

function SelectedLanguageList({
  bucket,
  locale,
  draft,
}: SelectedLanguageListProps) {
  const t = useTranslations("profile.languages")
  const codes =
    bucket === "professional" ? draft.professional : draft.conversational
  if (codes.length === 0) return null
  const listLabelKey =
    bucket === "professional" ? "professionalLabel" : "conversationalLabel"

  return (
    <ul className="mb-3 flex flex-wrap gap-2" aria-label={t(listLabelKey)}>
      {codes.map((code) => (
        <li key={code}>
          <LanguageChip
            code={code}
            locale={locale}
            onRemove={() => draft.remove(bucket, code)}
          />
        </li>
      ))}
    </ul>
  )
}

interface LanguageChipProps {
  code: string
  locale: LanguageComboboxLocale
  onRemove: () => void
}

function LanguageChip({ code, locale, onRemove }: LanguageChipProps) {
  const t = useTranslations("profile.languages")
  const label = getLanguageLabel(code, locale)
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full",
        "bg-primary/10 text-primary border border-primary/20",
        "px-2.5 py-1 text-xs font-medium",
        "transition-colors duration-150",
      )}
    >
      <Globe
        className="h-3.5 w-3.5 opacity-70"
        aria-hidden="true"
        strokeWidth={2.25}
      />
      {label}
      <Button variant="ghost" size="auto"
        type="button"
        onClick={onRemove}
        aria-label={`${t("remove")} ${label}`}
        className={cn(
          "inline-flex h-4 w-4 items-center justify-center rounded-full",
          "hover:bg-primary/20 transition-colors duration-150",
          "focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2",
        )}
      >
        <X className="h-3 w-3" aria-hidden="true" />
      </Button>
    </span>
  )
}
