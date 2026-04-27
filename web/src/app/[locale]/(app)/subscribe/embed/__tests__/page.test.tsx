import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactElement } from "react"
import SubscribeEmbedPage from "../page"

// ---- Mocks ----

const mockSearchParams = new Map<string, string>()
vi.mock("next/navigation", () => ({
  useSearchParams: () => ({
    get: (key: string) => mockSearchParams.get(key) ?? null,
    forEach: (cb: (value: string, key: string) => void) => {
      mockSearchParams.forEach((value, key) => cb(value, key))
    },
  }),
}))

const routerReplaceMock = vi.fn((url: string) => {
  // Mirror the real router: rewriting `?cycle=...` updates the
  // searchParams the page reads on the next render. Tests assert
  // the URL passed AND that the page re-fires `subscribe()` with
  // the new cycle, so we must keep the in-memory params in sync.
  const queryIndex = url.indexOf("?")
  if (queryIndex < 0) return
  const next = new URLSearchParams(url.slice(queryIndex + 1))
  mockSearchParams.clear()
  next.forEach((value, key) => mockSearchParams.set(key, value))
})
vi.mock("@i18n/navigation", () => ({
  useRouter: () => ({ replace: routerReplaceMock, push: vi.fn() }),
}))

// Stripe React provider mock — the real one tries to load Stripe.js,
// which we don't want in unit tests. Render a sentinel div so we can
// assert that the embedded provider was mounted with the expected
// clientSecret.
vi.mock("@stripe/react-stripe-js", () => ({
  EmbeddedCheckoutProvider: ({
    children,
    options,
  }: {
    children: React.ReactNode
    options: { clientSecret: string }
  }) => (
    <div data-testid="embedded-checkout-provider" data-client-secret={options.clientSecret}>
      {children}
    </div>
  ),
  EmbeddedCheckout: () => <div data-testid="embedded-checkout" />,
}))

// stripe-client.ts is a module-level promise; we just stub it.
vi.mock("@/shared/lib/stripe-client", () => ({
  stripePromise: Promise.resolve(null),
}))

// BillingProfileForm is heavy — render a simple stub that calls
// onSaved when its "Save" button is clicked. The real form is unit
// tested elsewhere.
vi.mock("@/features/invoicing/components/billing-profile-form", () => ({
  BillingProfileForm: ({ onSaved }: { variant?: string; onSaved?: () => void }) => (
    <div data-testid="billing-profile-form">
      <button type="button" onClick={() => onSaved?.()}>save-billing</button>
    </div>
  ),
}))

const billingProfileSnapshot = {
  profile: {
    organization_id: "org_1",
    profile_type: "business" as const,
    legal_name: "Acme",
    trading_name: "",
    legal_form: "",
    tax_id: "",
    vat_number: "",
    vat_validated_at: null,
    address_line1: "1 rue",
    address_line2: "",
    postal_code: "75001",
    city: "Paris",
    country: "FR",
    invoicing_email: "billing@acme.fr",
    synced_from_kyc_at: "2026-04-01T00:00:00Z",
    created_at: "2026-04-01T00:00:00Z",
    updated_at: "2026-04-01T00:00:00Z",
  },
  missing_fields: [],
  is_complete: true,
}

const useBillingProfileMock = vi.fn()
const syncMutateMock = vi.fn()
vi.mock("@/features/invoicing/hooks/use-billing-profile", () => ({
  useBillingProfile: () => useBillingProfileMock(),
  useSyncBillingProfile: () => ({ mutate: syncMutateMock, isPending: false }),
}))

const subscribeApiMock = vi.fn()
vi.mock("@/features/subscription/api/subscription-api", () => ({
  subscribe: (...args: unknown[]) => subscribeApiMock(...args),
}))

// ---- Helpers ----

function renderPage(): ReactElement {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return (
    <QueryClientProvider client={client}>
      <SubscribeEmbedPage />
    </QueryClientProvider>
  )
}

