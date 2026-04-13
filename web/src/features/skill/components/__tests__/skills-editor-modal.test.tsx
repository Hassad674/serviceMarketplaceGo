import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import messages from "@/../messages/en.json"
import { SkillsEditorModal } from "../skills-editor-modal"

const mockMutateAsync = vi.fn()

// Stable references so the modal's `useEffect([persisted])` doesn't
// spin into an infinite render loop — the real hook returns cached
// TanStack Query data with a stable reference across renders, and
// the mock must mirror that.
const STABLE_SKILLS = [
  { skill_text: "react", display_text: "React", position: 0 },
  { skill_text: "typescript", display_text: "TypeScript", position: 1 },
]
const STABLE_PROFILE_SKILLS_RESULT = {
  data: STABLE_SKILLS,
  isLoading: false,
}

vi.mock("../../hooks/use-profile-skills", () => ({
  useProfileSkills: () => STABLE_PROFILE_SKILLS_RESULT,
}))

vi.mock("../../hooks/use-update-profile-skills", () => ({
  useUpdateProfileSkills: () => ({
    mutateAsync: (value: string[]) => mockMutateAsync(value),
    isPending: false,
  }),
}))

// Avoid hitting the real catalog + autocomplete endpoints.
vi.mock("../popular-skills-row", () => ({
  PopularSkillsRow: () => <div data-testid="popular-row" />,
}))
vi.mock("../expertise-panel", () => ({
  ExpertisePanel: ({ expertiseKey }: { expertiseKey: string }) => (
    <div data-testid={`panel-${expertiseKey}`} />
  ),
}))
vi.mock("../skill-search-bar", () => ({
  SkillSearchBar: ({
    onAdd,
  }: {
    onAdd: (s: {
      skill_text: string
      display_text: string
      expertise_keys: string[]
      is_curated: boolean
      usage_count: number
    }) => void
  }) => (
    <button
      data-testid="add-via-search"
      type="button"
      onClick={() =>
        onAdd({
          skill_text: "vue",
          display_text: "Vue.js",
          expertise_keys: ["development"],
          is_curated: true,
          usage_count: 1000,
        })
      }
    >
      add-vue
    </button>
  ),
}))

function renderModal(
  props: Partial<Parameters<typeof SkillsEditorModal>[0]> = {},
) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  const defaults = {
    open: true,
    onClose: vi.fn(),
    expertiseKeys: ["development"],
    maxSkills: 25,
  }
  const merged = { ...defaults, ...props }
  const utils = render(
    <QueryClientProvider client={queryClient}>
      <NextIntlClientProvider locale="en" messages={messages}>
        <SkillsEditorModal {...merged} />
      </NextIntlClientProvider>
    </QueryClientProvider>,
  )
  return { ...utils, props: merged }
}

beforeEach(() => {
  vi.clearAllMocks()
  mockMutateAsync.mockResolvedValue(undefined)
})

describe("SkillsEditorModal", () => {
  it("returns null when closed", () => {
    const { container } = renderModal({ open: false })
    expect(container).toBeEmptyDOMElement()
  })

  it("renders the modal title when open", () => {
    renderModal()
    expect(
      screen.getByRole("heading", {
        name: messages.profile.skills.modalTitle,
      }),
    ).toBeInTheDocument()
  })

  it("renders the persisted skills in the selected list", () => {
    renderModal()
    expect(screen.getByText("React")).toBeInTheDocument()
    expect(screen.getByText("TypeScript")).toBeInTheDocument()
  })

  it("adds a skill from the search bar and enables Save", async () => {
    const user = userEvent.setup()
    renderModal()

    await user.click(screen.getByTestId("add-via-search"))
    expect(screen.getByText("Vue.js")).toBeInTheDocument()

    const save = screen.getByRole("button", {
      name: new RegExp(messages.profile.skills.save, "i"),
    })
    expect(save).not.toBeDisabled()
  })

  it("removes a skill from the selected list", async () => {
    const user = userEvent.setup()
    renderModal()

    const removeButton = screen.getByRole("button", {
      name: messages.profile.skills.remove.replace("{label}", "React"),
    })
    await user.click(removeButton)

    expect(screen.queryByText("React")).toBeNull()
  })

  it("calls the mutation with the selected list on Save", async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    renderModal({ onClose })

    await user.click(screen.getByTestId("add-via-search"))
    await user.click(
      screen.getByRole("button", {
        name: new RegExp(messages.profile.skills.save, "i"),
      }),
    )

    await waitFor(() =>
      expect(mockMutateAsync).toHaveBeenCalledWith([
        "react",
        "typescript",
        "vue",
      ]),
    )
    await waitFor(() => expect(onClose).toHaveBeenCalled())
  })

  it("closes on the close button click", async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    renderModal({ onClose })

    await user.click(
      screen.getByRole("button", { name: messages.profile.skills.close }),
    )
    expect(onClose).toHaveBeenCalled()
  })
})
