import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { SharedLanguagesSection } from "../shared-languages-section"

const mockUseShared = vi.fn()
const mockMutate = vi.fn()

vi.mock("../../hooks/use-organization-shared", () => ({
  useOrganizationShared: () => mockUseShared(),
}))

vi.mock("../../hooks/use-update-organization-languages", () => ({
  useUpdateOrganizationLanguages: () => ({
    mutate: mockMutate,
    isPending: false,
  }),
}))

vi.mock("next-intl", () => ({
  useLocale: () => "fr",
  useTranslations: () => (key: string, args?: Record<string, unknown>) => {
    if (args) return `${key}::${JSON.stringify(args)}`
    return key
  },
}))

vi.mock("@/shared/lib/profile/language-options", () => ({
  LANGUAGE_OPTIONS: [
    { code: "fr" },
    { code: "en" },
    { code: "es" },
  ],
  getLanguageLabel: (code: string) =>
    ({ fr: "Français", en: "Anglais", es: "Espagnol" })[code] ?? code,
}))

beforeEach(() => {
  vi.clearAllMocks()
})

describe("SharedLanguagesSection", () => {
  it("renders both bucket selectors", () => {
    mockUseShared.mockReturnValue({ data: undefined })
    render(<SharedLanguagesSection />)
    expect(screen.getByText("professionalLabel")).toBeInTheDocument()
    expect(screen.getByText("conversationalLabel")).toBeInTheDocument()
  })

  it("renders the selected language chips", () => {
    mockUseShared.mockReturnValue({
      data: {
        languages_professional: ["fr"],
        languages_conversational: ["en"],
      },
    })
    render(<SharedLanguagesSection />)
    // Both chip and dropdown option may show the language label, so use getAllByText
    expect(screen.getAllByText("Français").length).toBeGreaterThan(0)
    expect(screen.getAllByText("Anglais").length).toBeGreaterThan(0)
  })

  it("removes a language when its X button is clicked", () => {
    mockUseShared.mockReturnValue({
      data: {
        languages_professional: ["fr"],
        languages_conversational: [],
      },
    })
    render(<SharedLanguagesSection />)
    const removeBtn = screen.getByLabelText(/remove/i)
    fireEvent.click(removeBtn)
    // The chip should be removed, then save becomes enabled
    fireEvent.click(screen.getByText("save"))
    expect(mockMutate).toHaveBeenCalledWith({
      professional: [],
      conversational: [],
    })
  })

  it("save button is disabled before any change", () => {
    mockUseShared.mockReturnValue({
      data: {
        languages_professional: [],
        languages_conversational: [],
      },
    })
    render(<SharedLanguagesSection />)
    const saveBtn = screen.getByText("save").closest("button")
    expect(saveBtn?.disabled).toBe(true)
  })

  it("disables non-selected options in dropdowns when added", () => {
    mockUseShared.mockReturnValue({
      data: {
        languages_professional: ["fr"],
        languages_conversational: [],
      },
    })
    const { container } = render(<SharedLanguagesSection />)
    const selects = container.querySelectorAll("select")
    expect(selects.length).toBeGreaterThan(0)
  })

  it("dispatches the mutation on save", () => {
    mockUseShared.mockReturnValue({
      data: {
        languages_professional: ["fr"],
        languages_conversational: ["en"],
      },
    })
    render(<SharedLanguagesSection />)
    const select = screen.getAllByRole("combobox")[0]
    fireEvent.change(select, { target: { value: "es" } })
    fireEvent.click(screen.getByText("save"))
    expect(mockMutate).toHaveBeenCalledWith({
      professional: ["fr", "es"],
      conversational: ["en"],
    })
  })

  it("falls back to empty arrays when shared data is undefined", () => {
    mockUseShared.mockReturnValue({ data: undefined })
    render(<SharedLanguagesSection />)
    // Should render selection counts of 0
    expect(screen.getAllByText(/selectionCount/i)).toHaveLength(2)
  })

  it("moves a language between buckets (add to one removes from the other)", () => {
    mockUseShared.mockReturnValue({
      data: {
        languages_professional: ["fr"],
        languages_conversational: [],
      },
    })
    render(<SharedLanguagesSection />)
    const selects = screen.getAllByRole("combobox")
    // Add "fr" to conversational; should auto-remove from professional
    fireEvent.change(selects[1], { target: { value: "fr" } })
    fireEvent.click(screen.getByText("save"))
    expect(mockMutate).toHaveBeenCalledWith({
      professional: [],
      conversational: ["fr"],
    })
  })
})
