"use client"

import { useCallback, useState } from "react"
import { Check, Loader2, X } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import {
  LANGUAGE_OPTIONS,
  getLanguageLabel,
} from "../lib/language-options"
import { useProfile } from "../hooks/use-profile"
import { useUpdateLanguages } from "../hooks/use-update-languages"

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
  const locale = useLocale() === "fr" ? "fr" : "en"
  const { data: profile } = useProfile()
  const mutation = useUpdateLanguages()

  // Depend on a stable string key, not the array reference, and reset
  // the local draft during render instead of in an effect (the React
  // recommended pattern for syncing state to props).
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

  const addLanguage = useCallback(
    (bucket: LanguageBucket, code: string) => {
      if (!code) return
      if (bucket === "professional") {
        setProfessional((current) =>
          current.includes(code) ? current : [...current, code],
        )
        setConversational((current) => current.filter((c) => c !== code))
      } else {
        setConversational((current) =>
          current.includes(code) ? current : [...current, code],
        )
        setProfessional((current) => current.filter((c) => c !== code))
      }
    },
    [],
  )

  const removeLanguage = useCallback(
    (bucket: LanguageBucket, code: string) => {
      if (bucket === "professional") {
        setProfessional((current) => current.filter((c) => c !== code))
      } else {
        setConversational((current) => current.filter((c) => c !== code))
      }
    },
    [],
  )

  const handleSave = useCallback(() => {
    mutation.mutate({ professional, conversational })
  }, [conversational, mutation, professional])

  if (orgType === "enterprise") return null
  if (readOnly) return null

  return (
    <section
      aria-labelledby="languages-section-title"
      className="bg-card border border-border rounded-xl p-6 shadow-sm"
    >
      <header className="mb-4 flex flex-col gap-1">
        <h2
          id="languages-section-title"
          className="text-lg font-semibold text-foreground"
        >
          {t("sectionTitle")}
        </h2>
        <p className="text-sm text-muted-foreground">{t("sectionSubtitle")}</p>
      </header>

      <div className="space-y-5">
        <LanguageBucketEditor
          bucket="professional"
          selected={professional}
          locale={locale}
          onAdd={(code) => addLanguage("professional", code)}
          onRemove={(code) => removeLanguage("professional", code)}
        />
        <LanguageBucketEditor
          bucket="conversational"
          selected={conversational}
          locale={locale}
          onAdd={(code) => addLanguage("conversational", code)}
          onRemove={(code) => removeLanguage("conversational", code)}
        />
        <div className="flex items-center justify-end">
          <button
            type="button"
            onClick={handleSave}
            disabled={!isDirty || mutation.isPending}
            className="bg-primary text-primary-foreground rounded-md h-9 px-4 text-sm font-medium hover:opacity-90 transition-opacity duration-150 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 disabled:opacity-50 disabled:cursor-not-allowed inline-flex items-center gap-2"
          >
            {mutation.isPending ? (
              <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />
            ) : (
              <Check className="w-4 h-4" aria-hidden="true" />
            )}
            {mutation.isPending ? t("saving") : t("save")}
          </button>
        </div>
      </div>
    </section>
  )
}

interface LanguageBucketEditorProps {
  bucket: LanguageBucket
  selected: string[]
  locale: "fr" | "en"
  onAdd: (code: string) => void
  onRemove: (code: string) => void
}

function LanguageBucketEditor({
  bucket,
  selected,
  locale,
  onAdd,
  onRemove,
}: LanguageBucketEditorProps) {
  const t = useTranslations("profile.languages")
  const labelKey = bucket === "professional" ? "professionalLabel" : "conversationalLabel"
  const helpKey = bucket === "professional" ? "professionalHelp" : "conversationalHelp"
  const inputId = `languages-${bucket}-select`

  const unselected = LANGUAGE_OPTIONS.filter(
    (option) => !selected.includes(option.code),
  )

  return (
    <div>
      <label htmlFor={inputId} className="block text-sm font-medium text-foreground">
        {t(labelKey)}
      </label>
      <p className="text-xs text-muted-foreground mb-2">{t(helpKey)}</p>

      {selected.length > 0 ? (
        <ul className="mb-2 flex flex-wrap gap-1.5" aria-label={t(labelKey)}>
          {selected.map((code) => (
            <li key={code}>
              <span className="inline-flex items-center gap-1.5 rounded-full bg-primary/10 text-primary px-2.5 py-1 text-xs font-medium border border-primary/20">
                {getLanguageLabel(code, locale)}
                <button
                  type="button"
                  onClick={() => onRemove(code)}
                  aria-label={`${t("remove")} ${getLanguageLabel(code, locale)}`}
                  className={cn(
                    "inline-flex h-4 w-4 items-center justify-center rounded-full",
                    "hover:bg-primary/20 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2",
                  )}
                >
                  <X className="h-3 w-3" aria-hidden="true" />
                </button>
              </span>
            </li>
          ))}
        </ul>
      ) : null}

      <select
        id={inputId}
        value=""
        onChange={(e) => {
          onAdd(e.target.value)
          e.currentTarget.value = ""
        }}
        className="h-10 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-primary focus:ring-4 focus:ring-primary/10 focus:outline-none"
      >
        <option value="">{t("searchPlaceholder")}</option>
        {unselected.map((option) => (
          <option key={option.code} value={option.code}>
            {getLanguageLabel(option.code, locale)}
          </option>
        ))}
      </select>
    </div>
  )
}

