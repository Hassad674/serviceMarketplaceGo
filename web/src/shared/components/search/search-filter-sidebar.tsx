"use client"

import { useState, type KeyboardEvent } from "react"
import { Star, X } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { EXPERTISE_DOMAIN_KEYS } from "@/shared/lib/profile/expertise"
import type { SearchDocumentPersona } from "@/shared/lib/search/search-document"
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
  /**
   * persona drives the price section's labels and unit suffix:
   *   - freelance -> "TJM min / max",  suffix "€"
   *   - agency    -> "Budget min / max", suffix "€"
   *   - referrer  -> "Commission min / max", suffix "%"
   *
   * Undefined falls back to the generic "Price min / max" labels —
   * this keeps the sidebar usable in contexts that never picked a
   * persona (e.g. stories / legacy callers). The min / max input
   * values are still raw numbers; the backend filter builder maps
   * them to the correct Typesense clause per persona.
   */
  persona?: SearchDocumentPersona
}

// POPULAR_SKILLS is rendered as quick-add chips below the free-text
// input so the user can one-click the common ones without having to
// type them. The list is intentionally short — curated suggestions,
// not an exhaustive directory. A proper catalog-driven autocomplete
// ships in phase 3 alongside the server-side facet index.
const POPULAR_SKILLS = [
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
  persona,
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
        persona={persona}
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

// PriceSection renders the min / max bounds that the parent pipes
// into the Typesense filter_by builder. The labels and the unit
// suffix are persona-aware (see buildPriceLabels) so the UX matches
// the primary pricing shape for the persona being searched. The
// input values stay raw numbers — the persona only affects how the
// bounds are labelled for the user, not how they are persisted or
// sent to the backend.
function PriceSection({
  persona,
  min,
  max,
  onMinChange,
  onMaxChange,
}: {
  persona: SearchDocumentPersona | undefined
  min: number | null
  max: number | null
  onMinChange: (next: number | null) => void
  onMaxChange: (next: number | null) => void
}) {
  const t = useTranslations("search.filters")
  const labels = buildPriceLabels(t, persona)
  return (
    <SectionShell title={labels.title}>
      <div className="flex items-center gap-2">
        <NumberInputWithSuffix
          placeholder={labels.minPlaceholder}
          ariaLabel={labels.minPlaceholder}
          suffix={labels.unit}
          value={min}
          onChange={onMinChange}
        />
        <span className="text-xs text-muted-foreground">–</span>
        <NumberInputWithSuffix
          placeholder={labels.maxPlaceholder}
          ariaLabel={labels.maxPlaceholder}
          suffix={labels.unit}
          value={max}
          onChange={onMaxChange}
        />
      </div>
    </SectionShell>
  )
}

interface PriceLabels {
  title: string
  minPlaceholder: string
  maxPlaceholder: string
  unit: string
}

// buildPriceLabels returns the persona-specific title / placeholders /
// unit suffix for the PriceSection. Undefined persona falls back to
// the generic price labels so legacy callers keep working.
function buildPriceLabels(
  t: ReturnType<typeof useTranslations>,
  persona: SearchDocumentPersona | undefined,
): PriceLabels {
  switch (persona) {
    case "freelance":
      return {
        title: t("freelancePrice"),
        minPlaceholder: t("freelancePriceMin"),
        maxPlaceholder: t("freelancePriceMax"),
        unit: "€",
      }
    case "agency":
      return {
        title: t("agencyPrice"),
        minPlaceholder: t("agencyPriceMin"),
        maxPlaceholder: t("agencyPriceMax"),
        unit: "€",
      }
    case "referrer":
      return {
        title: t("referrerPrice"),
        minPlaceholder: t("referrerPriceMin"),
        maxPlaceholder: t("referrerPriceMax"),
        unit: "%",
      }
    default:
      return {
        title: t("price"),
        minPlaceholder: t("priceMin"),
        maxPlaceholder: t("priceMax"),
        unit: "",
      }
  }
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
  const [draft, setDraft] = useState("")

  const addSkill = (raw: string) => {
    const trimmed = raw.trim()
    if (trimmed.length === 0) return
    // Dedupe case-insensitively so "react" and "React" do not stack
    // as separate filter clauses. Typesense's `:` operator is
    // case-insensitive at query time too.
    if (selected.some((s) => s.toLowerCase() === trimmed.toLowerCase())) return
    onChange([...selected, trimmed])
  }

  const removeSkill = (value: string) => {
    onChange(selected.filter((s) => s !== value))
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter" || e.key === ",") {
      e.preventDefault()
      addSkill(draft)
      setDraft("")
    } else if (e.key === "Backspace" && draft.length === 0 && selected.length > 0) {
      removeSkill(selected[selected.length - 1])
    }
  }

  return (
    <SectionShell title={t("skills")}>
      <SelectedSkillsChips selected={selected} onRemove={removeSkill} />
      <input
        type="text"
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onKeyDown={handleKeyDown}
        onBlur={() => {
          if (draft.trim().length > 0) {
            addSkill(draft)
            setDraft("")
          }
        }}
        placeholder={t("skillsSearchPlaceholder")}
        aria-label={t("skillsSearchPlaceholder")}
        className="h-10 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
      />
      <PopularSkillChips
        selected={selected}
        onPick={(skill) => addSkill(skill)}
      />
    </SectionShell>
  )
}

