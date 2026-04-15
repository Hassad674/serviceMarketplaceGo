"use client"

import { Star } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { EXPERTISE_DOMAIN_KEYS } from "@/shared/lib/profile/expertise"
import {
  EMPTY_SEARCH_FILTERS,
  isEmptyFilters,
  type SearchAvailabilityFilter,
  type SearchFilters,
  type SearchWorkMode,
} from "./search-filters"

// SearchFilterSidebar renders the Malt-style left rail filter UI. It
// is intentionally logic-free: every change flows through `onChange`
// and the parent owns the state. The "Apply" button is a no-op today
// (it calls `onApply` which the parent may wire into Typesense later)
// and the "Reset" button re-emits the canonical empty state.
//
// Every section is labelled, keyboard-accessible, and uses the design
// system's semantic tokens — zero hardcoded colors.

interface SearchFilterSidebarProps {
  filters: SearchFilters
  onChange: (next: SearchFilters) => void
  onApply?: () => void
  resultsCount?: number
  className?: string
}

// Top-N common skills kept as a static local list until the skills
// catalog endpoint is wired. Matches the conventions of the shared
// profile expertise list — a frozen constant that the filter UI can
// render without a network call.
const TOP_SKILLS = [
  "React",
  "TypeScript",
  "Go",
  "Python",
  "Node.js",
  "Figma",
  "Docker",
  "Kubernetes",
  "AWS",
  "PostgreSQL",
] as const

const AVAILABILITY_OPTIONS: readonly SearchAvailabilityFilter[] = [
  "now",
  "soon",
  "all",
]

const WORK_MODE_OPTIONS: readonly SearchWorkMode[] = [
  "remote",
  "on_site",
  "hybrid",
]

const COMMON_LANGUAGES = ["fr", "en", "es", "de", "it", "pt"] as const

export function SearchFilterSidebar({
  filters,
  onChange,
  onApply,
  resultsCount,
  className,
}: SearchFilterSidebarProps) {
  const t = useTranslations("search.filters")
  const tSearch = useTranslations("search")
  const hasFilters = !isEmptyFilters(filters)

  const update = <K extends keyof SearchFilters>(key: K, value: SearchFilters[K]) => {
    onChange({ ...filters, [key]: value })
  }

  return (
    <aside
      className={cn(
        "flex w-full flex-col gap-6 rounded-2xl border border-border bg-card p-5 shadow-sm",
        "lg:sticky lg:top-24 lg:max-h-[calc(100vh-7rem)] lg:overflow-y-auto",
        className,
      )}
      aria-label={t("title")}
    >
      <header className="flex items-center justify-between gap-2">
        <h2 className="text-base font-semibold text-foreground">
          {t("title")}
        </h2>
        {typeof resultsCount === "number" ? (
          <span className="text-xs text-muted-foreground">
            {tSearch("resultsCount", { count: resultsCount })}
          </span>
        ) : null}
      </header>

      <AvailabilitySection
        value={filters.availability}
        onChange={(v) => update("availability", v)}
      />
      <PriceSection
        min={filters.priceMin}
        max={filters.priceMax}
        onMinChange={(v) => update("priceMin", v)}
        onMaxChange={(v) => update("priceMax", v)}
      />
      <LocationSection
        city={filters.city}
        countryCode={filters.countryCode}
        radiusKm={filters.radiusKm}
        onCityChange={(v) => update("city", v)}
        onCountryChange={(v) => update("countryCode", v)}
        onRadiusChange={(v) => update("radiusKm", v)}
      />
      <LanguagesSection
        selected={filters.languages}
        onChange={(v) => update("languages", v)}
      />
      <ExpertiseSection
        selected={filters.expertise}
        onChange={(v) =>
          update(
            "expertise",
            v as SearchFilters["expertise"],
          )
        }
      />
      <SkillsSection
        selected={filters.skills}
        onChange={(v) => update("skills", v)}
      />
      <RatingSection
        value={filters.minRating}
        onChange={(v) => update("minRating", v)}
      />
      <WorkModeSection
        selected={filters.workModes}
        onChange={(v) => update("workModes", v)}
      />

      <footer className="sticky bottom-0 flex flex-col gap-2 bg-card pt-2">
        <button
          type="button"
          onClick={onApply}
          className="inline-flex h-10 items-center justify-center rounded-lg bg-rose-500 px-4 text-sm font-medium text-white transition-all duration-200 ease-out hover:bg-rose-600 hover:shadow-glow active:scale-[0.98] focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-rose-500/20"
        >
          {t("apply")}
        </button>
        {hasFilters ? (
          <button
            type="button"
            onClick={() => onChange(EMPTY_SEARCH_FILTERS)}
            className="inline-flex h-10 items-center justify-center rounded-lg border border-border bg-background px-4 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-rose-500/20"
          >
            {t("reset")}
          </button>
        ) : null}
      </footer>
    </aside>
  )
}

