"use client"

import { useState, type KeyboardEvent } from "react"
import { X } from "lucide-react"
import { useTranslations } from "next-intl"
import { EXPERTISE_DOMAIN_KEYS } from "@/shared/lib/profile/expertise"
import { LanguageMultiSelect } from "@/shared/components/forms/language-multi-select"
import {
  CheckboxRow,
  SectionShell,
  toggle,
} from "./filter-primitives"

import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"

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
//
// Per-persona visibility flags are honoured here so the parent never
// has to render a fragment of the component — referrer pages hide the
// skills sub-section entirely (skills don't apply to apporteurs), and
// agency/referrer pages keep the languages + expertise sub-sections.

interface FilterSectionSkillsExpertiseProps {
  languages: string[]
  expertise: string[]
  skills: string[]
  showLanguages?: boolean
  showExpertise?: boolean
  showSkills?: boolean
  onLanguagesChange: (next: string[]) => void
  onExpertiseChange: (next: string[]) => void
  onSkillsChange: (next: string[]) => void
}

export function FilterSectionSkillsExpertise({
  languages,
  expertise,
  skills,
  showLanguages = true,
  showExpertise = true,
  showSkills = true,
  onLanguagesChange,
  onExpertiseChange,
  onSkillsChange,
}: FilterSectionSkillsExpertiseProps) {
  const t = useTranslations("search.filters")
  const tDomains = useTranslations("profile.expertise.domains")

  return (
    <>
      {showLanguages ? (
        <SectionShell title={t("languages")}>
          <LanguageMultiSelect
            selected={languages}
            onChange={onLanguagesChange}
            placeholder={t("languagesPlaceholder")}
            ariaLabel={t("languages")}
          />
        </SectionShell>
      ) : null}

      {showExpertise ? (
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
      ) : null}

      {showSkills ? (
        <SkillsBlock selected={skills} onChange={onSkillsChange} />
      ) : null}
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
      <Input
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
        className="h-10 rounded-lg border border-border bg-background px-3 text-sm shadow-xs focus:border-primary focus:outline-none focus:ring-4 focus:ring-primary/10"
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
            className="inline-flex items-center gap-1 rounded-full bg-primary-soft px-2.5 py-1 text-xs font-medium text-primary-deep transition-colors hover:bg-primary/30"
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
          className="inline-flex items-center rounded-full border border-border bg-background px-2.5 py-1 text-xs text-muted-foreground transition-colors hover:border-primary/40 hover:text-primary-deep focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary/20"
        >
          + {skill}
        </Button>
      ))}
    </div>
  )
}
