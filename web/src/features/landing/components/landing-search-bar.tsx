"use client"

import { useState, type FormEvent } from "react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { Search } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import { trackSearch } from "@/shared/lib/analytics-events"

// LandingSearchBar — the centerpiece of the hero. Three role tabs
// (Freelance / Apporteur / Agence) drive the placeholder copy AND
// the redirect target on submit. The form has NO budget input —
// explicit user constraint, regression-tested.
//
// On submit, the bar redirects to one of the existing public listing
// routes (/freelancers, /referrers, /agencies) with `?q=` and `?city=`
// query params. Those routes already host the SearchPage component
// with the full filter sidebar.
//
// Submit handlers are explicit (form `onSubmit` AND a click handler
// on the button so keyboard Enter and mouse click follow the same
// path). Empty queries are still accepted — the listing page renders
// the unscoped catalog, which matches user intent ("just take me
// there").

const ROLE_TO_PATH = {
  freelance: "/freelancers",
  referrer: "/referrers",
  agency: "/agencies",
} as const

type LandingSearchRole = keyof typeof ROLE_TO_PATH

const ROLE_PLACEHOLDER_KEY: Record<LandingSearchRole, string> = {
  freelance: "queryPlaceholderFreelance",
  referrer: "queryPlaceholderReferrer",
  agency: "queryPlaceholderAgency",
}

const SUGGESTION_KEYS = [
  "suggestion1",
  "suggestion2",
  "suggestion3",
  "suggestion4",
  "suggestion5",
] as const

const ROLES: readonly LandingSearchRole[] = [
  "freelance",
  "referrer",
  "agency",
]

// buildSearchUrl returns the URL string the form pushes onto the
// router. Plain string + URLSearchParams keeps next-intl's typed
// router happy without per-route narrow types.
function buildSearchUrl(
  role: LandingSearchRole,
  query: string,
  city: string,
): string {
  const params = new URLSearchParams()
  const trimmedQuery = query.trim()
  const trimmedCity = city.trim()
  if (trimmedQuery) params.set("q", trimmedQuery)
  if (trimmedCity) params.set("city", trimmedCity)
  const search = params.toString()
  const path = ROLE_TO_PATH[role]
  return search ? `${path}?${search}` : path
}

export function LandingSearchBar() {
  const t = useTranslations("landing.search")
  const router = useRouter()

  const [role, setRole] = useState<LandingSearchRole>("freelance")
  const [query, setQuery] = useState("")
  const [city, setCity] = useState("")

  // Count "filled" filter slots for the GA4 search event so we can
  // see how often users hit Submit with a city scope vs. raw text.
  function filtersCount(values: { query: string; city: string }): number {
    let count = 0
    if (values.query.trim()) count += 1
    if (values.city.trim()) count += 1
    return count
  }

  function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    trackSearch({
      searchTerm: query.trim(),
      persona: role,
      filtersCount: filtersCount({ query, city }),
    })
    router.push(buildSearchUrl(role, query, city))
  }

  function handleSuggestionClick(label: string) {
    trackSearch({
      searchTerm: label.trim(),
      persona: role,
      filtersCount: filtersCount({ query: label, city }),
    })
    router.push(buildSearchUrl(role, label, city))
  }

  return (
    <div className="rounded-[28px] border border-border bg-card p-1.5 shadow-[var(--shadow-card)] sm:p-2">
      <RoleTabs current={role} onChange={setRole} label={t("label")} />
      <form
        id="landing-search-form"
        role="search"
        onSubmit={handleSubmit}
        className="mt-2 flex flex-col gap-3 px-3 pb-3 sm:mt-3 sm:flex-row sm:items-stretch sm:gap-0 sm:px-4 sm:pb-4 sm:pt-3"
      >
        <QueryField
          value={query}
          onChange={setQuery}
          placeholder={t(ROLE_PLACEHOLDER_KEY[role])}
          label={t("queryLabel")}
        />
        <div className="hidden h-auto w-px bg-border sm:block" />
        <CityField
          value={city}
          onChange={setCity}
          placeholder={t("locationPlaceholder")}
          label={t("locationLabel")}
        />
        <SubmitButton ariaLabel={t("submitAria")} label={t("submit")} />
      </form>
      <SuggestionChips
        label={t("popularLabel")}
        keys={SUGGESTION_KEYS}
        onPick={handleSuggestionClick}
      />
    </div>
  )
}

interface RoleTabsProps {
  current: LandingSearchRole
  onChange: (next: LandingSearchRole) => void
  label: string
}

const ROLE_TITLE_KEY: Record<LandingSearchRole, string> = {
  freelance: "tabFreelance",
  referrer: "tabReferrer",
  agency: "tabAgency",
}

