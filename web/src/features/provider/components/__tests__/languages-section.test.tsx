import { describe, expect, it, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
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

beforeEach(() => {
  vi.clearAllMocks()
  profileOverride = {}
})

describe("LanguagesSection", () => {
  it("renders nothing for enterprise", () => {
    const { container } = renderSection({ orgType: "enterprise" })
    expect(container).toBeEmptyDOMElement()
  })

  it("renders both professional and conversational labels", () => {
    renderSection()
    expect(
      screen.getByText(messages.profile.languages.professionalLabel),
    ).toBeInTheDocument()
    expect(
      screen.getByText(messages.profile.languages.conversationalLabel),
    ).toBeInTheDocument()
  })

  it("persists values from the profile into both buckets", () => {
    profileOverride = {
      languages_professional: ["en", "fr"],
      languages_conversational: ["es"],
    }
    renderSection()
    // The language name appears both inside a chip and inside the
    // "add" <select>; we only need to confirm at least one match.
    expect(screen.getAllByText("English").length).toBeGreaterThan(0)
    expect(screen.getAllByText("French").length).toBeGreaterThan(0)
    expect(screen.getAllByText("Spanish").length).toBeGreaterThan(0)
    expect(
      screen.getByRole("button", { name: "Remove English" }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: "Remove Spanish" }),
    ).toBeInTheDocument()
  })

  it("calls the mutation with the selected lists on save", async () => {
    const user = userEvent.setup()
    renderSection()

    const proSelect = screen.getByLabelText(
      messages.profile.languages.professionalLabel,
    )
    await user.selectOptions(proSelect, "en")

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
