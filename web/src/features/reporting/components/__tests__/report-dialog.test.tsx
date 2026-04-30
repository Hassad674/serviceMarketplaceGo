import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { ReportDialog } from "../report-dialog"

const mockMutate = vi.fn()
const toastSuccess = vi.fn()
const toastError = vi.fn()

vi.mock("../../hooks/use-report", () => ({
  useCreateReport: () => ({
    mutate: mockMutate,
    isPending: false,
  }),
}))

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

vi.mock("sonner", () => ({
  toast: {
    success: (...args: unknown[]) => toastSuccess(...args),
    error: (...args: unknown[]) => toastError(...args),
  },
}))

beforeEach(() => {
  vi.clearAllMocks()
})

describe("ReportDialog — visibility", () => {
  it("renders nothing when open=false", () => {
    const { container } = render(
      <ReportDialog
        open={false}
        onClose={vi.fn()}
        targetType="user"
        targetId="u-1"
      />,
    )
    expect(container.firstChild).toBeNull()
  })

  it("renders dialog when open=true", () => {
    render(
      <ReportDialog
        open={true}
        onClose={vi.fn()}
        targetType="user"
        targetId="u-1"
      />,
    )
    expect(screen.getByRole("dialog")).toBeInTheDocument()
  })
})

describe("ReportDialog — heading per targetType", () => {
  it.each(["message", "user", "job", "application"] as const)(
    "renders the right title for %s",
    (type) => {
      render(
        <ReportDialog
          open={true}
          onClose={vi.fn()}
          targetType={type}
          targetId="x"
        />,
      )
      // The title depends on the type — check that one of the labels is present
      const headings = {
        message: "reportMessage",
        user: "reportUser",
        job: "reportJob",
        application: "reportApplication",
      }
      expect(screen.getByText(headings[type])).toBeInTheDocument()
    },
  )
})

describe("ReportDialog — reasons list", () => {
  it("shows 5 reasons for message target", () => {
    render(
      <ReportDialog
        open={true}
        onClose={vi.fn()}
        targetType="message"
        targetId="x"
      />,
    )
    const labels = screen.getAllByText(/^reason_/)
    expect(labels.length).toBe(5)
  })

  it("shows 6 reasons for user target", () => {
    render(
      <ReportDialog
        open={true}
        onClose={vi.fn()}
        targetType="user"
        targetId="x"
      />,
    )
    const labels = screen.getAllByText(/^reason_/)
    expect(labels.length).toBe(6)
  })

  it("shows 5 reasons for job target", () => {
    render(
      <ReportDialog
        open={true}
        onClose={vi.fn()}
        targetType="job"
        targetId="x"
      />,
    )
    const labels = screen.getAllByText(/^reason_/)
    expect(labels.length).toBe(5)
  })

  it("shows 4 reasons for application target", () => {
    render(
      <ReportDialog
        open={true}
        onClose={vi.fn()}
        targetType="application"
        targetId="x"
      />,
    )
    const labels = screen.getAllByText(/^reason_/)
    expect(labels.length).toBe(4)
  })
})

describe("ReportDialog — submission", () => {
  it("disables submit when no reason is selected", () => {
    render(
      <ReportDialog
        open={true}
        onClose={vi.fn()}
        targetType="user"
        targetId="u-1"
      />,
    )
    const btn = screen.getByText("submitReport").closest("button")!
    expect(btn.disabled).toBe(true)
  })

  it("enables submit and calls mutation when a reason is selected", () => {
    const onClose = vi.fn()
    mockMutate.mockImplementation((_data, opts) => {
      opts?.onSuccess?.()
    })
    render(
      <ReportDialog
        open={true}
        onClose={onClose}
        targetType="user"
        targetId="u-1"
        conversationId="c-1"
      />,
    )
    fireEvent.click(screen.getByText("reason_harassment"))
    fireEvent.click(screen.getByText("submitReport"))

    expect(mockMutate).toHaveBeenCalled()
    const arg = mockMutate.mock.calls[0][0]
    expect(arg.target_id).toBe("u-1")
    expect(arg.target_type).toBe("user")
    expect(arg.reason).toBe("harassment")
    expect(arg.conversation_id).toBe("c-1")
    expect(toastSuccess).toHaveBeenCalledWith("reportSent")
    expect(onClose).toHaveBeenCalled()
  })

  it("shows error toast on mutation error (non-409)", () => {
    mockMutate.mockImplementation((_data, opts) => {
      opts?.onError?.(new Error("boom"))
    })
    render(
      <ReportDialog
        open={true}
        onClose={vi.fn()}
        targetType="user"
        targetId="u-1"
      />,
    )
    fireEvent.click(screen.getByText("reason_harassment"))
    fireEvent.click(screen.getByText("submitReport"))
    expect(toastError).toHaveBeenCalledWith("reportError")
  })

  it("uses empty string conversation_id when none provided", () => {
    mockMutate.mockImplementation(() => {})
    render(
      <ReportDialog
        open={true}
        onClose={vi.fn()}
        targetType="user"
        targetId="u-1"
      />,
    )
    fireEvent.click(screen.getByText("reason_harassment"))
    fireEvent.click(screen.getByText("submitReport"))
    expect(mockMutate.mock.calls[0][0].conversation_id).toBe("")
  })

  it("submits the description text the user typed", () => {
    mockMutate.mockImplementation(() => {})
    render(
      <ReportDialog
        open={true}
        onClose={vi.fn()}
        targetType="user"
        targetId="u-1"
      />,
    )
    fireEvent.click(screen.getByText("reason_harassment"))
    fireEvent.change(screen.getByPlaceholderText("descriptionPlaceholder"), {
      target: { value: "details here" },
    })
    fireEvent.click(screen.getByText("submitReport"))
    expect(mockMutate.mock.calls[0][0].description).toBe("details here")
  })
})

describe("ReportDialog — close behavior", () => {
  it("invokes onClose when clicking the X button", () => {
    const onClose = vi.fn()
    render(
      <ReportDialog
        open={true}
        onClose={onClose}
        targetType="user"
        targetId="u-1"
      />,
    )
    // The X icon is inside an SVG; click the button containing it
    const buttons = screen.getAllByRole("button")
    const xBtn = buttons.find((b) => b.querySelector("svg.lucide-x"))
    expect(xBtn).toBeDefined()
    fireEvent.click(xBtn!)
    expect(onClose).toHaveBeenCalled()
  })

  it("invokes onClose when clicking the backdrop", () => {
    const onClose = vi.fn()
    render(
      <ReportDialog
        open={true}
        onClose={onClose}
        targetType="user"
        targetId="u-1"
      />,
    )
    // Click the outermost div (backdrop). The dialog stops propagation.
    const dialog = screen.getByRole("dialog")
    fireEvent.click(dialog.parentElement!)
    expect(onClose).toHaveBeenCalled()
  })
})
