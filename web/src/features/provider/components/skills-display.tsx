"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"

// A read-only skill as surfaced by the backend ProfileResponse and
// PublicProfileSummary DTOs. `skill_text` is the canonical lowercase key
// and `display_text` is the user-facing label.
export type SkillChipData = {
  skill_text: string
  display_text: string
}

interface SkillsDisplayProps {
  skills: SkillChipData[] | undefined
  maxVisible?: number
  className?: string
}

// SkillsDisplay is the read-only pill list used on provider cards and
// public profiles. It mirrors ExpertiseDisplay in tone: no interactivity,
// no edit affordance, just chips. The component lives inside the provider
// feature (not the skill feature) so the provider feature remains
// independently removable per the project's modularity rules — no
// cross-feature imports.
//
// `maxVisible` is optional: when omitted the component renders all chips,
// which is the right behavior for the full public profile section. When
// set (typically on the compact provider card) the first N chips are
// rendered followed by a "+X" overflow chip that is non-interactive.
export function SkillsDisplay({
  skills,
  maxVisible,
  className,
}: SkillsDisplayProps) {
  const t = useTranslations("profile.skillsDisplay")

  if (!skills || skills.length === 0) return null

  const shouldTruncate =
    typeof maxVisible === "number" && skills.length > maxVisible
  const visible = shouldTruncate ? skills.slice(0, maxVisible) : skills
  const hiddenCount = shouldTruncate ? skills.length - visible.length : 0

  return (
    <ul
      aria-label={t("listLabel")}
      className={cn("flex flex-wrap gap-1.5", className)}
    >
      {visible.map((skill) => (
        <li key={skill.skill_text}>
          <span className="inline-flex items-center rounded-full bg-primary/10 text-primary px-2.5 py-0.5 text-xs font-medium border border-primary/20">
            {skill.display_text}
          </span>
        </li>
      ))}
      {hiddenCount > 0 && (
        <li>
          <span
            className="inline-flex items-center rounded-full bg-muted text-muted-foreground px-2.5 py-0.5 text-xs font-medium border border-border"
            aria-hidden="false"
          >
            {t("moreSuffix", { count: hiddenCount })}
          </span>
        </li>
      )}
    </ul>
  )
}
