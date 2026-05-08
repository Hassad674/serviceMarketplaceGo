import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { PrestataireProfileHeader } from "../prestataire-profile-header"
import type { PrestataireProfileHeaderProps } from "../prestataire-profile-header"

function renderHeader(props: PrestataireProfileHeaderProps) {
  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <PrestataireProfileHeader {...props} />
    </NextIntlClientProvider>,
  )
}

function buildProps(
  overrides: Partial<PrestataireProfileHeaderProps["header"]> = {},
): PrestataireProfileHeaderProps {
  return {
    header: {
      kind: "freelance",
      identity: {
        photoUrl: "",
        displayName: "Ada Lovelace",
        title: "Senior Go engineer",
        availabilityStatus: "available_now",
        organizationId: "org-1",
        city: "Paris",
        countryCode: "FR",
        languagesProfessional: ["fr", "en"],
      },
      pricing: {
        value: {
          type: "daily",
          min_amount: 80000,
          max_amount: null,
          currency: "EUR",
        },
        fromLabel: "Starting at",
      },
      rating: { average: 4.6, count: 12 },
      ...overrides,
    },
  }
}

describe("PrestataireProfileHeader — shared shell", () => {
  it("renders the identity hero with the display name and italic title", () => {
    renderHeader(buildProps())
    expect(
      screen.getByRole("heading", { level: 1, name: "Ada Lovelace" }),
    ).toBeInTheDocument()
    expect(screen.getByText("Senior Go engineer")).toBeInTheDocument()
    expect(
      screen.getByTestId("availability-pill-available_now"),
    ).toBeInTheDocument()
  })

  it("renders the pricing rail with the formatted amount and label", () => {
    renderHeader(buildProps())
    expect(screen.getByText("Starting at")).toBeInTheDocument()
    // 80000 cents = €800/day formatted in EN locale.
    expect(screen.getByText(/€800/)).toBeInTheDocument()
  })

  it("hides the pricing rail when no pricing is provided", () => {
    renderHeader(
      buildProps({ pricing: { value: null, fromLabel: "Starting at" } }),
    )
    expect(screen.queryByText("Starting at")).not.toBeInTheDocument()
  })

  it("surfaces a badge when one is provided (referrer persona)", () => {
    renderHeader(
      buildProps({
        kind: "referrer",
        badge: { label: "Business referrer" },
      }),
    )
    expect(screen.getByText("Business referrer")).toBeInTheDocument()
  })

  it("renders the city · country meta row", () => {
    renderHeader(buildProps())
    expect(screen.getByText("Paris · FR")).toBeInTheDocument()
  })

  it("renders the rating block when reviews are present", () => {
    renderHeader(buildProps())
    expect(screen.getByText("4.6")).toBeInTheDocument()
    expect(screen.getByText(/12 reviews/)).toBeInTheDocument()
  })

  it("falls back to 'No reviews' when count is zero", () => {
    renderHeader(buildProps({ rating: { average: 0, count: 0 } }))
    expect(screen.getByText("No reviews")).toBeInTheDocument()
  })

  it("uses 'logo' wording in the alt text for the agency persona", () => {
    renderHeader(buildProps({ kind: "agency" }))
    // Portrait fallback gets the alt text via the shared imageAlt key.
    expect(
      screen.getByRole("img", { name: /Logo of Ada Lovelace/i }),
    ).toBeInTheDocument()
  })

  it("uses 'photo' wording for freelance and referrer", () => {
    renderHeader(buildProps({ kind: "freelance" }))
    expect(
      screen.getByRole("img", { name: /Photo of Ada Lovelace/i }),
    ).toBeInTheDocument()
  })

  it("opens the photo upload modal when the portrait is clicked in editable mode", async () => {
    const onUploadPhoto = vi.fn().mockResolvedValue(undefined)
    renderHeader(
      buildProps({
        editable: { onUploadPhoto, uploadingPhoto: false },
      }),
    )
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: /Edit your photo/i }))
    // The upload modal appears with the upload affordance.
    expect(
      screen.getByRole("dialog", { name: /Add a photo|Add photo/i }),
    ).toBeInTheDocument()
  })

  it("invokes onSaveTitle when the inline editor commits a new title", async () => {
    const onSaveTitle = vi.fn()
    renderHeader(buildProps({ editable: { onSaveTitle } }))
    const user = userEvent.setup()
    const button = screen.getByRole("button", {
      name: /Edit professional title/i,
    })
    await user.click(button)
    const input = screen.getByRole("textbox", {
      name: /Professional title/i,
    })
    await user.clear(input)
    await user.type(input, "Staff Go engineer")
    await user.keyboard("{Enter}")
    expect(onSaveTitle).toHaveBeenCalledWith("Staff Go engineer")
  })

  it("does not call onSaveTitle when the value is unchanged", async () => {
    const onSaveTitle = vi.fn()
    renderHeader(buildProps({ editable: { onSaveTitle } }))
    const user = userEvent.setup()
    await user.click(
      screen.getByRole("button", { name: /Edit professional title/i }),
    )
    await user.keyboard("{Enter}")
    expect(onSaveTitle).not.toHaveBeenCalled()
  })

  it("renders read-only title text when no editable handler is provided", () => {
    renderHeader(buildProps())
    expect(
      screen.queryByRole("button", { name: /Edit professional title/i }),
    ).not.toBeInTheDocument()
    expect(screen.getByText("Senior Go engineer")).toBeInTheDocument()
  })
})
