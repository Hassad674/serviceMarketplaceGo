import { describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"

import { SkillsDisplay, type SkillChipData } from "../skills-display"

function renderDisplay(
  skills: SkillChipData[] | undefined,
  maxVisible?: number,
) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <SkillsDisplay skills={skills} maxVisible={maxVisible} />
    </NextIntlClientProvider>,
  )
}

function makeSkills(count: number): SkillChipData[] {
  return Array.from({ length: count }, (_, index) => ({
    skill_text: `skill-${index + 1}`,
    display_text: `Skill ${index + 1}`,
  }))
}

describe("SkillsDisplay", () => {
  it("renders nothing when skills is undefined", () => {
    const { container } = renderDisplay(undefined)
    expect(container).toBeEmptyDOMElement()
  })

  it("renders nothing when skills is an empty array", () => {
    const { container } = renderDisplay([])
    expect(container).toBeEmptyDOMElement()
  })

  it("renders all chips when maxVisible is unset", () => {
    const skills = makeSkills(6)
    renderDisplay(skills)
    for (const skill of skills) {
      expect(screen.getByText(skill.display_text)).toBeInTheDocument()
    }
    expect(screen.queryByText(/^\+\d+$/)).not.toBeInTheDocument()
  })

  it("renders only the first N chips plus an overflow chip when maxVisible < total", () => {
    const skills = makeSkills(7)
    renderDisplay(skills, 4)
    expect(screen.getByText("Skill 1")).toBeInTheDocument()
    expect(screen.getByText("Skill 2")).toBeInTheDocument()
    expect(screen.getByText("Skill 3")).toBeInTheDocument()
    expect(screen.getByText("Skill 4")).toBeInTheDocument()
    expect(screen.queryByText("Skill 5")).not.toBeInTheDocument()
    expect(screen.queryByText("Skill 6")).not.toBeInTheDocument()
    expect(screen.queryByText("Skill 7")).not.toBeInTheDocument()
    expect(screen.getByText("+3")).toBeInTheDocument()
  })

  it("does not render an overflow chip when skills.length equals maxVisible", () => {
    renderDisplay(makeSkills(4), 4)
    expect(screen.getByText("Skill 1")).toBeInTheDocument()
    expect(screen.getByText("Skill 4")).toBeInTheDocument()
    expect(screen.queryByText(/^\+\d+$/)).not.toBeInTheDocument()
  })

  it("exposes the list with an accessible aria-label", () => {
    renderDisplay(makeSkills(3))
    expect(screen.getByRole("list", { name: "List of skills" })).toBeInTheDocument()
  })
})
