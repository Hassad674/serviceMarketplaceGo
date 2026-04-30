import { describe, expect, it, vi } from "vitest"
import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { ExpertiseEditor } from "../expertise-editor"

// The editor is a pure UI component now — `onSave` is required and
// supplied by the consumer. We exercise the whole picker / re-ordering
// / save / error pipeline through this single test surface.

const ALL_DOMAIN_KEYS = [
  "development",
  "data_ai_ml",
  "design_ui_ux",
  "design_3d_animation",
  "video_motion",
  "photo_audiovisual",
  "marketing_growth",
  "writing_translation",
  "business_dev_sales",
  "consulting_strategy",
  "product_ux_research",
  "ops_admin_support",
  "legal",
  "finance_accounting",
  "hr_recruitment",
] as const

interface RenderProps {
  domains?: string[]
  orgType?: string
  readOnly?: boolean
  onSave?: (next: string[]) => Promise<void>
  saving?: boolean
}

function renderEditor(props: RenderProps = {}) {
  const onSave = props.onSave ?? vi.fn().mockResolvedValue(undefined)
  const utils = render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <ExpertiseEditor
        domains={props.domains}
        orgType={props.orgType ?? "agency"}
        readOnly={props.readOnly}
        onSave={onSave}
        saving={props.saving ?? false}
      />
    </NextIntlClientProvider>,
  )
  return { ...utils, onSave }
}

