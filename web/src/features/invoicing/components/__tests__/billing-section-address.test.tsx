import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { useEffect } from "react"
import { FormProvider, useForm } from "react-hook-form"
import { BillingSectionAddress } from "../billing-section-address"
import type { BillingProfileFormValues } from "../billing-profile-form.schema"

// Stub the AddressAutocomplete — we only assert that the parent
// passes the country prop and gets the onSelect callback wired.
vi.mock("../address-autocomplete", () => ({
  AddressAutocomplete: ({
    country,
    onSelect,
  }: {
    country: string
    onSelect: (addr: { line1: string; postalCode: string; city: string }) => void
  }) => (
    <div data-testid="autocomplete" data-country={country}>
      <button
        type="button"
        onClick={() =>
          onSelect({ line1: "10 rue X", postalCode: "75002", city: "Paris" })
        }
      >
        autocomplete-pick
      </button>
    </div>
  ),
}))

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
        <BillingSectionAddress />
      </form>
    </FormProvider>
  )
}

describe("BillingSectionAddress", () => {
  it("renders the country + address sections", () => {
    render(<Harness />)
    expect(
      screen.getByRole("heading", { name: "Pays" }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("heading", { name: "Adresse" }),
    ).toBeInTheDocument()
    expect(screen.getByLabelText("Pays de facturation")).toBeInTheDocument()
  })

  it("shows the country dropdown with grouped options", () => {
    render(<Harness initial={{ country: "FR" }} />)
    const select = screen.getByLabelText("Pays de facturation")
    expect(select).toHaveValue("FR")
  })

  it("changes the country when a different option is picked", () => {
    let formRef: ReturnType<typeof useForm<BillingProfileFormValues>> | null = null
    render(
      <Harness
        initial={{ country: "FR" }}
        expose={(f) => (formRef = f)}
      />,
    )
    fireEvent.change(screen.getByLabelText("Pays de facturation"), {
      target: { value: "BE" },
    })
    expect(formRef!.getValues("country")).toBe("BE")
  })

  it("renders the autocomplete with the current country prop", () => {
    render(<Harness initial={{ country: "FR" }} />)
    expect(screen.getByTestId("autocomplete")).toHaveAttribute(
      "data-country",
      "FR",
    )
  })

  it("autocomplete onSelect populates address fields", () => {
    let formRef: ReturnType<typeof useForm<BillingProfileFormValues>> | null = null
    render(
      <Harness
        initial={{ country: "FR" }}
        expose={(f) => (formRef = f)}
      />,
    )
    fireEvent.click(screen.getByText("autocomplete-pick"))
    expect(formRef!.getValues("address_line1")).toBe("10 rue X")
    expect(formRef!.getValues("postal_code")).toBe("75002")
    expect(formRef!.getValues("city")).toBe("Paris")
  })

  it("typing address_line1 updates form state", () => {
    let formRef: ReturnType<typeof useForm<BillingProfileFormValues>> | null = null
    render(<Harness expose={(f) => (formRef = f)} />)
    fireEvent.change(screen.getByLabelText("Adresse"), {
      target: { value: "1 avenue Foch" },
    })
    expect(formRef!.getValues("address_line1")).toBe("1 avenue Foch")
  })

  it("typing postal_code + city updates form state", () => {
    let formRef: ReturnType<typeof useForm<BillingProfileFormValues>> | null = null
    render(<Harness expose={(f) => (formRef = f)} />)
    fireEvent.change(screen.getByLabelText("Code postal"), {
      target: { value: "75008" },
    })
    fireEvent.change(screen.getByLabelText("Ville"), {
      target: { value: "Paris 8e" },
    })
    expect(formRef!.getValues("postal_code")).toBe("75008")
    expect(formRef!.getValues("city")).toBe("Paris 8e")
  })

  it("address_line2 is optional and updates state when filled", () => {
    let formRef: ReturnType<typeof useForm<BillingProfileFormValues>> | null = null
    render(<Harness expose={(f) => (formRef = f)} />)
    fireEvent.change(screen.getByLabelText(/Complément d'adresse/), {
      target: { value: "Bât B" },
    })
    expect(formRef!.getValues("address_line2")).toBe("Bât B")
  })

  it("renders the country select 'choose' default option when country is empty", () => {
    render(<Harness initial={{ country: "" }} />)
    expect(screen.getByText(/Sélectionne/)).toBeInTheDocument()
  })
})
