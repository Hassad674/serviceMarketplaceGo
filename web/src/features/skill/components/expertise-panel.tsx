"use client"

import { useState } from "react"
import { ChevronDown, ChevronRight, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { useSkillCatalog } from "../hooks/use-skill-catalog"
import type { SkillResponse } from "../types"

interface ExpertisePanelProps {
  expertiseKey: string
  alreadySelected: Set<string>
  onAdd: (skill: SkillResponse) => void
  defaultOpen?: boolean
}

// Collapsible section showing every curated skill in one expertise
// domain as clickable chips. Loaded lazily — the first request for
// this expertise fires on expansion, not on mount.
export function ExpertisePanel({
  expertiseKey,
  alreadySelected,
  onAdd,
  defaultOpen = false,
}: ExpertisePanelProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen)
  const catalog = useSkillCatalog(expertiseKey, isOpen)
  const tDomains = useTranslations("profile.expertise.domains")
  const tSkills = useTranslations("profile.skills")

  const domainLabel = safeDomainLabel(tDomains, expertiseKey)
  const count = catalog.data?.total ?? 0

  return (
    <div className="rounded-lg border border-border bg-card">
      <button
        type="button"
        onClick={() => setIsOpen((v) => !v)}
        aria-expanded={isOpen}
        className={cn(
          "flex w-full items-center justify-between gap-3 px-4 py-3 text-left",
          "hover:bg-muted/40 focus-visible:outline-2 focus-visible:outline-ring",
          "focus-visible:outline-offset-[-2px]",
        )}
      >
        <span className="flex items-center gap-2">
          {isOpen ? (
            <ChevronDown className="h-4 w-4" aria-hidden="true" />
          ) : (
            <ChevronRight className="h-4 w-4" aria-hidden="true" />
          )}
          <span className="text-sm font-semibold text-foreground">
            {domainLabel}
          </span>
        </span>
        {isOpen && catalog.data ? (
          <span className="text-xs text-muted-foreground">
            {tSkills("panelCount", { count })}
          </span>
        ) : null}
      </button>
      {isOpen ? (
        <ExpertisePanelBody
          skills={catalog.data?.skills ?? []}
          isLoading={catalog.isLoading}
          alreadySelected={alreadySelected}
          onAdd={onAdd}
        />
      ) : null}
    </div>
  )
}

type ExpertisePanelBodyProps = {
  skills: SkillResponse[]
  isLoading: boolean
  alreadySelected: Set<string>
  onAdd: (skill: SkillResponse) => void
}

function ExpertisePanelBody({
  skills,
  isLoading,
  alreadySelected,
  onAdd,
}: ExpertisePanelBodyProps) {
  if (isLoading) {
    return (
      <div className="flex items-center gap-2 px-4 py-3 text-sm text-muted-foreground">
        <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
      </div>
    )
  }
  if (skills.length === 0) {
    return null
  }
  return (
    <div className="flex flex-wrap gap-2 px-4 pb-4">
      {skills.map((skill) => {
        const isSelected = alreadySelected.has(skill.skill_text)
        return (
          <button
            key={skill.skill_text}
            type="button"
            onClick={() => onAdd(skill)}
            disabled={isSelected}
            className={cn(
              "inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium",
              "transition-colors duration-150",
              "focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2",
              isSelected
                ? "cursor-not-allowed border-primary/30 bg-primary/5 text-primary/50"
                : "border-border bg-background text-foreground hover:border-primary/60 hover:bg-primary/5",
            )}
          >
            {skill.display_text}
          </button>
        )
      })}
    </div>
  )
}

// next-intl throws when a key is missing; expertise keys that aren't
// in the frontend catalog (e.g. backend added a new domain) are
// returned as-is with no localisation instead of crashing the panel.
function safeDomainLabel(
  t: ReturnType<typeof useTranslations>,
  key: string,
): string {
  try {
    return t(key)
  } catch {
    return key
  }
}
