"use client"

import { useState } from "react"
import { Pencil } from "lucide-react"
import { useTranslations } from "next-intl"
import {
  getMaxSkillsForOrgType,
  orgTypeSupportsSkills,
} from "../constants"
import { useProfileSkills } from "../hooks/use-profile-skills"
import { SkillChip } from "./skill-chip"
import { SkillsEditorModal } from "./skills-editor-modal"

import { Button } from "@/shared/components/ui/button"
interface SkillsSectionProps {
  orgType: string | undefined
  readOnly?: boolean
}

// Inline section rendered on the profile edit page right after the
// expertise editor. Acts as the read-only summary + the entry point
// to the modal editor.
export function SkillsSection({
  orgType,
  readOnly = false,
}: SkillsSectionProps) {
  const t = useTranslations("profile.skills")
  const [modalOpen, setModalOpen] = useState(false)
  const { data: skills, isLoading } = useProfileSkills()
  const maxSkills = getMaxSkillsForOrgType(orgType)

  if (!orgTypeSupportsSkills(orgType)) return null
  if (readOnly && (skills ?? []).length === 0) return null

  const orderedSkills = [...(skills ?? [])].sort(
    (a, b) => a.position - b.position,
  )

  return (
    <section
      aria-labelledby="skills-section-title"
      className="bg-card border border-border rounded-xl p-6 shadow-sm"
    >
      <header className="mb-4 flex flex-col gap-1">
        <h2
          id="skills-section-title"
          className="text-lg font-semibold text-foreground"
        >
          {t("sectionTitle")}
        </h2>
        {!readOnly && (
          <p className="text-sm text-muted-foreground">
            {t("sectionSubtitle", { max: maxSkills })}
          </p>
        )}
      </header>

      <div className="mb-4">
        {isLoading ? (
          <p className="text-sm text-muted-foreground">{t("loading")}</p>
        ) : orderedSkills.length === 0 ? (
          <p className="text-sm italic text-muted-foreground">{t("empty")}</p>
        ) : (
          <ul className="flex flex-wrap gap-2" aria-label={t("listLabel")}>
            {orderedSkills.map((skill) => (
              <li key={skill.skill_text}>
                <SkillChip displayText={skill.display_text} />
              </li>
            ))}
          </ul>
        )}
      </div>

      {!readOnly ? (
        <Button variant="ghost" size="auto"
          type="button"
          onClick={() => setModalOpen(true)}
          className="inline-flex items-center gap-2 rounded-md border border-border h-9 px-4 text-sm font-medium text-foreground hover:bg-muted focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
        >
          <Pencil className="h-4 w-4" aria-hidden="true" />
          {t("editButton")}
        </Button>
      ) : null}

      <SkillsEditorModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        maxSkills={maxSkills}
      />
    </section>
  )
}