describe("SubscribeEmbedPage", () => {
  beforeEach(() => {
    mockSearchParams.clear()
    useBillingProfileMock.mockReset()
    syncMutateMock.mockReset()
    subscribeApiMock.mockReset()
    routerReplaceMock.mockClear()
    mockSearchParams.set("plan", "freelance")
    mockSearchParams.set("cycle", "monthly")
    mockSearchParams.set("auto_renew", "false")
    useBillingProfileMock.mockReturnValue({
      data: billingProfileSnapshot,
      isLoading: false,
    })
    // Default: hang on the call so step 2 sits in "preparing" — tests
    // override per-case for success / error paths.
    subscribeApiMock.mockReturnValue(new Promise(() => {}))
  })

  it("renders the billing step by default with the form", () => {
    render(renderPage())
    expect(screen.getByText(/Informations de facturation/i)).toBeDefined()
    expect(screen.getByTestId("billing-profile-form")).toBeDefined()
  })

  it("auto-syncs from Stripe KYC when the profile is incomplete", () => {
    // Snapshot with is_complete=false — independent of synced_from_kyc_at,
    // we always retry the KYC pre-fill so a previous partial sync that
    // filled nothing doesn't lock the user out of future attempts.
    useBillingProfileMock.mockReturnValue({
      data: {
        ...billingProfileSnapshot,
        is_complete: false,
        missing_fields: [{ field: "legal_name", reason: "required" }],
      },
      isLoading: false,
    })
    render(renderPage())
    expect(syncMutateMock).toHaveBeenCalledTimes(1)
  })

  it("does NOT auto-sync when the profile is already complete", () => {
    render(renderPage()) // default snapshot has is_complete=true
    expect(syncMutateMock).not.toHaveBeenCalled()
  })

  it("transitions to the payment step when BillingProfileForm fires onSaved", async () => {
    subscribeApiMock.mockResolvedValue({ client_secret: "cs_test_abc" })

    render(renderPage())

    // Trigger the onSaved hook from the stub.
    act(() => {
      screen.getByText("save-billing").click()
    })

    await waitFor(() => {
      expect(screen.getByText(/^Paiement$/i)).toBeDefined()
    })
    await waitFor(() => {
      expect(screen.getByTestId("embedded-checkout-provider").getAttribute("data-client-secret")).toBe("cs_test_abc")
    })
    expect(subscribeApiMock).toHaveBeenCalled()
  })

  it("renders an error card when the subscribe call fails", async () => {
    subscribeApiMock.mockRejectedValue(new Error("network down"))

    render(renderPage())
    act(() => {
      screen.getByText("save-billing").click()
    })

    await waitFor(() => {
      expect(
        screen.getByText(/Le paiement n'a pas pu démarrer/i),
      ).toBeDefined()
    })
  })

  it("shows an invalid params card when query params are missing", () => {
    mockSearchParams.delete("plan")
    render(renderPage())
    expect(screen.getByText(/Paramètres de souscription invalides/i)).toBeDefined()
  })

  it("renders the inline cycle toggle with monthly active by default", () => {
    render(renderPage())
    const toggle = screen.getByRole("tablist", { name: /Periode de facturation/i })
    expect(toggle).toBeDefined()
    const monthly = screen.getByRole("tab", { name: /Mensuel/i })
    const annual = screen.getByRole("tab", { name: /Annuel/i })
    expect(monthly.getAttribute("aria-selected")).toBe("true")
    expect(annual.getAttribute("aria-selected")).toBe("false")
  })

  it("hides the cycle toggle when params are invalid", () => {
    mockSearchParams.delete("cycle")
    render(renderPage())
    expect(screen.queryByRole("tablist", { name: /Periode de facturation/i })).toBeNull()
  })

  it("shows the freelance pricing on the toggle by default", () => {
    render(renderPage())
    expect(screen.getByText(/19 €\/mois/i)).toBeDefined()
    expect(screen.getByText(/15 €\/mois/i)).toBeDefined()
  })

  it("shows the agency pricing when plan=agency", () => {
    mockSearchParams.set("plan", "agency")
    render(renderPage())
    expect(screen.getByText(/49 €\/mois/i)).toBeDefined()
    expect(screen.getByText(/39 €\/mois/i)).toBeDefined()
  })

  it("clicking the annual tab rewrites the URL via router.replace", () => {
    render(renderPage())
    const annual = screen.getByRole("tab", { name: /Annuel/i })
    act(() => {
      annual.click()
    })
    expect(routerReplaceMock).toHaveBeenCalledTimes(1)
    const url = routerReplaceMock.mock.calls[0][0] as string
    expect(url).toContain("/subscribe/embed?")
    expect(url).toContain("cycle=annual")
    expect(url).toContain("plan=freelance")
    expect(url).toContain("auto_renew=false")
  })

  it("does NOT call router.replace when clicking the already-active tab", () => {
    render(renderPage())
    const monthly = screen.getByRole("tab", { name: /Mensuel/i })
    act(() => {
      monthly.click()
    })
    expect(routerReplaceMock).not.toHaveBeenCalled()
  })

  it("re-fires subscribe with the new cycle after the user toggles annual on the payment step", async () => {
    subscribeApiMock.mockResolvedValue({ client_secret: "cs_test_abc" })

    const { rerender } = render(renderPage())

    // Step 1 -> Step 2 (payment): triggers first subscribe() call
    act(() => {
      screen.getByText("save-billing").click()
    })
    await waitFor(() => {
      expect(subscribeApiMock).toHaveBeenCalledTimes(1)
    })
    expect(subscribeApiMock.mock.calls[0][0]).toMatchObject({
      plan: "freelance",
      billing_cycle: "monthly",
    })

    // User toggles to annual: router.replace updates the in-memory
    // searchParams (see mock at top of file) and we manually rerender
    // to simulate Next.js re-rendering on URL change. PaymentStep's
    // useEffect lists `billingCycle` in its deps, so the new render
    // re-fires `subscribe()` with the updated cycle.
    const annual = screen.getByRole("tab", { name: /Annuel/i })
    act(() => {
      annual.click()
    })
    rerender(renderPage())

    await waitFor(() => {
      expect(subscribeApiMock).toHaveBeenCalledTimes(2)
    })
    expect(subscribeApiMock.mock.calls[1][0]).toMatchObject({
      plan: "freelance",
      billing_cycle: "annual",
    })
  })
})
