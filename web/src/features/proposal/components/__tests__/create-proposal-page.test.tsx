import { describe, it, expect, vi, beforeEach } from "vitest"
import { render as baseRender, screen, fireEvent } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactElement } from "react"
import { CreateProposalPage } from "../create-proposal-page"

// Grant every permission by default so the submit flow is exercisable
// in isolation. Tests that care about permission denial can override
// with `vi.mocked(useHasPermission).mockReturnValueOnce(false)`.
vi.mock("@/shared/hooks/use-permissions", () => ({
  useHasPermission: () => true,
}))

// CreateProposalPage reads `useHasPermission` → `useOrganization` →
// `useQuery` from the org-permissions system. Every render must sit
// inside a TanStack QueryClientProvider so the hook graph resolves.
function render(ui: ReactElement) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return baseRender(
    <QueryClientProvider client={client}>{ui}</QueryClientProvider>,
  )
}

// Mock next-intl
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

// Mock next/navigation (useSearchParams)
const mockSearchParams = new Map<string, string>()
vi.mock("next/navigation", () => ({
  useSearchParams: () => ({
    get: (key: string) => mockSearchParams.get(key) ?? null,
  }),
}))

// Mock @i18n/navigation
const pushFn = vi.fn()
const backFn = vi.fn()
vi.mock("@i18n/navigation", () => ({
  useRouter: () => ({ push: pushFn, back: backFn }),
}))

// Mock lucide-react icons
vi.mock("lucide-react", () => ({
  X: (props: Record<string, unknown>) => <span data-testid="x-icon" {...props} />,
  Loader2: (props: Record<string, unknown>) => <span data-testid="loader-icon" {...props} />,
  Euro: (props: Record<string, unknown>) => <span data-testid="euro-icon" {...props} />,
  Calendar: (props: Record<string, unknown>) => <span data-testid="calendar-icon" {...props} />,
  Paperclip: (props: Record<string, unknown>) => <span data-testid="paperclip-icon" {...props} />,
  User: (props: Record<string, unknown>) => <span data-testid="user-icon" {...props} />,
}))

// Mock @/shared/lib/utils
vi.mock("@/shared/lib/utils", () => ({
  cn: (...classes: unknown[]) => classes.filter(Boolean).join(" "),
}))

// Mock sub-components
vi.mock("../proposal-preview", () => ({
  ProposalPreview: () => <div data-testid="proposal-preview" />,
}))

vi.mock("../file-drop-zone", () => ({
  FileDropZone: ({ onFilesChange }: { files: File[]; onFilesChange: (f: File[]) => void }) => (
    <div data-testid="file-drop-zone">
      <button onClick={() => onFilesChange([new File([""], "test.pdf")])}>add file</button>
    </div>
  ),
}))

// Mock proposal hooks
const createMutateFn = vi.fn()
const modifyMutateFn = vi.fn()
vi.mock("../../hooks/use-proposals", () => ({
  useCreateProposal: () => ({
    mutate: createMutateFn,
    isPending: false,
  }),
  useModifyProposal: () => ({
    mutate: modifyMutateFn,
    isPending: false,
  }),
}))

// Mock proposal API
vi.mock("../../api/proposal-api", () => ({
  getProposal: vi.fn().mockResolvedValue({
    title: "Existing proposal",
    description: "Existing description",
    amount: 250000,
    deadline: "2026-06-15T00:00:00Z",
  }),
}))

describe("CreateProposalPage", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockSearchParams.clear()
    mockSearchParams.set("to", "recipient-1")
    mockSearchParams.set("conversation", "conv-1")
  })

  it("renders all form inputs", () => {
    render(<CreateProposalPage />)

    // Title input
    expect(screen.getByLabelText(/proposalTitle/)).toBeDefined()

    // Description textarea
    expect(screen.getByLabelText(/proposalDescription/)).toBeDefined()

    // Amount input
    expect(screen.getByLabelText(/proposalAmount/)).toBeDefined()

    // Deadline input
    expect(screen.getByLabelText(/proposalDeadline/)).toBeDefined()

    // File drop zone
    expect(screen.getByTestId("file-drop-zone")).toBeDefined()
  })

  it("shows recipient section", () => {
    render(<CreateProposalPage />)

    expect(screen.getByText("proposalRecipient")).toBeDefined()
  })

  it("submit button is disabled when required fields are empty", () => {
    render(<CreateProposalPage />)

    // The submit button (in header) should be disabled because title/description/amount are empty
    const submitButtons = screen.getAllByText("proposalSend")
    // Check header submit button (first one)
    const headerSubmit = submitButtons[0].closest("button")
    expect(headerSubmit?.disabled).toBe(true)
  })

  it("submit button is enabled when required fields are filled", () => {
    render(<CreateProposalPage />)

    // Fill in required fields
    fireEvent.change(screen.getByLabelText(/proposalTitle/), {
      target: { value: "Website redesign" },
    })
    fireEvent.change(screen.getByLabelText(/proposalDescription/), {
      target: { value: "Full redesign of the corporate website" },
    })
    fireEvent.change(screen.getByLabelText(/proposalAmount/), {
      target: { value: "5000" },
    })

    // The submit button should now be enabled
    const submitButtons = screen.getAllByText("proposalSend")
    const headerSubmit = submitButtons[0].closest("button")
    expect(headerSubmit?.disabled).toBe(false)
  })

  it("converts euros to centimes on submit", () => {
    render(<CreateProposalPage />)

    // Fill in required fields
    fireEvent.change(screen.getByLabelText(/proposalTitle/), {
      target: { value: "Website redesign" },
    })
    fireEvent.change(screen.getByLabelText(/proposalDescription/), {
      target: { value: "Full redesign" },
    })
    fireEvent.change(screen.getByLabelText(/proposalAmount/), {
      target: { value: "1500.50" },
    })

    // Submit the form
    const form = document.getElementById("proposal-form") as HTMLFormElement
    fireEvent.submit(form)

    // Verify the mutation was called with amount in centimes
    expect(createMutateFn).toHaveBeenCalledOnce()
    const callArgs = createMutateFn.mock.calls[0][0]
    expect(callArgs.amount).toBe(150050) // 1500.50 * 100
    expect(callArgs.title).toBe("Website redesign")
    expect(callArgs.description).toBe("Full redesign")
    expect(callArgs.recipient_id).toBe("recipient-1")
    expect(callArgs.conversation_id).toBe("conv-1")
  })

  it("calls back navigation when cancel button clicked", () => {
    render(<CreateProposalPage />)

    const cancelButton = screen.getByLabelText("proposalCancel")
    fireEvent.click(cancelButton)

    expect(backFn).toHaveBeenCalledOnce()
  })

  it("shows title character counter", () => {
    render(<CreateProposalPage />)

    // Initially 0/100
    expect(screen.getByText("0/100")).toBeDefined()

    // After typing
    fireEvent.change(screen.getByLabelText(/proposalTitle/), {
      target: { value: "Hello" },
    })
    expect(screen.getByText("5/100")).toBeDefined()
  })

  it("shows create header when not modifying", () => {
    render(<CreateProposalPage />)

    expect(screen.getByText("createProposal")).toBeDefined()
  })

  it("shows modify header when modify param is present", () => {
    mockSearchParams.set("modify", "proposal-123")

    render(<CreateProposalPage />)

    expect(screen.getByText("modify")).toBeDefined()
  })

  it("renders preview component", () => {
    render(<CreateProposalPage />)

    expect(screen.getByTestId("proposal-preview")).toBeDefined()
  })
})
