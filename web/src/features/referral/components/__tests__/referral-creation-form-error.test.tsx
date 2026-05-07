import { describe, expect, it, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import {
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query"

import { ApiError } from "@/shared/lib/api-client"

import { ReferralCreationForm } from "../referral-creation-form"

// next-intl is mocked to echo the translation key — the assertion
// then verifies the form picked the right key (alreadyInRelation)
// rather than rendering raw English / French copy that ships in
// `messages/{fr,en}.json`. Keeps the test stable when the copy is
// updated downstream.
vi.mock("next-intl", () => ({
  useTranslations: (_namespace?: string) => (key: string) => `referral.errors.${key}`,
}))

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
}))

// The picker components mount their own pickers via portals — we
// stub them to expose a deterministic "pick this party" trigger so
// the test can drive the form to a submittable state without
// reproducing the full picker UX.
vi.mock("../client-picker", () => ({
  ClientPicker: ({
    onChange,
  }: {
    onChange: (s: { userId: string; name: string; orgType: string }) => void
  }) => (
    <button
      type="button"
      data-testid="pick-client"
      onClick={() =>
        onChange({
          userId: "11111111-1111-1111-1111-111111111111",
          name: "Test Client",
          orgType: "enterprise",
        })
      }
    >
      pick client
    </button>
  ),
}))

vi.mock("../provider-picker", () => ({
  ProviderPicker: ({
    onChange,
  }: {
    onChange: (s: { userId: string; name: string; orgType: string }) => void
  }) => (
    <button
      type="button"
      data-testid="pick-provider"
      onClick={() =>
        onChange({
          userId: "22222222-2222-2222-2222-222222222222",
          name: "Test Provider",
          orgType: "freelance",
        })
      }
    >
      pick provider
    </button>
  ),
}))

const { mockCreate } = vi.hoisted(() => ({ mockCreate: vi.fn() }))
vi.mock("@/features/referral/hooks/use-referrals", () => ({
  useCreateReferral: () => ({
    mutateAsync: mockCreate,
    isPending: false,
  }),
}))

function renderForm() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <ReferralCreationForm />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  mockCreate.mockReset()
})

async function fillFormAndSubmit() {
  const user = userEvent.setup()
  await user.click(screen.getByTestId("pick-provider"))
  await user.click(screen.getByTestId("pick-client"))
  // Pitches are required before the gate runs — use the placeholder text
  // to locate the textareas without depending on French copy.
  const provPitch = screen.getByPlaceholderText(/refonte branding/i)
  const cliPitch = screen.getByPlaceholderText(/avec qui je travaille/i)
  await user.type(provPitch, "intro pour le presta")
  await user.type(cliPitch, "intro pour le client")
  await user.click(screen.getByRole("button", { name: /Envoyer/i }))
}

describe("ReferralCreationForm anti-fraud error mapping", () => {
  it("displays the alreadyInRelation i18n key when backend returns 409 already_in_relation", async () => {
    mockCreate.mockRejectedValueOnce(
      new ApiError(
        409,
        "already_in_relation",
        "provider and client party are already in relation",
        null,
      ),
    )

    renderForm()
    await fillFormAndSubmit()

    await waitFor(() => {
      expect(
        screen.getByRole("alert"),
      ).toHaveTextContent("referral.errors.alreadyInRelation")
    })
  })

  it("falls back to the raw error message for unrelated failures", async () => {
    mockCreate.mockRejectedValueOnce(
      new Error("Une erreur est survenue lors de la création de l'intro."),
    )

    renderForm()
    await fillFormAndSubmit()

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(
        /erreur est survenue/i,
      )
    })
  })

  it("falls back to the raw error message for non-409 ApiError", async () => {
    mockCreate.mockRejectedValueOnce(
      new ApiError(500, "internal_error", "boom", null),
    )

    renderForm()
    await fillFormAndSubmit()

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(/boom/)
    })
  })
})
