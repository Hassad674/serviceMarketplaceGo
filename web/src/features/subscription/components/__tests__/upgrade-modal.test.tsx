import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import type { ReactElement } from "react"
import { useState } from "react"
import { UpgradeModal } from "../upgrade-modal"

// Mock next-intl translations.
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

// Mock @i18n/navigation router.
const pushFn = vi.fn()
vi.mock("@i18n/navigation", () => ({
  useRouter: () => ({ push: pushFn }),
}))

// Mock the shared Modal so backdrop / portal plumbing doesn't get in
// the way of asserting on content.
vi.mock("@/shared/components/ui/modal", () => ({
  Modal: ({
    open,
    title,
    children,
  }: {
    open: boolean
    title: string
    children: React.ReactNode
  }) => (open ? <div data-testid="modal" aria-label={title}>{children}</div> : null),
}))

function renderModal(role: "freelance" | "agency" = "freelance"): ReactElement {
  return <UpgradeModal open role={role} onClose={() => {}} />
}

describe("UpgradeModal", () => {
  beforeEach(() => {
    pushFn.mockReset()
  })

  it("renders the freelance plan card with monthly cycle by default", () => {
    render(renderModal("freelance"))
    expect(screen.getByText("Premium Freelance")).toBeDefined()
    expect(screen.getAllByText(/19 €/).length).toBeGreaterThan(0)
  })

  it("renders the agency plan card when role=agency", () => {
    render(renderModal("agency"))
    expect(screen.getByText("Premium Agence")).toBeDefined()
    expect(screen.getAllByText(/49 €/).length).toBeGreaterThan(0)
  })

  it("switches to annual pricing when the annual tab is clicked", () => {
    render(renderModal("freelance"))
    const annualTab = screen.getByRole("tab", { name: /annuel/i })
    fireEvent.click(annualTab)
    expect(screen.getByText(/15 €/)).toBeDefined() // freelance per-month annual
    expect(screen.getByText(/180 €\/an/)).toBeDefined()
  })

  it("redirects to /subscribe/embed with plan + cycle + auto_renew query on submit", () => {
    render(renderModal("freelance"))
    const submit = screen.getByRole("button", { name: /Continuer/i })
    fireEvent.click(submit)
    expect(pushFn).toHaveBeenCalledTimes(1)
    const url = pushFn.mock.calls[0][0] as string
    expect(url).toContain("/subscribe/embed?")
    expect(url).toContain("plan=freelance")
    expect(url).toContain("cycle=monthly")
    expect(url).toContain("auto_renew=false")
  })

  it("forwards auto_renew=true after the user toggles the checkbox", () => {
    render(renderModal("agency"))
    const autoRenew = screen.getByRole("checkbox")
    fireEvent.click(autoRenew)
    fireEvent.click(screen.getByRole("button", { name: /Continuer/i }))
    expect(pushFn).toHaveBeenCalledTimes(1)
    const url = pushFn.mock.calls[0][0] as string
    expect(url).toContain("plan=agency")
    expect(url).toContain("auto_renew=true")
  })

  it("resets the pending CTA when the modal is reopened after a previous submit", () => {
    // Modal is mounted by the layout-level header — submit calls
    // onClose() which sets the parent's `open` to false, navigates,
    // and leaves UpgradeModal mounted with `pending=true`. Reopening
    // (browser back, click Premium badge again) must reset the CTA
    // to "Continuer", not leave it stuck on "Redirection…".
    function Harness() {
      const [open, setOpen] = useState(true)
      return (
        <>
          <button type="button" onClick={() => setOpen((v) => !v)}>
            toggle
          </button>
          <UpgradeModal open={open} role="freelance" onClose={() => setOpen(false)} />
        </>
      )
    }
    render(<Harness />)
    // First open: submit. handleSubmit calls onClose() synchronously
    // so the modal hides immediately — but pending=true is set on
    // UpgradeModal's state right before onClose, and persists.
    fireEvent.click(screen.getByRole("button", { name: /Continuer/i }))
    // Modal is now hidden; verify it.
    expect(screen.queryByTestId("modal")).toBeNull()
    // Reopen (e.g. user came back via browser back).
    fireEvent.click(screen.getByRole("button", { name: "toggle" }))
    // CTA must be back to "Continuer", not stuck on "Redirection…".
    const cta = screen.getByRole("button", { name: /Continuer/i })
    expect(cta).toBeDefined()
    expect((cta as HTMLButtonElement).disabled).toBe(false)
    expect(screen.queryByRole("button", { name: /Redirection/i })).toBeNull()
  })
})
