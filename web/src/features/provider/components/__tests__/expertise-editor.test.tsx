import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import messages from "@/../messages/en.json"
import { ExpertiseEditor } from "../expertise-editor"

const mockMutateAsync = vi.fn()

vi.mock("../../hooks/use-update-expertise", () => ({
  useUpdateExpertiseDomains: () => ({
    mutateAsync: (value: string[]) => mockMutateAsync(value),
    isPending: false,
  }),
}))

function renderEditor(
  props: Partial<Parameters<typeof ExpertiseEditor>[0]> = {},
) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  const defaults = {
    domains: [] as string[],
    orgType: "agency",
    readOnly: false,
  }
  return render(
    <QueryClientProvider client={queryClient}>
      <NextIntlClientProvider locale="en" messages={messages}>
        <ExpertiseEditor {...defaults} {...props} />
      </NextIntlClientProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  mockMutateAsync.mockResolvedValue({
    expertise_domains: [],
  })
})

describe("ExpertiseEditor", () => {
  it("renders nothing when the org type is enterprise", () => {
    const { container } = renderEditor({ orgType: "enterprise" })
    expect(container).toBeEmptyDOMElement()
  })

  it("renders nothing when readOnly and the domain list is empty", () => {
    const { container } = renderEditor({
      readOnly: true,
      domains: [],
    })
    expect(container).toBeEmptyDOMElement()
  })

  it("renders the section title and subtitle with the org max", () => {
    renderEditor({ orgType: "agency" })
    expect(
      screen.getByRole("heading", {
        name: messages.profile.expertise.sectionTitle,
      }),
    ).toBeInTheDocument()
    expect(
      screen.getByText(/Pick up to 8 domains/i),
    ).toBeInTheDocument()
  })

  it("uses the provider_personal max of 5", () => {
    renderEditor({ orgType: "provider_personal" })
    expect(
      screen.getByText(/Pick up to 5 domains/i),
    ).toBeInTheDocument()
  })

  it("toggles a pill on click and enables the Save button", async () => {
    const user = userEvent.setup()
    renderEditor({ orgType: "agency" })

    const developmentPill = screen.getByRole("button", {
      name: messages.profile.expertise.domains.development,
    })
    expect(developmentPill).toHaveAttribute("aria-pressed", "false")

    await user.click(developmentPill)
    expect(developmentPill).toHaveAttribute("aria-pressed", "true")

    const saveButton = screen.getByRole("button", {
      name: new RegExp(messages.profile.expertise.save, "i"),
    })
    expect(saveButton).not.toBeDisabled()
  })

  it("disables unselected pills when the maximum is reached", async () => {
    const user = userEvent.setup()
    renderEditor({ orgType: "provider_personal" })

    const keys = [
      "development",
      "data_ai_ml",
      "design_ui_ux",
      "design_3d_animation",
      "video_motion",
    ] as const
    for (const key of keys) {
      await user.click(
        screen.getByRole("button", {
          name: messages.profile.expertise.domains[key],
        }),
      )
    }

    const unselected = screen.getByRole("button", {
      name: messages.profile.expertise.domains.photo_audiovisual,
    })
    expect(unselected).toBeDisabled()
    expect(
      screen.getByText(/reached the maximum of 5 domains/i),
    ).toBeInTheDocument()
  })

  it("calls the mutation with the selected list on Save", async () => {
    const user = userEvent.setup()
    renderEditor({ orgType: "agency" })

    await user.click(
      screen.getByRole("button", {
        name: messages.profile.expertise.domains.development,
      }),
    )
    await user.click(
      screen.getByRole("button", {
        name: messages.profile.expertise.domains.legal,
      }),
    )

    await user.click(
      screen.getByRole("button", {
        name: new RegExp(messages.profile.expertise.save, "i"),
      }),
    )

    await waitFor(() =>
      expect(mockMutateAsync).toHaveBeenCalledWith([
        "development",
        "legal",
      ]),
    )
  })

  it("renders selected domains as read-only pills when readOnly is true", () => {
    renderEditor({
      readOnly: true,
      domains: ["development", "legal"],
    })
    expect(
      screen.getByText(messages.profile.expertise.domains.development),
    ).toBeInTheDocument()
    expect(
      screen.getByText(messages.profile.expertise.domains.legal),
    ).toBeInTheDocument()
    expect(
      screen.queryByRole("button", {
        name: new RegExp(messages.profile.expertise.save, "i"),
      }),
    ).toBeNull()
  })
})
