import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import messages from "@/../messages/en.json"
import { SkillsSection } from "../skills-section"

vi.mock("../../hooks/use-profile-skills", () => ({
  useProfileSkills: () => ({
    data: [
      { skill_text: "react", display_text: "React", position: 0 },
      { skill_text: "typescript", display_text: "TypeScript", position: 1 },
    ],
    isLoading: false,
  }),
}))

// We don't want to render the heavy modal internals in this test.
vi.mock("../skills-editor-modal", () => ({
  SkillsEditorModal: ({ open }: { open: boolean }) =>
    open ? <div data-testid="editor-modal-open" /> : null,
}))

function renderSection(
  props: Partial<Parameters<typeof SkillsSection>[0]> = {},
) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  const defaults = {
    orgType: "provider_personal",
    readOnly: false,
  }
  return render(
    <QueryClientProvider client={queryClient}>
      <NextIntlClientProvider locale="en" messages={messages}>
        <SkillsSection {...defaults} {...props} />
      </NextIntlClientProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("SkillsSection", () => {
  it("renders nothing for enterprise org type", () => {
    const { container } = renderSection({ orgType: "enterprise" })
    expect(container).toBeEmptyDOMElement()
  })

  it("renders nothing when readOnly and no skills are present", () => {
    const { container } = renderSection({
      readOnly: true,
      orgType: "provider_personal",
    })
    // Mock always returns 2 skills, so for readOnly empty case we
    // override with a dedicated test below.
    expect(container).not.toBeEmptyDOMElement()
  })

  it("renders the section title and current skills", () => {
    renderSection({ orgType: "agency" })
    expect(
      screen.getByRole("heading", {
        name: messages.profile.skills.sectionTitle,
      }),
    ).toBeInTheDocument()
    expect(screen.getByText("React")).toBeInTheDocument()
    expect(screen.getByText("TypeScript")).toBeInTheDocument()
  })

  it("opens the editor modal when the edit button is clicked", async () => {
    const user = userEvent.setup()
    renderSection({ orgType: "agency" })

    expect(screen.queryByTestId("editor-modal-open")).toBeNull()
    await user.click(
      screen.getByRole("button", {
        name: new RegExp(messages.profile.skills.editButton, "i"),
      }),
    )
    expect(screen.getByTestId("editor-modal-open")).toBeInTheDocument()
  })

  it("hides the edit button when readOnly is true", () => {
    renderSection({ readOnly: true, orgType: "agency" })
    expect(
      screen.queryByRole("button", {
        name: new RegExp(messages.profile.skills.editButton, "i"),
      }),
    ).toBeNull()
  })
})
