"use client"

import { Globe } from "lucide-react"
import { useLocale, useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { getFlagEmoji } from "@/shared/lib/profile/country-options"
import {
  getLanguageFlagCountry,
  getLanguageLabel,
} from "@/shared/lib/profile/language-options"

const MAX_LANGUAGE_FLAGS = 5

interface LanguagesStripProps {
  professional: string[]
  conversational: string[]
  className?: string
}

// LanguagesStrip is the compact flag + code badge list used on public
// profile strips and listing cards. Shows up to MAX_LANGUAGE_FLAGS
// primary codes, collapses the rest into a "+N" overflow pill, then
// surfaces the conversational count as a secondary line.
export function LanguagesStrip({
  professional,
  conversational,
  className,
}: LanguagesStripProps) {
  const t = useTranslations("profile.languages")
  const locale = useLocale() === "fr" ? "fr" : "en"
  if (professional.length === 0 && conversational.length === 0) return null

  const visible = professional.slice(0, MAX_LANGUAGE_FLAGS)
  const overflow = professional.length - visible.length

  return (
    <div className={cn("flex flex-col gap-1", className)}>
      <div className="flex flex-wrap items-center gap-1.5">
        <Globe
          className="h-4 w-4 text-muted-foreground"
          aria-hidden="true"
        />
        {visible.map((code) => (
          <LanguageBadge key={code} code={code} locale={locale} />
        ))}
        {overflow > 0 ? (
          <span className="inline-flex items-center rounded-full border border-border bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
            +{overflow}
          </span>
        ) : null}
      </div>
      {conversational.length > 0 ? (
        <span className="text-[11px] text-muted-foreground">
          {t("conversationalShort", { count: conversational.length })}
        </span>
      ) : null}
    </div>
  )
}

interface LanguageBadgeProps {
  code: string
  locale: "fr" | "en"
}

function LanguageBadge({ code, locale }: LanguageBadgeProps) {
  const flag = getFlagEmoji(getLanguageFlagCountry(code))
  return (
    <span
      title={getLanguageLabel(code, locale)}
      className="inline-flex items-center gap-1 rounded-full border border-border bg-muted px-2 py-0.5 text-xs font-medium text-foreground"
    >
      {flag ? <span aria-hidden="true">{flag}</span> : null}
      {code.toUpperCase()}
    </span>
  )
}
