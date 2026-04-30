import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { useEffect } from "react"
import { FormProvider, useForm } from "react-hook-form"
import { BillingSectionFiscal } from "../billing-section-fiscal"
import type { BillingProfileFormValues } from "../billing-profile-form.schema"

function defaults(
  overrides: Partial<BillingProfileFormValues> = {},
): BillingProfileFormValues {
  return {
    profile_type: "individual",
    legal_name: "",
    trading_name: "",
    legal_form: "",
    tax_id: "",
    vat_number: "",
    address_line1: "",
    address_line2: "",
    postal_code: "",
    city: "",
    country: "",
    invoicing_email: "",
    ...overrides,
  }
}

function Harness({
  initial,
  validatedAt = null,
  isValidating = false,
  validateError = null,
  onValidate = vi.fn(),
  expose,
}: {
  initial?: Partial<BillingProfileFormValues>
  validatedAt?: string | null
  isValidating?: boolean
  validateError?: Error | null
  onValidate?: () => void
  expose?: (form: ReturnType<typeof useForm<BillingProfileFormValues>>) => void
}) {
  const form = useForm<BillingProfileFormValues>({
    defaultValues: defaults(initial),
  })
  useEffect(() => {
    expose?.(form)
  }, [expose, form])
  return (
    <FormProvider {...form}>
      <BillingSectionFiscal
        validatedAt={validatedAt}
        isValidating={isValidating}
        validateError={validateError}
        onValidate={onValidate}
      />
    </FormProvider>
  )
}

describe("BillingSectionFiscal", () => {
  it("renders nothing when country is non-EU", () => {
    const { container } = render(<Harness initial={{ country: "US" }} />)
    expect(container).toBeEmptyDOMElement()
  })

  it("renders the section + VAT input when country is EU non-FR", () => {
    render(<Harness initial={{ country: "BE" }} />)
    expect(
      screen.getByRole("heading", { name: "Identifiants fiscaux" }),
    ).toBeInTheDocument()
    expect(
      screen.getByLabelText(/Numéro de TVA intracommunautaire/),
    ).toBeInTheDocument()
  })

  it("renders for FR (FR is EU)", () => {
    render(<Harness initial={{ country: "FR" }} />)
    expect(
      screen.getByLabelText(/Numéro de TVA intracommunautaire/),
    ).toBeInTheDocument()
  })

  it("disables the validate button when VAT is empty", () => {
    render(<Harness initial={{ country: "BE", vat_number: "" }} />)
    expect(
      screen.getByRole("button", { name: /Valider mon n° TVA/ }),
    ).toBeDisabled()
  })

  it("enables the validate button when VAT has content", () => {
    render(<Harness initial={{ country: "BE", vat_number: "BE0123456789" }} />)
    expect(
      screen.getByRole("button", { name: /Valider mon n° TVA/ }),
    ).toBeEnabled()
  })

  it("calls onValidate when the validate button is clicked", () => {
    const onValidate = vi.fn()
    render(
      <Harness
        initial={{ country: "BE", vat_number: "BE0123456789" }}
        onValidate={onValidate}
      />,
    )
    fireEvent.click(
      screen.getByRole("button", { name: /Valider mon n° TVA/ }),
    )
    expect(onValidate).toHaveBeenCalledOnce()
  })

  it("disables the button while validating", () => {
    render(
      <Harness
        initial={{ country: "BE", vat_number: "BE0123456789" }}
        isValidating
      />,
    )
    expect(
      screen.getByRole("button", { name: /Valider mon n° TVA/ }),
    ).toBeDisabled()
  })

  it("shows the validated indicator when validatedAt is set", () => {
    render(
      <Harness
        initial={{ country: "BE", vat_number: "BE0123456789" }}
        validatedAt="2026-04-01T10:00:00Z"
      />,
    )
    expect(screen.getByText(/Validé le/)).toBeInTheDocument()
  })

  it("shows the error indicator when validateError is set", () => {
    render(
      <Harness
        initial={{ country: "BE", vat_number: "BE0123456789" }}
        validateError={new Error("nope")}
      />,
    )
    expect(screen.getByText(/Numéro non reconnu par VIES/)).toBeInTheDocument()
  })

  it("hides the validated indicator when there is also an error", () => {
    render(
      <Harness
        initial={{ country: "BE", vat_number: "BE0123456789" }}
        validatedAt="2026-04-01T10:00:00Z"
        validateError={new Error("nope")}
      />,
    )
    expect(screen.queryByText(/Validé le/)).not.toBeInTheDocument()
    expect(screen.getByText(/Numéro non reconnu par VIES/)).toBeInTheDocument()
  })

  it("typing the VAT number updates form state", () => {
    let formRef: ReturnType<typeof useForm<BillingProfileFormValues>> | null = null
    render(
      <Harness initial={{ country: "BE" }} expose={(f) => (formRef = f)} />,
    )
    fireEvent.change(
      screen.getByLabelText(/Numéro de TVA intracommunautaire/),
      { target: { value: "BE9999" } },
    )
    expect(formRef!.getValues("vat_number")).toBe("BE9999")
  })
})