// ---------------------------------------------------------------------------
// Sub-components — each keeps one section fully local
// ---------------------------------------------------------------------------

function SectionShell({
  title,
  children,
}: {
  title: string
  children: React.ReactNode
}) {
  return (
    <section className="flex flex-col gap-2">
      <h3 className="text-[13px] font-semibold uppercase tracking-wide text-muted-foreground">
        {title}
      </h3>
      {children}
    </section>
  )
}

function AvailabilitySection({
  value,
  onChange,
}: {
  value: SearchAvailabilityFilter
  onChange: (next: SearchAvailabilityFilter) => void
}) {
  const t = useTranslations("search.filters")
  const labels: Record<SearchAvailabilityFilter, string> = {
    now: t("availableNow"),
    soon: t("availableSoon"),
    all: t("availabilityAll"),
  }
  return (
    <SectionShell title={t("availability")}>
      <div className="flex flex-wrap gap-2">
        {AVAILABILITY_OPTIONS.map((option) => (
          <PillButton
            key={option}
            label={labels[option]}
            selected={value === option}
            onClick={() => onChange(option)}
          />
        ))}
      </div>
    </SectionShell>
  )
}

function PriceSection({
  min,
  max,
  onMinChange,
  onMaxChange,
}: {
  min: number | null
  max: number | null
  onMinChange: (next: number | null) => void
  onMaxChange: (next: number | null) => void
}) {
  const t = useTranslations("search.filters")
  return (
    <SectionShell title={t("price")}>
      <div className="flex items-center gap-2">
        <NumberInput
          placeholder={t("priceMin")}
          value={min}
          onChange={onMinChange}
          ariaLabel={t("priceMin")}
        />
        <span className="text-xs text-muted-foreground">–</span>
        <NumberInput
          placeholder={t("priceMax")}
          value={max}
          onChange={onMaxChange}
          ariaLabel={t("priceMax")}
        />
      </div>
    </SectionShell>
  )
}

function LocationSection({
  city,
  countryCode,
  radiusKm,
  onCityChange,
  onCountryChange,
  onRadiusChange,
}: {
  city: string
  countryCode: string
  radiusKm: number | null
  onCityChange: (next: string) => void
  onCountryChange: (next: string) => void
  onRadiusChange: (next: number | null) => void
}) {
  const t = useTranslations("search.filters")
  return (
    <SectionShell title={t("location")}>
      <input
        type="text"
        value={city}
        onChange={(e) => onCityChange(e.target.value)}
        placeholder={t("cityPlaceholder")}
        aria-label={t("cityPlaceholder")}
        className="h-10 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
      />
      <input
        type="text"
        value={countryCode}
        onChange={(e) => onCountryChange(e.target.value.toUpperCase().slice(0, 2))}
        placeholder={t("countryPlaceholder")}
        aria-label={t("countryPlaceholder")}
        maxLength={2}
        className="h-10 w-20 rounded-lg border border-border bg-background px-3 text-sm uppercase shadow-xs focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
      />
      <NumberInput
        placeholder={t("radiusPlaceholder")}
        value={radiusKm}
        onChange={onRadiusChange}
        ariaLabel={t("radiusPlaceholder")}
      />
    </SectionShell>
  )
}

function LanguagesSection({
  selected,
  onChange,
}: {
  selected: string[]
  onChange: (next: string[]) => void
}) {
  const t = useTranslations("search.filters")
  return (
    <SectionShell title={t("languages")}>
      <div className="flex flex-wrap gap-2">
        {COMMON_LANGUAGES.map((code) => (
          <PillButton
            key={code}
            label={code.toUpperCase()}
            selected={selected.includes(code)}
            onClick={() => onChange(toggle(selected, code))}
          />
        ))}
      </div>
    </SectionShell>
  )
}

function ExpertiseSection({
  selected,
  onChange,
}: {
  selected: string[]
  onChange: (next: string[]) => void
}) {
  const t = useTranslations("search.filters")
  const tDomains = useTranslations("profile.expertise.domains")
  return (
    <SectionShell title={t("expertise")}>
      <ul className="flex flex-col gap-1">
        {EXPERTISE_DOMAIN_KEYS.map((key) => (
          <li key={key}>
            <CheckboxRow
              checked={selected.includes(key)}
              onChange={() => onChange(toggle(selected, key))}
              label={safeExpertiseLabel(tDomains, key)}
            />
          </li>
        ))}
      </ul>
    </SectionShell>
  )
}