const ROLE_DESCRIPTION_KEY: Record<LandingSearchRole, string> = {
  freelance: "tabFreelanceDescription",
  referrer: "tabReferrerDescription",
  agency: "tabAgencyDescription",
}

function RoleTabs({ current, onChange, label }: RoleTabsProps) {
  return (
    <div
      role="tablist"
      aria-label={label}
      className="grid grid-cols-1 gap-2 px-3 pt-3 sm:grid-cols-3 sm:gap-0 sm:px-4 sm:pt-4"
    >
      {ROLES.map((id) => (
        <RoleTabButton
          key={id}
          id={id}
          isActive={current === id}
          onClick={() => onChange(id)}
        />
      ))}
    </div>
  )
}

interface RoleTabButtonProps {
  id: LandingSearchRole
  isActive: boolean
  onClick: () => void
}

function RoleTabButton({ id, isActive, onClick }: RoleTabButtonProps) {
  const t = useTranslations("landing.search")
  return (
    <button
      type="button"
      role="tab"
      id={`landing-tab-${id}`}
      aria-selected={isActive}
      aria-controls="landing-search-form"
      onClick={onClick}
      className={cn(
        "flex flex-col items-start gap-1 rounded-2xl px-4 py-2 text-left transition-colors sm:rounded-none sm:border-b-2 sm:px-2 sm:pb-3",
        isActive
          ? "bg-primary-soft sm:border-foreground sm:bg-transparent"
          : "border-transparent hover:bg-primary-soft/40 sm:hover:bg-transparent",
      )}
    >
      <span
        className={cn(
          "font-serif text-[18px] font-medium leading-none tracking-[-0.01em]",
          isActive ? "text-foreground" : "text-muted-foreground",
        )}
      >
        {t(ROLE_TITLE_KEY[id])}
      </span>
      <span className="text-[12.5px] text-muted-foreground">
        {t(ROLE_DESCRIPTION_KEY[id])}
      </span>
    </button>
  )
}

interface FieldProps {
  value: string
  onChange: (next: string) => void
  placeholder: string
  label: string
}

function QueryField({ value, onChange, placeholder, label }: FieldProps) {
  return (
    <div className="flex flex-1 flex-col px-1 sm:px-2">
      <label
        htmlFor="landing-search-query"
        className="font-mono text-[10px] font-bold uppercase tracking-[0.1em] text-subtle-foreground"
      >
        {label}
      </label>
      <input
        id="landing-search-query"
        name="q"
        type="search"
        autoComplete="off"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="mt-1 w-full bg-transparent text-[14px] text-foreground outline-none placeholder:text-muted-foreground/70"
      />
    </div>
  )
}

function CityField({ value, onChange, placeholder, label }: FieldProps) {
  return (
    <div className="flex flex-col px-1 sm:w-44 sm:px-3">
      <label
        htmlFor="landing-search-city"
        className="font-mono text-[10px] font-bold uppercase tracking-[0.1em] text-subtle-foreground"
      >
        {label}
      </label>
      <input
        id="landing-search-city"
        name="city"
        type="text"
        autoComplete="off"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="mt-1 w-full bg-transparent text-[14px] text-foreground outline-none placeholder:text-muted-foreground/70"
      />
    </div>
  )
}

interface SubmitButtonProps {
  ariaLabel: string
  label: string
}

function SubmitButton({ ariaLabel, label }: SubmitButtonProps) {
  return (
    <button
      type="submit"
      aria-label={ariaLabel}
      className="mt-2 inline-flex items-center justify-center gap-2 rounded-full bg-primary px-6 py-3 text-sm font-semibold text-primary-foreground shadow-[var(--shadow-message)] transition-colors hover:bg-primary-deep sm:mt-0 sm:px-7"
    >
      <Search className="h-4 w-4" strokeWidth={2} aria-hidden="true" />
      <span>{label}</span>
    </button>
  )
}

interface SuggestionChipsProps {
  label: string
  keys: readonly string[]
  onPick: (label: string) => void
}

function SuggestionChips({ label, keys, onPick }: SuggestionChipsProps) {
  const t = useTranslations("landing.search")
  return (
    <div className="hidden flex-wrap items-center gap-2 px-5 pb-4 pt-1 sm:flex">
      <span className="text-[12.5px] text-muted-foreground">{label}</span>
      {keys.map((key) => {
        const text = t(key)
        return (
          <button
            key={key}
            type="button"
            onClick={() => onPick(text)}
            className="rounded-full border border-border bg-background px-3 py-1.5 text-[12.5px] text-muted-foreground transition-colors hover:border-primary hover:text-primary-deep"
          >
            {text}
          </button>
        )
      })}
    </div>
  )
}
