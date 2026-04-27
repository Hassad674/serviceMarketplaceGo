import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import type { ReactNode } from "react"

import { OnboardingWizard } from "../onboarding-wizard"

// next-intl: pass the key through so tests can match on it.
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

// Heavy sub-components — replace with thin stubs.
vi.mock("../country-selector", () => ({
  CountrySelector: ({
    onChange,
  }: {
    value: string | null
    onChange: (code: string) => void
    disabled?: boolean
  }) => (
    <button data-testid="country-stub" onClick={() => onChange("FR")}>
      pick-fr
    </button>
  ),
}))

vi.mock("../trust-signals", () => ({
  TrustSignals: () => <div data-testid="trust-signals" />,
}))

// Stripe Elements — captures the onChange handler so the test can fire
// a synthetic StripeAddressElementChangeEvent payload.
const addressOnChangeRef: { current: ((event: unknown) => void) | null } = {
  current: null,
}

vi.mock("@stripe/react-stripe-js", () => ({
  Elements: ({ children }: { children: ReactNode }) => (
    <div data-testid="elements-provider">{children}</div>
  ),
  AddressElement: ({
    onChange,
  }: {
    options: unknown
    onChange: (event: unknown) => void
  }) => {
    addressOnChangeRef.current = onChange
    return <div data-testid="address-element" />
  },
}))

vi.mock("@/shared/lib/stripe-client", () => ({
  stripePromise: Promise.resolve(null),
}))

describe("OnboardingWizard", () => {
  it("does not render the AddressElement until a country has been picked", () => {
    render(<OnboardingWizard loading={false} onSubmit={vi.fn()} />)
    expect(screen.queryByTestId("address-element")).toBeNull()
  })

  it("renders the AddressElement inside an Elements provider once a country is selected", () => {
    render(<OnboardingWizard loading={false} onSubmit={vi.fn()} />)

    fireEvent.click(screen.getByTestId("country-stub"))

    expect(screen.getByTestId("elements-provider")).toBeDefined()
    expect(screen.getByTestId("address-element")).toBeDefined()
  })

  it("forwards AddressElement onChange values to the parent via onAddressChange", () => {
    const onAddressChange = vi.fn()
    render(
      <OnboardingWizard
        loading={false}
        onSubmit={vi.fn()}
        onAddressChange={onAddressChange}
      />,
    )

    fireEvent.click(screen.getByTestId("country-stub"))

    // Synthetic StripeAddressElementChangeEvent — we only care about
    // event.value.address. The other top-level fields are typed but
    // never read by the wizard.
    addressOnChangeRef.current?.({
      elementType: "address",
      elementMode: "billing",
      empty: false,
      complete: true,
      isNewAddress: true,
      value: {
        name: "Acme",
        address: {
          line1: "10 Rue de la Paix",
          line2: null,
          city: "Paris",
          state: "",
          postal_code: "75002",
          country: "FR",
        },
      },
    })

    expect(onAddressChange).toHaveBeenCalledWith({
      line1: "10 Rue de la Paix",
      line2: "",
      city: "Paris",
      postal_code: "75002",
      country: "FR",
    })
  })
})