function SelectedSkillsChips({
  selected,
  onRemove,
}: {
  selected: string[]
  onRemove: (value: string) => void
}) {
  if (selected.length === 0) return null
  return (
    <ul
      className="flex flex-wrap gap-1.5"
      aria-label="selected skills"
    >
      {selected.map((skill) => (
        <li key={skill}>
          <button
            type="button"
            onClick={() => onRemove(skill)}
            aria-label={`Remove ${skill}`}
            className="inline-flex items-center gap-1 rounded-full bg-rose-100 px-2.5 py-1 text-xs font-medium text-rose-700 transition-colors hover:bg-rose-200 dark:bg-rose-500/15 dark:text-rose-300 dark:hover:bg-rose-500/25"
          >
            <span>{skill}</span>
            <X className="h-3 w-3" aria-hidden strokeWidth={2.5} />
          </button>
        </li>
      ))}
    </ul>
  )
}

function PopularSkillChips({
  selected,
  onPick,
}: {
  selected: string[]
  onPick: (skill: string) => void
}) {
  const selectedLower = new Set(selected.map((s) => s.toLowerCase()))
  const available = POPULAR_SKILLS.filter(
    (s) => !selectedLower.has(s.toLowerCase()),
  )
  if (available.length === 0) return null
  return (
    <div className="flex flex-wrap gap-1.5 pt-1">
      {available.map((skill) => (
        <button
          key={skill}
          type="button"
          onClick={() => onPick(skill)}
          className="inline-flex items-center rounded-full border border-border bg-background px-2.5 py-1 text-xs text-muted-foreground transition-colors hover:border-rose-300 hover:text-rose-700 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-rose-500/20 dark:hover:text-rose-300"
        >
          + {skill}
        </button>
      ))}
    </div>
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

// NumberInputWithSuffix is a NumberInput decorated with a trailing
// unit suffix (€ or %). Kept in-file because the suffix is purely
// cosmetic to the price section — the rest of the sidebar does not
// need it. When suffix is empty we fall back to the plain input so
// we do not reserve padding for nothing.
function NumberInputWithSuffix({
  value,
  onChange,
  placeholder,
  ariaLabel,
  suffix,
}: {
  value: number | null
  onChange: (next: number | null) => void
  placeholder: string
  ariaLabel: string
  suffix: string
}) {
  if (suffix === "") {
    return (
      <NumberInput
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        ariaLabel={ariaLabel}
      />
    )
  }
  return (
    <div className="relative w-full min-w-0">
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
        className="h-10 w-full min-w-0 rounded-lg border border-border bg-background pl-3 pr-8 text-sm shadow-xs focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10"
      />
      <span
        aria-hidden="true"
        className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-xs font-medium text-muted-foreground"
      >
        {suffix}
      </span>
    </div>
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