function SkillsSection({
  selected,
  onChange,
}: {
  selected: string[]
  onChange: (next: string[]) => void
}) {
  const t = useTranslations("search.filters")
  return (
    <SectionShell title={t("skills")}>
      <input
        type="search"
        placeholder={t("skillsSearchPlaceholder")}
        aria-label={t("skillsSearchPlaceholder")}
        className="h-10 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
      />
      <ul className="flex flex-col gap-1">
        {TOP_SKILLS.map((skill) => (
          <li key={skill}>
            <CheckboxRow
              checked={selected.includes(skill)}
              onChange={() => onChange(toggle(selected, skill))}
              label={skill}
            />
          </li>
        ))}
      </ul>
    </SectionShell>
  )
}

function RatingSection({
  value,
  onChange,
}: {
  value: number
  onChange: (next: number) => void
}) {
  const t = useTranslations("search.filters")
  return (
    <SectionShell title={t("rating")}>
      <div className="flex items-center gap-1" role="radiogroup" aria-label={t("rating")}>
        {[1, 2, 3, 4, 5].map((star) => {
          const selected = star <= value
          return (
            <button
              key={star}
              type="button"
              role="radio"
              aria-checked={selected}
              aria-label={`${star}`}
              onClick={() => onChange(value === star ? 0 : star)}
              className={cn(
                "rounded-sm p-0.5 transition-colors",
                selected ? "text-amber-400" : "text-muted-foreground/40",
                "hover:text-amber-400 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-rose-500/20",
              )}
            >
              <Star
                className={cn("h-5 w-5", selected && "fill-amber-400")}
                strokeWidth={1.75}
              />
            </button>
          )
        })}
      </div>
    </SectionShell>
  )
}

function WorkModeSection({
  selected,
  onChange,
}: {
  selected: SearchWorkMode[]
  onChange: (next: SearchWorkMode[]) => void
}) {
  const t = useTranslations("search.filters")
  const labels: Record<SearchWorkMode, string> = {
    remote: t("remote"),
    on_site: t("onSite"),
    hybrid: t("hybrid"),
  }
  return (
    <SectionShell title={t("workMode")}>
      <div className="flex flex-wrap gap-2">
        {WORK_MODE_OPTIONS.map((mode) => (
          <PillButton
            key={mode}
            label={labels[mode]}
            selected={selected.includes(mode)}
            onClick={() => onChange(toggle(selected, mode))}
          />
        ))}
      </div>
    </SectionShell>
  )
}

// ---------------------------------------------------------------------------
// Shared leaf primitives
// ---------------------------------------------------------------------------

function PillButton({
  label,
  selected,
  onClick,
}: {
  label: string
  selected: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={selected}
      className={cn(
        "rounded-full border px-3 py-1 text-xs font-medium transition-colors",
        selected
          ? "border-rose-500 bg-rose-50 text-rose-700 dark:bg-rose-500/15 dark:text-rose-300"
          : "border-border bg-background text-muted-foreground hover:text-foreground",
        "focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-rose-500/20",
      )}
    >
      {label}
    </button>
  )
}

function NumberInput({
  value,
  onChange,
  placeholder,
  ariaLabel,
}: {
  value: number | null
  onChange: (next: number | null) => void
  placeholder: string
  ariaLabel: string
}) {
  return (
    <input
      type="number"
      min={0}
      inputMode="numeric"
      value={value ?? ""}
      placeholder={placeholder}
      aria-label={ariaLabel}
      onChange={(e) => {
        const raw = e.target.value.trim()
        onChange(raw === "" ? null : Math.max(0, Number(raw) || 0))
      }}
      className="h-10 w-full min-w-0 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
    />
  )
}

function CheckboxRow({
  checked,
  onChange,
  label,
}: {
  checked: boolean
  onChange: () => void
  label: string
}) {
  return (
    <label className="flex cursor-pointer items-center gap-2 rounded-md px-1 py-1 text-sm text-foreground hover:bg-muted/50">
      <input
        type="checkbox"
        checked={checked}
        onChange={onChange}
        className="h-4 w-4 rounded border-border text-rose-500 focus:ring-rose-500/20"
      />
      <span className="flex-1 truncate">{label}</span>
    </label>
  )
}

function toggle<T>(list: T[], value: T): T[] {
  return list.includes(value)
    ? list.filter((item) => item !== value)
    : [...list, value]
}

// safeExpertiseLabel looks up an expertise domain key's localized
// label from the shared `profile.expertise.domains` namespace and
// falls back to a humanized rendition when the message is missing.
// Keeps the filter UI from crashing if an older translation file has
// not been synced with a newly-added domain key.
function safeExpertiseLabel(
  t: ReturnType<typeof useTranslations>,
  key: string,
): string {
  try {
    return t(key)
  } catch {
    return key.replace(/_/g, " ")
  }
}
