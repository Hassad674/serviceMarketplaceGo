// Smoke tests for the BillingProfileInlineModal — the component that
// embeds the canonical billing-profile form in a modal so the proposal
// payment flow can keep the user on /projects/pay while they fix their
// billing identity. The form's behaviour is exhaustively tested in
// `features/invoicing/components/__tests__/billing-profile-form.test.tsx`;
// here we only assert the modal scaffolding (title, subtitle, open/close
// gating, onSaved propagation).

import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => `i18n:${key}`,
}))

// Stub the form so we don't have to wire a QueryClientProvider. We
// surface an `onSaved` button that lets the test fire the callback
// without touching any real network code.
vi.mock("@/features/invoicing/components/billing-profile-form", () => ({
  BillingProfileForm: ({ variant, onSaved }: { variant?: string; onSaved?: () => void }) => (
    <div data-testid="form-stub" data-variant={variant ?? ""}>
      <button data-testid="form-save" onClick={onSaved}>
        save
      </button>
    </div>
  ),
}))

beforeEach(() => {
  vi.clearAllMocks()
})

async function renderModal(props: {
  open: boolean
  onClose?: () => void
  onSaved?: () => void
}) {
  const { BillingProfileInlineModal } = await import(
    "../billing-profile-inline-modal"
  )
  return render(
    <BillingProfileInlineModal
      open={props.open}
      onClose={props.onClose ?? (() => {})}
      onSaved={props.onSaved ?? (() => {})}
    />,
  )
}

describe("BillingProfileInlineModal", () => {
  it("renders nothing when closed", async () => {
    await renderModal({ open: false })
    expect(screen.queryByTestId("form-stub")).not.toBeInTheDocument()
  })

  it("renders the canonical billing-profile form in compact variant when open", async () => {
    await renderModal({ open: true })
    const stub = screen.getByTestId("form-stub")
    expect(stub).toBeInTheDocument()
    expect(stub.dataset.variant).toBe("compact")
  })

  it("renders the i18n-keyed title + subtitle", async () => {
    await renderModal({ open: true })
    // The stubbed translator prefixes with "i18n:" so we can assert
    // the keys are wired and there are no hardcoded strings.
    expect(
      screen.getByText("i18n:title"),
    ).toBeInTheDocument()
    expect(
      screen.getByText("i18n:subtitle"),
    ).toBeInTheDocument()
  })

  it("propagates the form's onSaved callback to the parent", async () => {
    const onSaved = vi.fn()
    await renderModal({ open: true, onSaved })
    screen.getByTestId("form-save").click()
    expect(onSaved).toHaveBeenCalledTimes(1)
  })
})
