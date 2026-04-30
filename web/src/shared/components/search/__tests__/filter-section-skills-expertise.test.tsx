import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { FilterSectionSkillsExpertise } from "../filter-section-skills-expertise"

function renderSection(opts: {
  languages?: string[]
  expertise?: string[]
  skills?: string[]
} = {}) {
  const onLanguagesChange = vi.fn()
  const onExpertiseChange = vi.fn()
  const onSkillsChange = vi.fn()
  render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <FilterSectionSkillsExpertise
        languages={opts.languages ?? []}
        expertise={opts.expertise ?? []}
        skills={opts.skills ?? []}
        onLanguagesChange={onLanguagesChange}
        onExpertiseChange={onExpertiseChange}
        onSkillsChange={onSkillsChange}
      />
    </NextIntlClientProvider>,
  )
  return { onLanguagesChange, onExpertiseChange, onSkillsChange }
}

describe("FilterSectionSkillsExpertise", () => {
  it("renders the 3 section headings (languages, expertise, skills)", () => {
    renderSection()
    expect(
      screen.getByRole("heading", { name: messages.search.filters.languages }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("heading", { name: messages.search.filters.expertise }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("heading", { name: messages.search.filters.skills }),
    ).toBeInTheDocument()
  })

  it("toggles a language on/off", () => {
    const { onLanguagesChange } = renderSection({ languages: [] })
    fireEvent.click(screen.getByRole("button", { name: "FR" }))
    expect(onLanguagesChange).toHaveBeenCalledWith(["fr"])
  })

  it("removes a language when already selected", () => {
    const { onLanguagesChange } = renderSection({ languages: ["fr", "en"] })
    fireEvent.click(screen.getByRole("button", { name: "FR" }))
    expect(onLanguagesChange).toHaveBeenCalledWith(["en"])
  })

  it("toggles an expertise checkbox", () => {
    const { onExpertiseChange } = renderSection({ expertise: [] })
    // Find any expertise checkbox (label varies by domain key)
    const checkboxes = screen.getAllByRole("checkbox")
    expect(checkboxes.length).toBeGreaterThan(0)
    fireEvent.click(checkboxes[0])
    expect(onExpertiseChange).toHaveBeenCalled()
  })

  it("adds a skill on Enter", () => {
    const { onSkillsChange } = renderSection({ skills: [] })
    const input = screen.getByLabelText(
      messages.search.filters.skillsSearchPlaceholder,
    )
    fireEvent.change(input, { target: { value: "Rust" } })
    fireEvent.keyDown(input, { key: "Enter" })
    expect(onSkillsChange).toHaveBeenCalledWith(["Rust"])
  })

  it("adds a skill on comma", () => {
    const { onSkillsChange } = renderSection({ skills: [] })
    const input = screen.getByLabelText(
      messages.search.filters.skillsSearchPlaceholder,
    )
    fireEvent.change(input, { target: { value: "Rust" } })
    fireEvent.keyDown(input, { key: "," })
    expect(onSkillsChange).toHaveBeenCalledWith(["Rust"])
  })

  it("dedupes case-insensitively", () => {
    const { onSkillsChange } = renderSection({ skills: ["React"] })
    const input = screen.getByLabelText(
      messages.search.filters.skillsSearchPlaceholder,
    )
    fireEvent.change(input, { target: { value: "react" } })
    fireEvent.keyDown(input, { key: "Enter" })
    expect(onSkillsChange).not.toHaveBeenCalled()
  })

  it("ignores empty / whitespace-only skills", () => {
    const { onSkillsChange } = renderSection({ skills: [] })
    const input = screen.getByLabelText(
      messages.search.filters.skillsSearchPlaceholder,
    )
    fireEvent.change(input, { target: { value: "   " } })
    fireEvent.keyDown(input, { key: "Enter" })
    expect(onSkillsChange).not.toHaveBeenCalled()
  })

  it("removes the last skill on Backspace when input is empty", () => {
    const { onSkillsChange } = renderSection({ skills: ["A", "B"] })
    const input = screen.getByLabelText(
      messages.search.filters.skillsSearchPlaceholder,
    )
    fireEvent.keyDown(input, { key: "Backspace" })
    expect(onSkillsChange).toHaveBeenCalledWith(["A"])
  })

  it("keeps the input when Backspace is pressed with content", () => {
    const { onSkillsChange } = renderSection({ skills: ["A"] })
    const input = screen.getByLabelText(
      messages.search.filters.skillsSearchPlaceholder,
    )
    fireEvent.change(input, { target: { value: "x" } })
    fireEvent.keyDown(input, { key: "Backspace" })
    expect(onSkillsChange).not.toHaveBeenCalled()
  })

  it("commits the draft on blur when non-empty", () => {
    const { onSkillsChange } = renderSection({ skills: [] })
    const input = screen.getByLabelText(
      messages.search.filters.skillsSearchPlaceholder,
    )
    fireEvent.change(input, { target: { value: "GraphQL" } })
    fireEvent.blur(input)
    expect(onSkillsChange).toHaveBeenCalledWith(["GraphQL"])
  })

  it("does NOT commit on blur when draft is whitespace", () => {
    const { onSkillsChange } = renderSection({ skills: [] })
    const input = screen.getByLabelText(
      messages.search.filters.skillsSearchPlaceholder,
    )
    fireEvent.change(input, { target: { value: "   " } })
    fireEvent.blur(input)
    expect(onSkillsChange).not.toHaveBeenCalled()
  })

  it("renders selected skill chips as removable buttons", () => {
    const { onSkillsChange } = renderSection({ skills: ["Rust", "Zig"] })
    const removeBtn = screen.getByRole("button", { name: "Remove Rust" })
    fireEvent.click(removeBtn)
    expect(onSkillsChange).toHaveBeenCalledWith(["Zig"])
  })

  it("renders popular-skill suggestions excluding already selected", () => {
    renderSection({ skills: ["React"] })
    expect(screen.queryByText("+ React")).not.toBeInTheDocument()
    // TypeScript is one of the popular-skill suggestions
    expect(screen.getByText("+ TypeScript")).toBeInTheDocument()
  })

  it("clicking a popular-skill chip adds it to the selection", () => {
    const { onSkillsChange } = renderSection({ skills: [] })
    fireEvent.click(screen.getByText("+ TypeScript"))
    expect(onSkillsChange).toHaveBeenCalledWith(["TypeScript"])
  })

  it("hides the popular-skill row when all suggestions are already selected", () => {
    renderSection({
      skills: [
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
      ],
    })
    expect(screen.queryByText(/^\+ /)).not.toBeInTheDocument()
  })
})
