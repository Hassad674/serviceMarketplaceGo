/**
 * Tests for the candidates list filter chips (Fix 3).
 *
 * Asserts:
 *  - all four chips render (Tous / Freelances / Agences / Apporteurs)
 *  - clicking a chip writes ?kind=... to the URL via router.replace
 *  - clicking "Tous" removes the kind query param
 *  - the current ?kind= URL value drives the active chip
 *  - the hook is called with the matching kindFilter argument
 */
import { describe, expect, it, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"

import { CandidatesList } from "../candidates-list"

const messages = {
  opportunity: {
    candidates: "Candidatures",
    filterAllCandidates: "Tous",
    filterFreelances: "Freelances",
    filterAgencies: "Agences",
    filterReferrers: "Apporteurs d'affaires",
  },
  job: {
    jobDetail_w08_eyebrow: "ATELIER · CANDIDATURES",
    jobDetail_w08_count: "{count, plural, =0 {Aucune candidature} other {# candidatures}}",
    jobDetail_w08_emptyTitle: "Aucune candidature pour le moment.",
    jobDetail_w08_emptyBody: "Patiente.",
    jobDetail_w08_emptyCta: "Voir mes annonces",
  },
}

const mockReplace = vi.fn()
let mockSearchString = ""
const mockListCalls: Array<{ jobId: string; cursor?: string; kind?: string }> = []

vi.mock("next/navigation", () => ({
  useSearchParams: () => new URLSearchParams(mockSearchString),
}))

vi.mock("@i18n/navigation", () => ({
  Link: ({ href, children, className }: { href: string; children: React.ReactNode; className?: string }) => (
    <a href={href} className={className}>
      {children}
    </a>
  ),
  useRouter: () => ({ replace: mockReplace }),
  usePathname: () => "/jobs/job-1",
}))

vi.mock("../../hooks/use-job-applications", () => ({
  useJobApplications: (jobId: string, cursor: string | undefined, kind: string | undefined) => {
    mockListCalls.push({ jobId, cursor, kind })
    return { data: { data: [], next_cursor: "", has_more: false }, isLoading: false }
  },
}))

vi.mock("../candidate-card", () => ({
  CandidateCard: () => null,
}))

vi.mock("../candidate-detail-panel", () => ({
  CandidateDetailPanel: () => null,
}))

function renderList() {
  return render(
    <NextIntlClientProvider locale="fr" messages={messages}>
      <CandidatesList jobId="job-1" />
    </NextIntlClientProvider>,
  )
}

beforeEach(() => {
  mockReplace.mockClear()
  mockListCalls.length = 0
  mockSearchString = ""
})

describe("CandidatesList · kind filter", () => {
  it("renders all four filter chips", () => {
    renderList()
    expect(screen.getByRole("tab", { name: "Tous" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Freelances" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Agences" })).toBeInTheDocument()
    expect(screen.getByRole("tab", { name: "Apporteurs d'affaires" })).toBeInTheDocument()
  })

  it("defaults the hook to undefined kind when no ?kind= is set", () => {
    renderList()
    const last = mockListCalls.at(-1)!
    expect(last.kind).toBeUndefined()
  })

  it("forwards ?kind=referrer to the hook when present", () => {
    mockSearchString = "kind=referrer"
    renderList()
    const last = mockListCalls.at(-1)!
    expect(last.kind).toBe("referrer")
  })

  it("falls back to 'all' when ?kind= holds an unknown value", () => {
    mockSearchString = "kind=hacker"
    renderList()
    expect(mockListCalls.at(-1)!.kind).toBeUndefined()
    // Active chip should be "Tous" (aria-selected=true)
    const tousChip = screen.getByRole("tab", { name: "Tous" })
    expect(tousChip.getAttribute("aria-selected")).toBe("true")
  })

  it("marks the matching chip aria-selected when ?kind= is set", () => {
    mockSearchString = "kind=agency"
    renderList()
    const agencyChip = screen.getByRole("tab", { name: "Agences" })
    expect(agencyChip.getAttribute("aria-selected")).toBe("true")
    const tousChip = screen.getByRole("tab", { name: "Tous" })
    expect(tousChip.getAttribute("aria-selected")).toBe("false")
  })

  it("writes ?kind=freelance to the URL when the Freelances chip is clicked", () => {
    renderList()
    fireEvent.click(screen.getByRole("tab", { name: "Freelances" }))
    expect(mockReplace).toHaveBeenCalledTimes(1)
    expect(mockReplace.mock.calls[0][0]).toContain("kind=freelance")
  })

  it("removes ?kind= when the 'Tous' chip is clicked", () => {
    mockSearchString = "kind=referrer"
    renderList()
    fireEvent.click(screen.getByRole("tab", { name: "Tous" }))
    expect(mockReplace).toHaveBeenCalledTimes(1)
    // No kind= in the URL after the click; pathname-only URL means
    // the implementation passes the bare path.
    expect(mockReplace.mock.calls[0][0]).not.toContain("kind=")
  })

  it("clears the selected candidate when filter changes", () => {
    mockSearchString = "kind=freelance&candidate=app-1"
    renderList()
    fireEvent.click(screen.getByRole("tab", { name: "Agences" }))
    expect(mockReplace).toHaveBeenCalledTimes(1)
    const url = mockReplace.mock.calls[0][0]
    expect(url).toContain("kind=agency")
    expect(url).not.toContain("candidate=")
  })
})
