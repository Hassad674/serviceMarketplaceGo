import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { DeleteAccountCard } from "../delete-account-card"

// next-intl: pass through key as the rendered string.
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string, vars?: Record<string, unknown>) => {
    if (vars && Object.keys(vars).length > 0) {
      return `${key}:${JSON.stringify(vars)}`
    }
    return key
  },
}))

// gdpr API: we intercept the three functions and return controllable
// promises so each test can assert on the UI transitions.
const requestDeletionMock = vi.fn()
const cancelDeletionMock = vi.fn()
const downloadExportMock = vi.fn()

vi.mock("@/features/account/api/gdpr", () => ({
  requestDeletion: (...args: unknown[]) => requestDeletionMock(...args),
  cancelDeletion: () => cancelDeletionMock(),
  downloadExport: () => downloadExportMock(),
}))

// Mock the shared Modal so portal plumbing doesn't get in the way.
vi.mock("@/shared/components/ui/modal", () => ({
  Modal: ({
    open,
    title,
    children,
  }: {
    open: boolean
    title: string
    children: React.ReactNode
  }) =>
    open ? (
      <div role="dialog" aria-modal="true" aria-label={title}>
        {children}
      </div>
    ) : null,
}))

beforeEach(() => {
  requestDeletionMock.mockReset()
  cancelDeletionMock.mockReset()
  downloadExportMock.mockReset()
})

describe("DeleteAccountCard", () => {
  it("renders export + delete buttons for a healthy account", () => {
    render(<DeleteAccountCard pendingDeletionAt={null} hardDeleteAt={null} />)
    expect(screen.getByText("export.button")).toBeInTheDocument()
    expect(screen.getByText("delete.button")).toBeInTheDocument()
  })

  it("renders the pending-deletion banner when soft-deleted", () => {
    render(
      <DeleteAccountCard
        pendingDeletionAt="2026-05-01T12:00:00Z"
        hardDeleteAt="2026-05-31T12:00:00Z"
      />,
    )
    expect(screen.getByRole("alert")).toBeInTheDocument()
    expect(screen.getByText("cancelButton")).toBeInTheDocument()
  })

  it("hides the delete button while in cooldown", () => {
    render(
      <DeleteAccountCard
        pendingDeletionAt="2026-05-01T12:00:00Z"
        hardDeleteAt="2026-05-31T12:00:00Z"
      />,
    )
    expect(screen.queryByText("delete.button")).toBeNull()
  })

  it("calls downloadExport when the Export button is clicked", async () => {
    downloadExportMock.mockResolvedValue(undefined)
    render(<DeleteAccountCard pendingDeletionAt={null} hardDeleteAt={null} />)
    await userEvent.click(screen.getByText("export.button"))
    expect(downloadExportMock).toHaveBeenCalledTimes(1)
  })

  it("opens the delete modal on click", async () => {
    render(<DeleteAccountCard pendingDeletionAt={null} hardDeleteAt={null} />)
    await userEvent.click(screen.getByText("delete.button"))
    expect(screen.getByRole("dialog")).toBeInTheDocument()
  })

  it("submits the delete form with confirm + password", async () => {
    requestDeletionMock.mockResolvedValue({
      email_sent_to: "alice@example.com",
      expires_at: "2026-05-02T12:00:00Z",
    })
    render(<DeleteAccountCard pendingDeletionAt={null} hardDeleteAt={null} />)
    await userEvent.click(screen.getByText("delete.button"))

    const password = screen.getByLabelText(/passwordLabel/i)
    fireEvent.change(password, { target: { value: "correct" } })
    const confirm = screen.getByRole("checkbox")
    fireEvent.click(confirm)

    await userEvent.click(screen.getByRole("button", { name: /submit$/i }))

    await waitFor(() => {
      expect(requestDeletionMock).toHaveBeenCalledWith("correct")
    })
    expect(await screen.findByText("alice@example.com")).toBeInTheDocument()
  })

  it("calls cancelDeletion from the banner", async () => {
    cancelDeletionMock.mockResolvedValue({ cancelled: true })
    const onCancelled = vi.fn()
    render(
      <DeleteAccountCard
        pendingDeletionAt="2026-05-01T12:00:00Z"
        hardDeleteAt="2026-05-31T12:00:00Z"
        onCancelled={onCancelled}
      />,
    )
    await userEvent.click(screen.getByText("cancelButton"))
    await waitFor(() => {
      expect(cancelDeletionMock).toHaveBeenCalledTimes(1)
      expect(onCancelled).toHaveBeenCalledTimes(1)
    })
  })

  it("modal has dialog role + aria-modal=true (accessibility)", async () => {
    render(<DeleteAccountCard pendingDeletionAt={null} hardDeleteAt={null} />)
    await userEvent.click(screen.getByText("delete.button"))
    const dialog = screen.getByRole("dialog")
    expect(dialog.getAttribute("aria-modal")).toBe("true")
    expect(dialog.getAttribute("aria-label")).toBeTruthy()
  })

  it("delete submit button is disabled until confirm + password set", async () => {
    render(<DeleteAccountCard pendingDeletionAt={null} hardDeleteAt={null} />)
    await userEvent.click(screen.getByText("delete.button"))

    const submit = screen.getByRole("button", { name: /submit$/i })
    expect(submit).toBeDisabled()

    fireEvent.change(screen.getByLabelText(/passwordLabel/i), {
      target: { value: "x" },
    })
    expect(submit).toBeDisabled() // still disabled — confirm not checked

    fireEvent.click(screen.getByRole("checkbox"))
    expect(submit).not.toBeDisabled()
  })
})
