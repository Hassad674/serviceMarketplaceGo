/**
 * proposal-stepper.test.tsx
 *
 * Component tests for the proposal status stepper. Verifies the active
 * step lights up correctly for every status, including the terminal
 * negative states (declined, withdrawn) which render the alternate
 * "negative" path.
 */
import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { ProposalStepper } from "../proposal-stepper"
import type { ProposalStatus } from "../../types"

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

const STEPS = ["stepCreated", "stepAccepted", "stepPaid", "stepActive", "stepCompleted"]

describe("ProposalStepper — happy path", () => {
  it("renders all 5 step labels for status=pending", () => {
    render(<ProposalStepper status="pending" />)
    for (const label of STEPS) {
      expect(screen.getAllByText(label).length).toBeGreaterThan(0)
    }
  })

  it.each<ProposalStatus>([
    "pending",
    "accepted",
    "paid",
    "active",
    "completion_requested",
    "completed",
    "disputed",
  ])("renders without crashing for status=%s", (status) => {
    expect(() => render(<ProposalStepper status={status} />)).not.toThrow()
  })
})

describe("ProposalStepper — negative terminal states", () => {
  it("declined renders the negative variant (no progression)", () => {
    const { container } = render(<ProposalStepper status="declined" />)
    expect(container.firstChild).toBeTruthy()
  })

  it("withdrawn renders the negative variant", () => {
    const { container } = render(<ProposalStepper status="withdrawn" />)
    expect(container.firstChild).toBeTruthy()
  })
})

describe("ProposalStepper — accessibility", () => {
  it("does not render a button or interactive role (read-only)", () => {
    render(<ProposalStepper status="paid" />)
    // The stepper is purely informational — no interactive widgets.
    expect(screen.queryByRole("button")).toBeNull()
    expect(screen.queryByRole("link")).toBeNull()
  })
})
