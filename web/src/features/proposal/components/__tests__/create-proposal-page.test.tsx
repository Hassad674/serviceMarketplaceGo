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
  useRouter: () => ({ push: pushFn, back: backFn, replace: () => {}, refresh: () => {}, prefetch: () => {} }),
  usePathname: () => "/",
}))

// Mock @i18n/navigation
const pushFn = vi.fn()
const backFn = vi.fn()
vi.mock("@i18n/navigation", () => ({
  useRouter: () => ({ push: pushFn, back: backFn }),
}))

// Mock lucide-react icons
vi.mock("lucide-react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("lucide-react")>()
  return {
    ...actual,
    X: (props: Record<string, unknown>) => <span data-testid="x-icon" {...props} />,
    Loader2: (props: Record<string, unknown>) => <span data-testid="loader-icon" {...props} />,
    Euro: (props: Record<string, unknown>) => <span data-testid="euro-icon" {...props} />,
    Calendar: (props: Record<string, unknown>) => <span data-testid="calendar-icon" {...props} />,
    Paperclip: (props: Record<string, unknown>) => <span data-testid="paperclip-icon" {...props} />,
    User: (props: Record<string, unknown>) => <span data-testid="user-icon" {...props} />,
  }
})

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
    client_id: "recipient-1",
    provider_id: "user-self",
    client_name: "Acme Corp",
    provider_name: "Jane Freelance",
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

  it("uses the `name` query param as the displayed recipient name", () => {
    mockSearchParams.set("name", "Alice Smith")

    render(<CreateProposalPage />)

    expect(screen.getByText("Alice Smith")).toBeDefined()
  })

  it("renders the empty placeholder when no name query param and not modifying", () => {
    // No `name` set, only `to=recipient-1` from beforeEach. The
    // legacy `User <id>` fallback only runs when the query params
    // change after mount; on initial mount the recipient field shows
    // the em-dash placeholder until the name resolves.
    render(<CreateProposalPage />)

    expect(screen.getByText("—")).toBeDefined()
  })

  it("resolves recipient name from client_name when modifying and recipientId matches client_id", async () => {
    mockSearchParams.set("modify", "proposal-123")
    mockSearchParams.set("to", "recipient-1") // matches mocked client_id
    mockSearchParams.delete("name")

    render(<CreateProposalPage />)

    // After the proposal fetch resolves, the participant name from
    // the backend (client_name) is rendered — never a `User XXX`
    // placeholder.
    await screen.findByText("Acme Corp")
  })

  it("renders the global amount input when payment mode is one_time (default)", () => {
    render(<CreateProposalPage />)

    // Default mode is one_time — the global Montant € input must be present.
    expect(screen.queryByLabelText(/proposalAmount/)).not.toBeNull()
  })

  it("hides the global amount input when payment mode is milestone", () => {
    render(<CreateProposalPage />)

    // Switch to milestone mode by clicking the milestone tab.
    // PaymentModeToggle renders the option as a button with the
    // translation key "milestone" as its label.
    fireEvent.click(screen.getByText("milestone"))

    // In milestone mode each milestone has its own amount input,
    // so the global "Montant €" input must NOT be in the DOM.
    expect(screen.queryByLabelText(/proposalAmount/)).toBeNull()
    // The mode-panel id used by aria-controls also disappears.
    expect(document.getElementById("payment-mode-panel-one_time")).toBeNull()
  })

  it("hides global title, description and deadline inputs when payment mode is milestone", () => {
    render(<CreateProposalPage />)

    // Defaults to one_time — the four global inputs are visible.
    expect(screen.queryByLabelText(/proposalTitle/)).not.toBeNull()
    expect(screen.queryByLabelText(/proposalDescription/)).not.toBeNull()
    expect(screen.queryByLabelText(/proposalDeadline/)).not.toBeNull()

    // Toggle to milestone — the global title/description/deadline
    // inputs vanish (Contra-style: per-milestone fields replace them,
    // proposal-level fields are derived at submit time).
    fireEvent.click(screen.getByText("milestone"))

    expect(screen.queryByLabelText(/proposalTitle/)).toBeNull()
    expect(screen.queryByLabelText(/proposalDescription/)).toBeNull()
    expect(screen.queryByLabelText(/proposalDeadline/)).toBeNull()
  })

  it("renders 2 empty milestone rows by default when toggling into milestone mode", () => {
    render(<CreateProposalPage />)

    // Switch to milestone mode.
    fireEvent.click(screen.getByText("milestone"))

    // Two milestone titles should be rendered (one per row). The aria
    // label includes the sequence number, so we can target each one.
    const m1 = screen.getByLabelText(
      /milestone 1 titleAriaLabel/,
    )
    const m2 = screen.getByLabelText(
      /milestone 2 titleAriaLabel/,
    )
    expect(m1).not.toBeNull()
    expect(m2).not.toBeNull()

    // No third milestone — the floor is exactly 2 and adding more is
    // explicit (the user clicks "Add a milestone").
    expect(
      screen.queryByLabelText(
        /milestone 3 titleAriaLabel/,
      ),
    ).toBeNull()
  })

  it("submit button is disabled in milestone mode while milestones are incomplete", () => {
    render(<CreateProposalPage />)

    fireEvent.click(screen.getByText("milestone"))

    // Empty milestones — submit must be disabled even though the
    // global title/description inputs are no longer required.
    const submitButtons = screen.getAllByText("proposalSend")
    const headerSubmit = submitButtons[0].closest("button")
    expect(headerSubmit?.disabled).toBe(true)
  })

  it("submit button is enabled when both milestones are filled (min 2)", () => {
    render(<CreateProposalPage />)

    fireEvent.click(screen.getByText("milestone"))

    // Fill both milestones with title + amount (description is
    // optional in milestone mode).
    fireEvent.change(
      screen.getByLabelText(
        /milestone 1 titleAriaLabel/,
      ),
      { target: { value: "Phase 1" } },
    )
    fireEvent.change(
      screen.getByLabelText(
        /milestone 1 amountAriaLabel/,
      ),
      { target: { value: "1000" } },
    )
    fireEvent.change(
      screen.getByLabelText(
        /milestone 2 titleAriaLabel/,
      ),
      { target: { value: "Phase 2" } },
    )
    fireEvent.change(
      screen.getByLabelText(
        /milestone 2 amountAriaLabel/,
      ),
      { target: { value: "2000" } },
    )

    const submitButtons = screen.getAllByText("proposalSend")
    const headerSubmit = submitButtons[0].closest("button")
    expect(headerSubmit?.disabled).toBe(false)
  })

  it("submits the milestones array and a derived title in milestone mode", () => {
    render(<CreateProposalPage />)

    fireEvent.click(screen.getByText("milestone"))

    fireEvent.change(
      screen.getByLabelText(
        /milestone 1 titleAriaLabel/,
      ),
      { target: { value: "Maquettes" } },
    )
    fireEvent.change(
      screen.getByLabelText(
        /milestone 1 amountAriaLabel/,
      ),
      { target: { value: "1500" } },
    )
    fireEvent.change(
      screen.getByLabelText(
        /milestone 2 titleAriaLabel/,
      ),
      { target: { value: "Intégration" } },
    )
    fireEvent.change(
      screen.getByLabelText(
        /milestone 2 amountAriaLabel/,
      ),
      { target: { value: "3500" } },
    )

    const form = document.getElementById("proposal-form") as HTMLFormElement
    fireEvent.submit(form)

    expect(createMutateFn).toHaveBeenCalledOnce()
    const callArgs = createMutateFn.mock.calls[0][0]
    expect(callArgs.payment_mode).toBe("milestone")
    expect(callArgs.milestones).toHaveLength(2)
    expect(callArgs.milestones[0].sequence).toBe(1)
    expect(callArgs.milestones[0].title).toBe("Maquettes")
    expect(callArgs.milestones[0].amount).toBe(150000) // 1500 EUR in centimes
    expect(callArgs.milestones[1].sequence).toBe(2)
    expect(callArgs.milestones[1].amount).toBe(350000)
    // Total amount derived from the milestone sum (no global amount
    // input was filled in milestone mode).
    expect(callArgs.amount).toBe(500000)
    // Title derived from the first milestone title — the global
    // proposalTitle input is not rendered in milestone mode so the
    // form synthesises a project title at submit time.
    expect(callArgs.title).toBe("Maquettes")
  })

  it("toggling back to one_time restores the global inputs and drops milestone-mode payload", () => {
    render(<CreateProposalPage />)

    // Toggle into milestone mode then back.
    fireEvent.click(screen.getByText("milestone"))
    fireEvent.click(screen.getByText("oneTime"))

    // Global inputs are visible again.
    expect(screen.queryByLabelText(/proposalTitle/)).not.toBeNull()
    expect(screen.queryByLabelText(/proposalDescription/)).not.toBeNull()
    expect(screen.queryByLabelText(/proposalAmount/)).not.toBeNull()

    // Fill the legacy global inputs and submit.
    fireEvent.change(screen.getByLabelText(/proposalTitle/), {
      target: { value: "Site web" },
    })
    fireEvent.change(screen.getByLabelText(/proposalDescription/), {
      target: { value: "Refonte complète" },
    })
    fireEvent.change(screen.getByLabelText(/proposalAmount/), {
      target: { value: "1500" },
    })

    const form = document.getElementById("proposal-form") as HTMLFormElement
    fireEvent.submit(form)

    expect(createMutateFn).toHaveBeenCalledOnce()
    const callArgs = createMutateFn.mock.calls[0][0]
    expect(callArgs.payment_mode).toBe("one_time")
    // milestones are undefined in one_time mode (legacy contract).
    expect(callArgs.milestones).toBeUndefined()
    expect(callArgs.amount).toBe(150000)
    expect(callArgs.title).toBe("Site web")
  })
})
