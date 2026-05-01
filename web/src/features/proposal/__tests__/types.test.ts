import { describe, expect, it } from "vitest"
import type { MilestoneFormItem } from "../types"
import {
  createEmptyMilestoneItem,
  minDateForMilestone,
  validateMilestoneDeadlines,
} from "../types"

// Build a milestone form item with the deadline overridden — every
// other field is irrelevant to the deadline-validation logic so we
// can keep the fixtures concise.
function withDeadline(deadline: string): MilestoneFormItem {
  return { ...createEmptyMilestoneItem(), deadline }
}

describe("validateMilestoneDeadlines", () => {
  it("accepts an empty list", () => {
    expect(validateMilestoneDeadlines([])).toEqual({})
  })

  it("accepts a single milestone (nothing to compare)", () => {
    const items = [withDeadline("2026-05-07")]
    expect(validateMilestoneDeadlines(items)).toEqual({})
  })

  it("accepts a strictly increasing sequence", () => {
    const items = [
      withDeadline("2026-05-07"),
      withDeadline("2026-05-14"),
      withDeadline("2026-05-28"),
    ]
    expect(validateMilestoneDeadlines(items)).toEqual({})
  })

  it("rejects two milestones on the same day (strict-after)", () => {
    const items = [withDeadline("2026-05-07"), withDeadline("2026-05-07")]
    expect(validateMilestoneDeadlines(items)).toEqual({ 1: "not_after_previous" })
  })

  it("rejects an out-of-order pair (the reported bug)", () => {
    // milestone 1 = 07/05, milestone 2 = 06/05 — exactly the bug report.
    const items = [withDeadline("2026-05-07"), withDeadline("2026-05-06")]
    expect(validateMilestoneDeadlines(items)).toEqual({ 1: "not_after_previous" })
  })

  it("rejects three milestones with the middle one out of order", () => {
    const items = [
      withDeadline("2026-05-01"),
      withDeadline("2026-04-25"),
      withDeadline("2026-05-10"),
    ]
    expect(validateMilestoneDeadlines(items)).toEqual({ 1: "not_after_previous" })
  })

  it("skips empty deadlines but still validates set ones", () => {
    const items = [
      withDeadline("2026-05-01"),
      withDeadline(""),
      withDeadline("2026-05-10"),
    ]
    expect(validateMilestoneDeadlines(items)).toEqual({})
  })

  it("treats sparse deadlines correctly: nil between two strictly-decreasing fails", () => {
    const items = [
      withDeadline("2026-05-10"),
      withDeadline(""),
      withDeadline("2026-05-05"),
    ]
    expect(validateMilestoneDeadlines(items)).toEqual({ 2: "not_after_previous" })
  })

  it("flags a milestone past the project deadline", () => {
    const items = [withDeadline("2026-05-07"), withDeadline("2026-07-01")]
    expect(validateMilestoneDeadlines(items, "2026-06-01")).toEqual({
      1: "after_project_deadline",
    })
  })

  it("allows a milestone equal to the project deadline (≤ rule)", () => {
    const items = [withDeadline("2026-05-07"), withDeadline("2026-06-01")]
    expect(validateMilestoneDeadlines(items, "2026-06-01")).toEqual({})
  })

  it("prefers the order-violation message over the project-deadline one", () => {
    // Both rules fail simultaneously — only the first detected error
    // for that index lands in the map. Strict-after is checked first
    // because that's what the user is most directly authoring.
    const items = [withDeadline("2026-07-15"), withDeadline("2026-07-01")]
    expect(validateMilestoneDeadlines(items, "2026-06-01")).toEqual({
      0: "after_project_deadline",
      1: "not_after_previous",
    })
  })
})

describe("minDateForMilestone", () => {
  const today = "2026-05-01"

  it("returns today when no previous milestone has a deadline", () => {
    const items = [withDeadline(""), withDeadline("")]
    expect(minDateForMilestone(items, 1, today)).toBe(today)
  })

  it("returns today for the first milestone (index 0)", () => {
    const items = [withDeadline(""), withDeadline("")]
    expect(minDateForMilestone(items, 0, today)).toBe(today)
  })

  it("returns the previous deadline + 1 day for the second milestone", () => {
    const items = [withDeadline("2026-05-07"), withDeadline("")]
    expect(minDateForMilestone(items, 1, today)).toBe("2026-05-08")
  })

  it("uses the latest previous deadline when multiple are set", () => {
    const items = [
      withDeadline("2026-05-07"),
      withDeadline("2026-05-14"),
      withDeadline(""),
    ]
    expect(minDateForMilestone(items, 2, today)).toBe("2026-05-15")
  })

  it("clamps to today when previous + 1 day is in the past", () => {
    // A leftover historical deadline shouldn't let the picker drop
    // below today — defence in depth against bad seed data.
    const items = [withDeadline("2024-01-01"), withDeadline("")]
    expect(minDateForMilestone(items, 1, today)).toBe(today)
  })

  it("handles month boundary correctly (Jan 31 -> Feb 1)", () => {
    const items = [withDeadline("2026-01-31"), withDeadline("")]
    expect(minDateForMilestone(items, 1, "2026-01-01")).toBe("2026-02-01")
  })

  it("handles year boundary correctly (Dec 31 -> Jan 1)", () => {
    const items = [withDeadline("2026-12-31"), withDeadline("")]
    expect(minDateForMilestone(items, 1, "2026-01-01")).toBe("2027-01-01")
  })
})