describe("ExpertiseEditor", () => {
  // -------------------------------------------------------------------
  // Visibility / no-render guards
  // -------------------------------------------------------------------

  it("renders nothing when readOnly is true and the persisted list is empty", () => {
    const { container } = renderEditor({
      domains: [],
      readOnly: true,
    })
    expect(container).toBeEmptyDOMElement()
  })

  it("renders nothing when the org type does not support expertise", () => {
    const { container } = renderEditor({
      orgType: "enterprise",
      domains: ["development"],
    })
    expect(container).toBeEmptyDOMElement()
  })

  it("renders nothing when org type is undefined", () => {
    // Use a direct render here — `renderEditor` defaults orgType to
    // "agency" so the test must bypass the helper to assert the
    // explicit-undefined behaviour.
    const { container } = render(
      <NextIntlClientProvider locale="en" messages={messages}>
        <ExpertiseEditor
          domains={["development"]}
          orgType={undefined}
          onSave={vi.fn()}
          saving={false}
        />
      </NextIntlClientProvider>,
    )
    expect(container).toBeEmptyDOMElement()
  })

  // -------------------------------------------------------------------
  // Read-only display
  // -------------------------------------------------------------------

  it("renders a read-only pill list when readOnly is true and there are persisted items", () => {
    renderEditor({
      readOnly: true,
      domains: ["development", "design_ui_ux"],
    })
    expect(screen.getByText("Development")).toBeInTheDocument()
    expect(screen.getByText("Design & UI/UX")).toBeInTheDocument()
    // No Save / Cancel buttons in read-only mode.
    expect(screen.queryByRole("button", { name: /save/i })).toBeNull()
  })

  // -------------------------------------------------------------------
  // Editable picker — selection
  // -------------------------------------------------------------------

  it("renders the section title and subtitle in editable mode", () => {
    renderEditor({})
    expect(screen.getByText("Areas of expertise")).toBeInTheDocument()
    expect(
      screen.getByText(/Pick up to 8 domains/i),
    ).toBeInTheDocument()
  })

  it("renders all 15 picker buttons available for selection", () => {
    renderEditor({})
    const grid = screen.getByRole("group", {
      name: /expertise domain picker/i,
    })
    // Each domain should appear as an aria-pressed button in the picker.
    for (const key of ALL_DOMAIN_KEYS) {
      const label = messages.profile.expertise.domains[
        key as keyof typeof messages.profile.expertise.domains
      ]
      const buttons = screen.getAllByRole("button", { name: label })
      // At least one button (the picker tile) must exist.
      expect(buttons.length).toBeGreaterThan(0)
    }
    expect(grid).toBeInTheDocument()
  })

  it("toggles a domain into the selected list when the picker button is clicked", () => {
    renderEditor({})
    const buttons = screen.getAllByRole("button", { name: "Development" })
    // The picker button has aria-pressed=false initially.
    const pickerButton = buttons.find(
      (b) => b.getAttribute("aria-pressed") === "false",
    )!
    fireEvent.click(pickerButton)

    // After click, the same picker button is aria-pressed=true.
    expect(pickerButton.getAttribute("aria-pressed")).toBe("true")
    // And the counter advances to 1/8.
    expect(screen.getByText("1/8 selected")).toBeInTheDocument()
  })

  it("toggling a domain that is already selected removes it", () => {
    renderEditor({ domains: ["development"] })
    const pickerButtons = screen
      .getAllByRole("button", { name: "Development" })
      .filter((b) => b.hasAttribute("aria-pressed"))
    const pickerButton = pickerButtons[0]
    expect(pickerButton.getAttribute("aria-pressed")).toBe("true")
    fireEvent.click(pickerButton)
    // Counter goes back down to 0/8.
    expect(screen.getByText("0/8 selected")).toBeInTheDocument()
  })

  it("disables additional picker buttons once max is reached", () => {
    // provider_personal has a max of 5
    renderEditor({
      orgType: "provider_personal",
      domains: ["development", "data_ai_ml", "design_ui_ux", "video_motion", "marketing_growth"],
    })
    expect(screen.getByText("You've reached the maximum of 5 domains.")).toBeInTheDocument()
    // A non-selected button should be disabled.
    const buttons = screen.getAllByRole("button", { name: "Legal" })
    const pickerButton = buttons.find((b) => b.hasAttribute("aria-pressed"))!
    expect(pickerButton).toBeDisabled()
  })

  // -------------------------------------------------------------------
  // Re-ordering
  // -------------------------------------------------------------------

  it("moves an item down when the move-down arrow is clicked", () => {
    renderEditor({ domains: ["development", "design_ui_ux"] })
    const moveDown = screen.getByRole("button", {
      name: /Move Development down/i,
    })
    fireEvent.click(moveDown)
    // After move, the second slot in the ordered list should now be Development.
    const list = screen.getByRole("list", {
      name: /Selected expertise domains, ordered/i,
    })
    expect(list).toHaveTextContent("1.Design & UI/UX")
    expect(list).toHaveTextContent("2.Development")
  })

  it("moves an item up when the move-up arrow is clicked", () => {
    renderEditor({ domains: ["design_ui_ux", "development"] })
    const moveUp = screen.getByRole("button", {
      name: /Move Development up/i,
    })
    fireEvent.click(moveUp)
    const list = screen.getByRole("list", {
      name: /Selected expertise domains, ordered/i,
    })
    expect(list).toHaveTextContent("1.Development")
  })

  it("removing an item from the selected list via the X button", () => {
    renderEditor({ domains: ["development"] })
    const removeBtn = screen.getByRole("button", {
      name: /Remove Development/i,
    })
    fireEvent.click(removeBtn)
    expect(screen.getByText("0/8 selected")).toBeInTheDocument()
  })

  // -------------------------------------------------------------------
  // Save / Reset / Saving state
  // -------------------------------------------------------------------

  it("Save button is disabled until the draft differs from the persisted list", () => {
    renderEditor({ domains: ["development"] })
    const saveBtn = screen.getByRole("button", { name: /^Save$/i })
    expect(saveBtn).toBeDisabled()
  })

  it("Save button enables once the draft diverges, and calls onSave with the full draft on click", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined)
    renderEditor({ domains: [], onSave })
    // Pick Development from the picker.
    const pickerButton = screen
      .getAllByRole("button", { name: "Development" })
      .find((b) => b.hasAttribute("aria-pressed"))!
    fireEvent.click(pickerButton)

    const saveBtn = screen.getByRole("button", { name: /^Save$/i })
    expect(saveBtn).not.toBeDisabled()
    fireEvent.click(saveBtn)

    await waitFor(() => {
      expect(onSave).toHaveBeenCalledOnce()
    })
    expect(onSave).toHaveBeenCalledWith(["development"])
  })

  it("Cancel button reverts the draft to the persisted list", () => {
    renderEditor({ domains: ["development"] })
    // Add design_ui_ux on top.
    const designPicker = screen
      .getAllByRole("button", { name: "Design & UI/UX" })
      .find((b) => b.hasAttribute("aria-pressed"))!
    fireEvent.click(designPicker)
    expect(screen.getByText("2/8 selected")).toBeInTheDocument()

    fireEvent.click(screen.getByRole("button", { name: /^Cancel$/i }))
    expect(screen.getByText("1/8 selected")).toBeInTheDocument()
  })

  it("displays a saving state and disables save while the mutation is pending", () => {
    renderEditor({
      domains: [],
      saving: true,
    })
    // Pick a domain so there is a draft change.
    const pickerButton = screen
      .getAllByRole("button", { name: "Development" })
      .find((b) => b.hasAttribute("aria-pressed"))!
    fireEvent.click(pickerButton)

    expect(screen.getByText("Saving...")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /Saving\.\.\./ })).toBeDisabled()
  })

  // -------------------------------------------------------------------
  // Error mapping
  // -------------------------------------------------------------------

  it("maps a 403 response to the forbidden error message", async () => {
    const err = new Error("forbidden") as Error & { status?: number }
    err.status = 403
    const onSave = vi.fn().mockRejectedValue(err)

    renderEditor({ domains: [], onSave })
    const pickerButton = screen
      .getAllByRole("button", { name: "Development" })
      .find((b) => b.hasAttribute("aria-pressed"))!
    fireEvent.click(pickerButton)
    fireEvent.click(screen.getByRole("button", { name: /^Save$/i }))

    expect(
      await screen.findByRole("alert", undefined, { timeout: 1000 }),
    ).toHaveTextContent(/permission/i)
  })

  it("maps a 400 response to the validation error message", async () => {
    const err = new Error("bad") as Error & { status?: number }
    err.status = 400
    const onSave = vi.fn().mockRejectedValue(err)

    renderEditor({ domains: [], onSave })
    const pickerButton = screen
      .getAllByRole("button", { name: "Development" })
      .find((b) => b.hasAttribute("aria-pressed"))!
    fireEvent.click(pickerButton)
    fireEvent.click(screen.getByRole("button", { name: /^Save$/i }))

    expect(
      await screen.findByRole("alert", undefined, { timeout: 1000 }),
    ).toHaveTextContent(/not valid|exceed the maximum/i)
  })

  it("falls back to a generic message for any other failure", async () => {
    const onSave = vi.fn().mockRejectedValue(new Error("boom"))

    renderEditor({ domains: [], onSave })
    const pickerButton = screen
      .getAllByRole("button", { name: "Development" })
      .find((b) => b.hasAttribute("aria-pressed"))!
    fireEvent.click(pickerButton)
    fireEvent.click(screen.getByRole("button", { name: /^Save$/i }))

    expect(
      await screen.findByRole("alert", undefined, { timeout: 1000 }),
    ).toHaveTextContent(/Could not save/i)
  })

  // -------------------------------------------------------------------
  // Re-sync after server refetch
  // -------------------------------------------------------------------

  it("re-syncs the local draft when the persisted list changes externally", () => {
    const { rerender } = render(
      <NextIntlClientProvider locale="en" messages={messages}>
        <ExpertiseEditor
          domains={["development"]}
          orgType="agency"
          onSave={vi.fn()}
          saving={false}
        />
      </NextIntlClientProvider>,
    )
    expect(screen.getByText("1/8 selected")).toBeInTheDocument()
    rerender(
      <NextIntlClientProvider locale="en" messages={messages}>
        <ExpertiseEditor
          domains={["development", "design_ui_ux"]}
          orgType="agency"
          onSave={vi.fn()}
          saving={false}
        />
      </NextIntlClientProvider>,
    )
    expect(screen.getByText("2/8 selected")).toBeInTheDocument()
  })

  it("filters out unknown domain strings coming from the backend", () => {
    renderEditor({ domains: ["development", "made_up_key"] })
    // The unknown key should be ignored — counter shows 1, not 2.
    expect(screen.getByText("1/8 selected")).toBeInTheDocument()
  })
})
