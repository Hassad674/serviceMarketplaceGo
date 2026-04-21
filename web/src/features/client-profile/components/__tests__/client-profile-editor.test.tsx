import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/en.json"
import { ClientProfileEditor } from "../client-profile-editor"

function renderEditor(
  props: Partial<Parameters<typeof ClientProfileEditor>[0]> = {},
) {
  const defaultProps = {
    initialValues: {
      company_name: "Acme Corp",
      client_description: "",
    },
    onSubmit: vi.fn().mockResolvedValue(undefined),
    saving: false,
    submitError: null,
    ...props,
  }

  return render(
    <NextIntlClientProvider locale="en" messages={messages}>
      <ClientProfileEditor {...defaultProps} />
    </NextIntlClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("ClientProfileEditor", () => {
  it("renders with initial values populated", () => {
    renderEditor({
      initialValues: {
        company_name: "Acme Corp",
        client_description: "We run B2B campaigns.",
      },
    })

    expect(
      screen.getByRole("textbox", {
        name: messages.clientProfile.companyName,
      }),
    ).toHaveValue("Acme Corp")
    expect(
      screen.getByRole("textbox", {
        name: messages.clientProfile.description,
      }),
    ).toHaveValue("We run B2B campaigns.")
  })

  it("shows a character counter that updates as the user types", async () => {
    const user = userEvent.setup()
    renderEditor()

    const textarea = screen.getByRole("textbox", {
      name: messages.clientProfile.description,
    })
    await user.type(textarea, "Hello")
    expect(screen.getByText(/5 \/ 2000/)).toBeInTheDocument()
  })

  it("disables the submit button until the form is dirty", () => {
    renderEditor()
    const submit = screen.getByRole("button", {
      name: messages.clientProfile.saveChanges,
    })
    expect(submit).toBeDisabled()
  })

  it("calls onSubmit with the trimmed payload on happy path", async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined)
    const user = userEvent.setup()
    renderEditor({ onSubmit })

    const textarea = screen.getByRole("textbox", {
      name: messages.clientProfile.description,
    })
    await user.type(textarea, "Leading B2B studio")

    const submit = screen.getByRole("button", {
      name: messages.clientProfile.saveChanges,
    })
    await user.click(submit)

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1))
    expect(onSubmit).toHaveBeenCalledWith({
      company_name: "Acme Corp",
      client_description: "Leading B2B studio",
    })
  })

  it("surfaces validation error when company name is blanked out", async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined)
    const user = userEvent.setup()
    renderEditor({ onSubmit })

    const nameInput = screen.getByRole("textbox", {
      name: messages.clientProfile.companyName,
    })
    await user.clear(nameInput)

    const submit = screen.getByRole("button", {
      name: messages.clientProfile.saveChanges,
    })
    await user.click(submit)

    await waitFor(() => {
      expect(nameInput).toHaveAttribute("aria-invalid", "true")
    })
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it("renders the submit error when provided", () => {
    renderEditor({
      submitError: messages.clientProfile.saveError,
    })
    expect(
      screen.getByRole("alert", { name: "" })
        .textContent,
    ).toContain(messages.clientProfile.saveError)
  })

  it("shows the saving indicator while the mutation is in flight", () => {
    renderEditor({ saving: true })
    expect(
      screen.getByRole("button", { name: messages.clientProfile.saving }),
    ).toBeDisabled()
  })
})
