import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { useEffect } from "react"
import { FormProvider, useForm } from "react-hook-form"
import { BillingSectionLegalIdentity } from "../billing-section-legal-identity"
import type { BillingProfileFormValues } from "../billing-profile-form.schema"

vi.mock("@stripe/react-stripe-js", () => ({}))

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
  expose,
}: {
  initial?: Partial<BillingProfileFormValues>
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
      <form>
        <BillingSectionLegalIdentity />
      </form>
    </FormProvider>
  )
}

describe("BillingSectionLegalIdentity", () => {
  it("renders the section headings (profile + identity + fiscal IDs)", () => {
    render(<Harness />)
    expect(
      screen.getByRole("heading", { name: "Type de profil" }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("heading", { name: "Identité légale" }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("heading", { name: "Identifiants fiscaux" }),
    ).toBeInTheDocument()
  })

  it("renders the legal_name input with the current form value", () => {
    render(<Harness initial={{ legal_name: "Acme SAS" }} />)
    expect(screen.getByLabelText(/Raison sociale/i)).toHaveValue("Acme SAS")
  })

  it("hides trading_name + legal_form when profile_type is individual", () => {
    render(<Harness initial={{ profile_type: "individual" }} />)
    expect(screen.queryByLabelText(/Nom commercial/)).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/Forme juridique/)).not.toBeInTheDocument()
  })

  it("shows trading_name + legal_form when profile_type is business", () => {
    render(<Harness initial={{ profile_type: "business" }} />)
    expect(screen.getByLabelText(/Nom commercial/)).toBeInTheDocument()
    expect(screen.getByLabelText(/Forme juridique/)).toBeInTheDocument()
  })

  it("shows the SIRET input with hint when country is FR", () => {
    render(<Harness initial={{ country: "FR" }} />)
    expect(screen.getByLabelText(/Numéro SIRET/)).toBeInTheDocument()
    expect(
      screen.getByText("14 chiffres, sans espace"),
    ).toBeInTheDocument()
  })

  it("shows the generic 'Identifiant fiscal' input when country is non-FR", () => {
    render(<Harness initial={{ country: "BE" }} />)
    expect(screen.getByLabelText("Identifiant fiscal")).toBeInTheDocument()
    expect(screen.queryByLabelText(/Numéro SIRET/)).not.toBeInTheDocument()
  })

  it("typing legal_name updates the form state", () => {
    let formRef: ReturnType<typeof useForm<BillingProfileFormValues>> | null = null
    render(<Harness expose={(f) => (formRef = f)} />)
    fireEvent.change(screen.getByLabelText(/Raison sociale/i), {
      target: { value: "New Co" },
    })
    expect(formRef!.getValues("legal_name")).toBe("New Co")
  })

  it("changing the profile_type radio updates form state", () => {
    let formRef: ReturnType<typeof useForm<BillingProfileFormValues>> | null = null
    render(
      <Harness
        initial={{ profile_type: "individual" }}
        expose={(f) => (formRef = f)}
      />,
    )
    // Click the "Entreprise" radio
    const radios = screen.getAllByRole("radio")
    fireEvent.click(radios[1])
    expect(formRef!.getValues("profile_type")).toBe("business")
  })

  it("typing into trading_name updates form state (business only)", () => {
    let formRef: ReturnType<typeof useForm<BillingProfileFormValues>> | null = null
    render(
      <Harness
        initial={{ profile_type: "business" }}
        expose={(f) => (formRef = f)}
      />,
    )
    fireEvent.change(screen.getByLabelText(/Nom commercial/), {
      target: { value: "Acme TM" },
    })
    expect(formRef!.getValues("trading_name")).toBe("Acme TM")
  })
})
