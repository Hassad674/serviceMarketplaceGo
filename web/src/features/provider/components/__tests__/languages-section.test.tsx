import { describe, expect, it, vi, beforeEach } from "vitest"
import { render, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import messages from "@/../messages/en.json"
import { LanguagesSection } from "../languages-section"
import type { Profile } from "../../api/profile-api"

const baseProfile: Profile = {
  organization_id: "org-1",
  title: "",
  photo_url: "",
  presentation_video_url: "",
  referrer_video_url: "",
  about: "",
  referrer_about: "",
  languages_professional: [],
  languages_conversational: [],
  created_at: "2026-04-01",
  updated_at: "2026-04-01",
}

let profileOverride: Partial<Profile> = {}
const mockMutate = vi.fn()

vi.mock("../../hooks/use-profile", () => ({
  useProfile: () => ({ data: { ...baseProfile, ...profileOverride } }),
  profileQueryKey: () => ["user", "x", "profile"],
}))

vi.mock("../../hooks/use-update-languages", () => ({
  useUpdateLanguages: () => ({
    mutate: (value: unknown) => mockMutate(value),
    isPending: false,
  }),
}))

function renderSection(
  props: Partial<Parameters<typeof LanguagesSection>[0]> = {},
) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  const defaults = {
    orgType: "provider_personal",
    readOnly: false,
  }
  return render(
    <QueryClientProvider client={queryClient}>
      <NextIntlClientProvider locale="en" messages={messages}>
        <LanguagesSection {...defaults} {...props} />
      </NextIntlClientProvider>
    </QueryClientProvider>,
  )
}

function getProBucket(): HTMLElement {
  const heading = screen.getByRole("heading", {
    name: messages.profile.languages.professionalLabel,
  })
  const group = heading.closest('[role="group"]')
  if (!group) throw new Error("professional group not found")
  return group as HTMLElement
}

function getConvBucket(): HTMLElement {
  const heading = screen.getByRole("heading", {
    name: messages.profile.languages.conversationalLabel,
  })
  const group = heading.closest('[role="group"]')
  if (!group) throw new Error("conversational group not found")
  return group as HTMLElement
}

beforeEach(() => {
  vi.clearAllMocks()
  profileOverride = {}
})

describe("LanguagesSection", () => {
  it("renders nothing for enterprise", () => {
    const { container } = renderSection({ orgType: "enterprise" })
    expect(container).toBeEmptyDOMElement()
  })

  it("renders both professional and conversational headings", () => {
    renderSection()
    expect(
      screen.getByRole("heading", {
        name: messages.profile.languages.professionalLabel,
      }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("heading", {
        name: messages.profile.languages.conversationalLabel,
      }),
    ).toBeInTheDocument()
  })

  it("persists chips from the profile into both buckets", () => {
    profileOverride = {
      languages_professional: ["en", "fr"],
      languages_conversational: ["es"],
    }
    renderSection()
    expect(
      within(getProBucket()).getByText("English"),
    ).toBeInTheDocument()
    expect(within(getProBucket()).getByText("French")).toBeInTheDocument()
    expect(within(getConvBucket()).getByText("Spanish")).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: "Remove English" }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: "Remove Spanish" }),
    ).toBeInTheDocument()
  })

  it("filters the combobox listbox by query and picks on Enter", async () => {
    const user = userEvent.setup()
    renderSection()
    const comboboxes = screen.getAllByRole("combobox")
    const proCombobox = comboboxes[0]
    await user.click(proCombobox)
    await user.type(proCombobox, "Eng")
    const listbox = await screen.findAllByRole("listbox")
    const options = within(listbox[0]).getAllByRole("option")
    // "English" should be the single match for "Eng"
    expect(options).toHaveLength(1)
    expect(options[0]).toHaveTextContent("English")
    await user.keyboard("{Enter}")
    // Chip appears in the pro bucket
    expect(
      within(getProBucket()).getByText("English"),
    ).toBeInTheDocument()
  })

  it("shows the empty state when no language matches", async () => {
    const user = userEvent.setup()
    renderSection()
    const comboboxes = screen.getAllByRole("combobox")
    await user.click(comboboxes[0])
    await user.type(comboboxes[0], "zzzzz")
    expect(
      await screen.findByText(messages.profile.languages.noResults),
    ).toBeInTheDocument()
  })

  it("announces added and removed languages via aria-live", async () => {
    profileOverride = { languages_professional: ["en"] }
    const user = userEvent.setup()
    renderSection()
    const status = screen.getByRole("status")
    const removeBtn = screen.getByRole("button", { name: "Remove English" })
    await user.click(removeBtn)
    expect(status).toHaveTextContent("English removed")
  })

  it("clears the bucket with the Clear all button", async () => {
    profileOverride = { languages_professional: ["en", "fr"] }
    const user = userEvent.setup()
    renderSection()
    const proBucket = getProBucket()
    const clearBtn = within(proBucket).getByRole("button", {
      name: messages.profile.languages.clearAll,
    })
    await user.click(clearBtn)
    expect(within(proBucket).queryByText("English")).not.toBeInTheDocument()
    expect(within(proBucket).queryByText("French")).not.toBeInTheDocument()
  })

  it("calls the mutation with the selected lists on save", async () => {
    const user = userEvent.setup()
    renderSection()
    const proCombobox = screen.getAllByRole("combobox")[0]
    await user.click(proCombobox)
    await user.type(proCombobox, "English")
    await user.keyboard("{Enter}")
    const saveBtn = screen.getByRole("button", {
      name: new RegExp(messages.profile.languages.save, "i"),
    })
    expect(saveBtn).not.toBeDisabled()
    await user.click(saveBtn)
    expect(mockMutate).toHaveBeenCalledWith({
      professional: ["en"],
      conversational: [],
    })
  })
})
