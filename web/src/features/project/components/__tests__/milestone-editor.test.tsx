import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { MilestoneEditor } from "../milestone-editor"
import type { Milestone } from "../../types"

// Mock next-intl
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

// Mock lucide-react icons
vi.mock("lucide-react", () => ({
  GripVertical: (props: Record<string, unknown>) => <span data-testid="grip-icon" {...props} />,
  Plus: (props: Record<string, unknown>) => <span data-testid="plus-icon" {...props} />,
  Trash2: (props: Record<string, unknown>) => <span data-testid="trash-icon" {...props} />,
}))

// Mock crypto.randomUUID since JSDOM does not implement it
let uuidCounter = 0
vi.stubGlobal("crypto", {
  randomUUID: () => `mock-uuid-${++uuidCounter}`,
})

function createMilestone(overrides: Partial<Milestone> = {}): Milestone {
  return {
    id: `ms-${++uuidCounter}`,
    title: "",
    description: "",
    deadline: "",
    amount: "",
    ...overrides,
  }
}

describe("MilestoneEditor", () => {
  it("renders initial milestones", () => {
    const milestones = [
      createMilestone({ id: "ms-1", title: "Design" }),
      createMilestone({ id: "ms-2", title: "Development" }),
    ]
    const onChange = vi.fn()

    render(<MilestoneEditor milestones={milestones} onChange={onChange} />)

    const titleInputs = screen.getAllByPlaceholderText("milestoneTitle")
    expect(titleInputs).toHaveLength(2)
    expect((titleInputs[0] as HTMLInputElement).value).toBe("Design")
    expect((titleInputs[1] as HTMLInputElement).value).toBe("Development")
  })

  it("add milestone button adds a new one", () => {
    const milestones = [createMilestone({ id: "ms-1" })]
    const onChange = vi.fn()

    render(<MilestoneEditor milestones={milestones} onChange={onChange} />)

    const addButton = screen.getByText("addMilestone")
    fireEvent.click(addButton)

    expect(onChange).toHaveBeenCalledTimes(1)
    const newMilestones = onChange.mock.calls[0][0] as Milestone[]
    expect(newMilestones).toHaveLength(2)
  })

  it("delete removes a milestone", () => {
    const milestones = [
      createMilestone({ id: "ms-1", title: "Keep" }),
      createMilestone({ id: "ms-2", title: "Remove" }),
    ]
    const onChange = vi.fn()

    render(<MilestoneEditor milestones={milestones} onChange={onChange} />)

    // Click the second delete button
    const deleteButtons = screen.getAllByRole("button", { name: "Delete milestone" })
    fireEvent.click(deleteButtons[1])

    expect(onChange).toHaveBeenCalledTimes(1)
    const updated = onChange.mock.calls[0][0] as Milestone[]
    expect(updated).toHaveLength(1)
    expect(updated[0].title).toBe("Keep")
  })

  it("minimum 1 milestone enforced — delete button hidden when only one", () => {
    const milestones = [createMilestone({ id: "ms-1" })]
    const onChange = vi.fn()

    render(<MilestoneEditor milestones={milestones} onChange={onChange} />)

    const deleteButtons = screen.queryAllByRole("button", { name: "Delete milestone" })
    expect(deleteButtons).toHaveLength(0)
  })

  it("updating title calls onChange with updated milestone", () => {
    const milestones = [createMilestone({ id: "ms-1", title: "Old" })]
    const onChange = vi.fn()

    render(<MilestoneEditor milestones={milestones} onChange={onChange} />)

    const titleInput = screen.getByPlaceholderText("milestoneTitle")
    fireEvent.change(titleInput, { target: { value: "New Title" } })

    expect(onChange).toHaveBeenCalledTimes(1)
    const updated = onChange.mock.calls[0][0] as Milestone[]
    expect(updated[0].title).toBe("New Title")
  })

  it("updating description calls onChange", () => {
    const milestones = [createMilestone({ id: "ms-1" })]
    const onChange = vi.fn()

    render(<MilestoneEditor milestones={milestones} onChange={onChange} />)

    const descInput = screen.getByPlaceholderText("milestoneDesc")
    fireEvent.change(descInput, { target: { value: "New description" } })

    expect(onChange).toHaveBeenCalledTimes(1)
    const updated = onChange.mock.calls[0][0] as Milestone[]
    expect(updated[0].description).toBe("New description")
  })

  it("renders milestone number badges", () => {
    const milestones = [
      createMilestone({ id: "ms-1" }),
      createMilestone({ id: "ms-2" }),
      createMilestone({ id: "ms-3" }),
    ]
    const onChange = vi.fn()

    render(<MilestoneEditor milestones={milestones} onChange={onChange} />)

    expect(screen.getByText("1")).toBeDefined()
    expect(screen.getByText("2")).toBeDefined()
    expect(screen.getByText("3")).toBeDefined()
  })

  it("shows delete buttons when multiple milestones exist", () => {
    const milestones = [
      createMilestone({ id: "ms-1" }),
      createMilestone({ id: "ms-2" }),
    ]
    const onChange = vi.fn()

    render(<MilestoneEditor milestones={milestones} onChange={onChange} />)

    const deleteButtons = screen.getAllByRole("button", { name: "Delete milestone" })
    expect(deleteButtons).toHaveLength(2)
  })
})
