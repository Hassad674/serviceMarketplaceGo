import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { MilestoneEditor } from "../milestone-editor"
import type { MilestoneFormItem } from "../../types"
import { createEmptyMilestoneItem } from "../../types"

// Translation mock returns the key, suffixed with the parameters when
// useTranslations(...).t(key, params) is called. Lets us assert on the
// underlying message keys without setting up the full i18n pipeline.
vi.mock("next-intl", () => ({
  useTranslations: (namespace?: string) => (key: string) =>
    namespace ? `${namespace}.${key}` : key,
}))

vi.mock("@/shared/lib/utils", () => ({
  cn: (...classes: unknown[]) => classes.filter(Boolean).join(" "),
}))

vi.mock("lucide-react", () => ({
  Plus: () => <span data-testid="plus-icon" />,
  Trash2: () => <span data-testid="trash-icon" />,
}))

function withDeadline(deadline: string): MilestoneFormItem {
  return { ...createEmptyMilestoneItem(), deadline }
}

describe("MilestoneEditor", () => {
  it("dynamically computes the min date for milestone N+1 from milestone N's deadline", () => {
    const onChange = vi.fn()
    const milestones = [
      withDeadline("2026-05-07"),
      withDeadline(""), // user has not set the second one yet
    ]
    render(<MilestoneEditor milestones={milestones} onChange={onChange} />)

    // Two date inputs, one per milestone. The aria-label includes the
    // sequence so we can target the second one specifically.
    const dateInputs = screen
      .getAllByRole("textbox", { hidden: true })
      .concat(screen.queryAllByLabelText(/deadlineAriaLabel/i))
    // Date inputs are not "textbox" — fall back to a CSS-style query
    // by aria-label that we know was set.
    const second = screen.getByLabelText(
      /milestone 2 .*deadlineAriaLabel/,
    ) as HTMLInputElement
    // Min must be 2026-05-08 — strictly after the previous milestone.
    expect(second.min).toBe("2026-05-08")

    // Sanity check the aggregation didn't break the simpler case.
    const first = screen.getByLabelText(
      /milestone 1 .*deadlineAriaLabel/,
    ) as HTMLInputElement
    // First milestone's `min` is today (no previous deadline) — we
    // can't know the exact ISO without knowing today, but it must
    // never be earlier than the previous milestone's value.
    expect(first.min.length).toBe(10) // YYYY-MM-DD shape
    expect(dateInputs.length).toBeGreaterThan(0)
  })

  it("renders an inline error when milestone N+1 deadline is BEFORE N's", () => {
    const onChange = vi.fn()
    const milestones = [withDeadline("2026-05-07"), withDeadline("2026-05-06")]
    render(<MilestoneEditor milestones={milestones} onChange={onChange} />)

    // The error message uses the i18n key "error.not_after_previous"
    // suffixed with the editor namespace by our test mock.
    const error = screen.getByRole("alert")
    expect(error.textContent).toContain("error.not_after_previous")

    // The offending input must be marked invalid for accessibility.
    const second = screen.getByLabelText(
      /milestone 2 .*deadlineAriaLabel/,
    ) as HTMLInputElement
    expect(second.getAttribute("aria-invalid")).toBe("true")
    expect(second.getAttribute("aria-describedby")).toBe(
      "milestone-2-deadline-error",
    )
  })

  it("renders an inline error when the milestone exceeds the project deadline", () => {
    const onChange = vi.fn()
    const milestones = [withDeadline("2026-05-07"), withDeadline("2026-07-01")]
    render(
      <MilestoneEditor
        milestones={milestones}
        onChange={onChange}
        projectDeadline="2026-06-01"
      />,
    )

    const error = screen.getByRole("alert")
    expect(error.textContent).toContain("error.after_project_deadline")
  })

  it("does not render any error when deadlines are strictly increasing", () => {
    const onChange = vi.fn()
    const milestones = [
      withDeadline("2026-05-07"),
      withDeadline("2026-05-14"),
      withDeadline("2026-05-28"),
    ]
    render(<MilestoneEditor milestones={milestones} onChange={onChange} />)

    expect(screen.queryByRole("alert")).toBeNull()
  })

  it("forwards the project deadline as the date picker's max", () => {
    const onChange = vi.fn()
    const milestones = [withDeadline("")]
    render(
      <MilestoneEditor
        milestones={milestones}
        onChange={onChange}
        projectDeadline="2026-06-01"
      />,
    )

    const first = screen.getByLabelText(
      /milestone 1 .*deadlineAriaLabel/,
    ) as HTMLInputElement
    expect(first.max).toBe("2026-06-01")
  })
})
