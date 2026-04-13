import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import messages from "@/../messages/en.json"
import { SkillSearchBar } from "../skill-search-bar"

const mockAutocomplete = vi.fn()
const mockCreate = vi.fn()

vi.mock("../../hooks/use-skill-autocomplete", () => ({
  useSkillAutocomplete: (query: string) => mockAutocomplete(query),
}))

vi.mock("../../hooks/use-create-user-skill", () => ({
  useCreateUserSkill: () => ({
    mutateAsync: (text: string) => mockCreate(text),
    isPending: false,
  }),
}))

function renderBar(props: Partial<Parameters<typeof SkillSearchBar>[0]> = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const defaults = {
    alreadySelected: new Set<string>(),
    onAdd: vi.fn(),
    disabled: false,
  }
  const merged = { ...defaults, ...props }
  const utils = render(
    <QueryClientProvider client={queryClient}>
      <NextIntlClientProvider locale="en" messages={messages}>
        <SkillSearchBar {...merged} />
      </NextIntlClientProvider>
    </QueryClientProvider>,
  )
  return { ...utils, props: merged }
}

beforeEach(() => {
  vi.clearAllMocks()
  mockAutocomplete.mockReturnValue({
    data: [
      {
        skill_text: "react",
        display_text: "React",
        expertise_keys: ["development"],
        is_curated: true,
        usage_count: 17000,
      },
      {
        skill_text: "react-native",
        display_text: "React Native",
        expertise_keys: ["development"],
        is_curated: true,
        usage_count: 5000,
      },
    ],
    isFetching: false,
  })
  mockCreate.mockResolvedValue({
    skill_text: "foo-bar",
    display_text: "Foo Bar",
    expertise_keys: [],
    is_curated: false,
    usage_count: 1,
  })
})

describe("SkillSearchBar", () => {
  it("renders the autocomplete input with placeholder", () => {
    renderBar()
    expect(
      screen.getByPlaceholderText(messages.profile.skills.searchPlaceholder),
    ).toBeInTheDocument()
  })

  it("shows suggestions from the autocomplete hook when typing", async () => {
    const user = userEvent.setup()
    renderBar()
    await user.type(
      screen.getByPlaceholderText(messages.profile.skills.searchPlaceholder),
      "re",
    )
    expect(await screen.findByText("React")).toBeInTheDocument()
    expect(screen.getByText("React Native")).toBeInTheDocument()
  })

  it("adds a skill when clicking a suggestion", async () => {
    const user = userEvent.setup()
    const onAdd = vi.fn()
    renderBar({ onAdd })

    await user.type(
      screen.getByPlaceholderText(messages.profile.skills.searchPlaceholder),
      "re",
    )
    await user.click(await screen.findByText("React"))

    expect(onAdd).toHaveBeenCalledWith(
      expect.objectContaining({ skill_text: "react" }),
    )
  })

  it("filters out already-selected skills from the suggestions", async () => {
    const user = userEvent.setup()
    renderBar({ alreadySelected: new Set(["react"]) })

    await user.type(
      screen.getByPlaceholderText(messages.profile.skills.searchPlaceholder),
      "re",
    )
    await waitFor(() =>
      expect(screen.getByText("React Native")).toBeInTheDocument(),
    )
    expect(screen.queryByText("React")).toBeNull()
  })

  it("navigates with ArrowDown and selects with Enter", async () => {
    const user = userEvent.setup()
    const onAdd = vi.fn()
    renderBar({ onAdd })

    const input = screen.getByPlaceholderText(
      messages.profile.skills.searchPlaceholder,
    )
    await user.type(input, "re")
    await user.keyboard("{ArrowDown}")
    await user.keyboard("{Enter}")

    expect(onAdd).toHaveBeenCalledWith(
      expect.objectContaining({ skill_text: "react-native" }),
    )
  })

  it("shows a create row when no exact match is found", async () => {
    const user = userEvent.setup()
    mockAutocomplete.mockReturnValue({ data: [], isFetching: false })

    renderBar()
    await user.type(
      screen.getByPlaceholderText(messages.profile.skills.searchPlaceholder),
      "foo bar",
    )

    expect(
      await screen.findByText(/create "foo bar"/i),
    ).toBeInTheDocument()
  })
})
