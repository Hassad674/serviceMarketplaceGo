"use client"

import { useState, type KeyboardEvent } from "react"
import { X } from "lucide-react"
import { useTranslations } from "next-intl"
import { EXPERTISE_DOMAIN_KEYS } from "@/shared/lib/profile/expertise"
import {
  CheckboxRow,
  PillButton,
  SectionShell,
  toggle,
} from "./filter-primitives"

import { Button } from "@/shared/components/ui/button"
const COMMON_LANGUAGES = ["fr", "en", "es", "de", "it", "pt"] as const

// POPULAR_SKILLS is rendered as quick-add chips below the free-text
// input so the user can one-click the common ones without having to
// type them. Curated suggestions, not an exhaustive directory.
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

// Aggregates the three "what do they do" filters: spoken languages,
// domain expertise (broad), and skills (granular). Keeps them in a
// single component because the section ordering matters for the UX
// flow (broad → narrow) and they share the same compositional shape.

interface FilterSectionSkillsExpertiseProps {
  languages: string[]
  expertise: string[]
  skills: string[]
  onLanguagesChange: (next: string[]) => void
  onExpertiseChange: (next: string[]) => void
  onSkillsChange: (next: string[]) => void
}

export function FilterSectionSkillsExpertise({
  languages,
  expertise,
  skills,
  onLanguagesChange,
  onExpertiseChange,
  onSkillsChange,
}: FilterSectionSkillsExpertiseProps) {
  const t = useTranslations("search.filters")
  const tDomains = useTranslations("profile.expertise.domains")

  return (
    <>
      <SectionShell title={t("languages")}>
        <div className="flex flex-wrap gap-2">
          {COMMON_LANGUAGES.map((code) => (
            <PillButton
              key={code}
              label={code.toUpperCase()}
              selected={languages.includes(code)}
              onClick={() => onLanguagesChange(toggle(languages, code))}
            />
          ))}
        </div>
      </SectionShell>

      <SectionShell title={t("expertise")}>
        <ul className="flex flex-col gap-1">
          {EXPERTISE_DOMAIN_KEYS.map((key) => (
            <li key={key}>
              <CheckboxRow
                checked={expertise.includes(key)}
                onChange={() => onExpertiseChange(toggle(expertise, key))}
                label={safeExpertiseLabel(tDomains, key)}
              />
            </li>
          ))}
        </ul>
      </SectionShell>

      <SkillsBlock selected={skills} onChange={onSkillsChange} />
    </>
  )
}

// safeExpertiseLabel looks up an expertise domain key's localized
// label and falls back to a humanized rendition when the message is
// missing. Keeps the filter UI from crashing if an older translation
// file has not been synced with a newly-added domain key.
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

function SkillsBlock({
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
      <PopularSkillChips selected={selected} onPick={(skill) => addSkill(skill)} />
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
    <ul className="flex flex-wrap gap-1.5" aria-label="selected skills">
      {selected.map((skill) => (
        <li key={skill}>
          <Button variant="ghost" size="auto"
            type="button"
            onClick={() => onRemove(skill)}
            aria-label={`Remove ${skill}`}
            className="inline-flex items-center gap-1 rounded-full bg-rose-100 px-2.5 py-1 text-xs font-medium text-rose-700 transition-colors hover:bg-rose-200 dark:bg-rose-500/15 dark:text-rose-300 dark:hover:bg-rose-500/25"
          >
            <span>{skill}</span>
            <X className="h-3 w-3" aria-hidden strokeWidth={2.5} />
          </Button>
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
        <Button variant="ghost" size="auto"
          key={skill}
          type="button"
          onClick={() => onPick(skill)}
          className="inline-flex items-center rounded-full border border-border bg-background px-2.5 py-1 text-xs text-muted-foreground transition-colors hover:border-rose-300 hover:text-rose-700 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-rose-500/20 dark:hover:text-rose-300"
        >
          + {skill}
        </Button>
      ))}
    </div>
  )
}
