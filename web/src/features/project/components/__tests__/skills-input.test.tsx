import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { SkillsInput } from "../skills-input"

// Mock next-intl
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

// Mock lucide-react icons
vi.mock("lucide-react", () => ({
  X: (props: Record<string, unknown>) => <span data-testid="x-icon" {...props} />,
}))

describe("SkillsInput", () => {
  it("renders with no skills", () => {
    const onChange = vi.fn()
    render(<SkillsInput skills={[]} onChange={onChange} />)

    expect(screen.getByText("requiredSkills")).toBeDefined()
    expect(screen.getByPlaceholderText("skillsPlaceholder")).toBeDefined()
  })

  it("enter adds a skill tag", () => {
    const onChange = vi.fn()
    render(<SkillsInput skills={[]} onChange={onChange} />)

    const input = screen.getByPlaceholderText("skillsPlaceholder")
    fireEvent.change(input, { target: { value: "React" } })
    fireEvent.keyDown(input, { key: "Enter", code: "Enter" })

    expect(onChange).toHaveBeenCalledWith(["React"])
  })

  it("click X removes a skill", () => {
    const onChange = vi.fn()
    render(<SkillsInput skills={["React", "TypeScript"]} onChange={onChange} />)

    // Find the remove button for React
    const removeButton = screen.getByRole("button", { name: "Remove React" })
    fireEvent.click(removeButton)

    expect(onChange).toHaveBeenCalledWith(["TypeScript"])
  })

  it("duplicate skills prevented", () => {
    const onChange = vi.fn()
    render(<SkillsInput skills={["React"]} onChange={onChange} />)

    const input = screen.getByRole("textbox")
    fireEvent.change(input, { target: { value: "React" } })
    fireEvent.keyDown(input, { key: "Enter", code: "Enter" })

    // onChange should not be called with a duplicate
    expect(onChange).not.toHaveBeenCalled()
  })

  it("duplicate check is case-insensitive", () => {
    const onChange = vi.fn()
    render(<SkillsInput skills={["React"]} onChange={onChange} />)

    const input = screen.getByRole("textbox")
    fireEvent.change(input, { target: { value: "react" } })
    fireEvent.keyDown(input, { key: "Enter", code: "Enter" })

    expect(onChange).not.toHaveBeenCalled()
  })

  it("renders existing skill tags", () => {
    const onChange = vi.fn()
    render(<SkillsInput skills={["Go", "PostgreSQL", "Docker"]} onChange={onChange} />)

    expect(screen.getByText("Go")).toBeDefined()
    expect(screen.getByText("PostgreSQL")).toBeDefined()
    expect(screen.getByText("Docker")).toBeDefined()
  })

  it("empty input does not add a skill", () => {
    const onChange = vi.fn()
    render(<SkillsInput skills={[]} onChange={onChange} />)

    const input = screen.getByPlaceholderText("skillsPlaceholder")
    fireEvent.change(input, { target: { value: "" } })
    fireEvent.keyDown(input, { key: "Enter", code: "Enter" })

    expect(onChange).not.toHaveBeenCalled()
  })

  it("whitespace-only input does not add a skill", () => {
    const onChange = vi.fn()
    render(<SkillsInput skills={[]} onChange={onChange} />)

    const input = screen.getByPlaceholderText("skillsPlaceholder")
    fireEvent.change(input, { target: { value: "   " } })
    fireEvent.keyDown(input, { key: "Enter", code: "Enter" })

    expect(onChange).not.toHaveBeenCalled()
  })

  it("trims whitespace from skill names", () => {
    const onChange = vi.fn()
    render(<SkillsInput skills={[]} onChange={onChange} />)

    const input = screen.getByPlaceholderText("skillsPlaceholder")
    fireEvent.change(input, { target: { value: "  React  " } })
    fireEvent.keyDown(input, { key: "Enter", code: "Enter" })

    expect(onChange).toHaveBeenCalledWith(["React"])
  })

  it("backspace removes last skill when input is empty", () => {
    const onChange = vi.fn()
    render(<SkillsInput skills={["Go", "Rust"]} onChange={onChange} />)

    const input = screen.getByRole("textbox")
    fireEvent.keyDown(input, { key: "Backspace", code: "Backspace" })

    expect(onChange).toHaveBeenCalledWith(["Go"])
  })

  it("placeholder hidden when skills exist", () => {
    const onChange = vi.fn()
    render(<SkillsInput skills={["React"]} onChange={onChange} />)

    const input = screen.getByRole("textbox") as HTMLInputElement
    expect(input.placeholder).toBe("")
  })

  it("adds skill on blur", () => {
    const onChange = vi.fn()
    render(<SkillsInput skills={[]} onChange={onChange} />)

    const input = screen.getByPlaceholderText("skillsPlaceholder")
    fireEvent.change(input, { target: { value: "Vue" } })
    fireEvent.blur(input)

    expect(onChange).toHaveBeenCalledWith(["Vue"])
  })
})
