"use client"

import { useTranslations } from "next-intl"
import type { ProfileSkill } from "../api/freelance-profile-api"

interface FreelanceSkillsStripProps {
  skills: ProfileSkill[]
}

// FreelanceSkillsStrip renders the read-only chip list of an
// organization's skills. The array comes denormalized in the
// freelance profile response, so no extra fetch is needed. Skills
// are intentionally not editable here — the owner edits them through
// the existing skills modal composed elsewhere on /profile.
export function FreelanceSkillsStrip({ skills }: FreelanceSkillsStripProps) {
  const t = useTranslations("profile.skillsDisplay")
  if (!skills || skills.length === 0) return null
  return (
    <section
      aria-labelledby="freelance-skills-strip-title"
      className="bg-card border border-border rounded-xl p-6 shadow-sm"
    >
      <h2
        id="freelance-skills-strip-title"
        className="text-lg font-semibold text-foreground mb-3"
      >
        {t("sectionTitle")}
      </h2>
      <ul
        className="flex flex-wrap gap-2"
        aria-label={t("listLabel")}
        data-testid="freelance-skills-list"
      >
        {skills.map((skill) => (
          <li key={skill.skill_text}>
            <span className="inline-flex items-center rounded-full border border-border bg-muted px-3 py-1 text-xs font-medium text-foreground">
              {skill.display_text}
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}
