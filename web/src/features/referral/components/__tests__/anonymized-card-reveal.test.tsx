import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"

import { AnonymizedProviderCard } from "../anonymized-provider-card"
import { AnonymizedClientCard } from "../anonymized-client-card"
import type {
  ProviderSnapshot,
  ClientSnapshot,
} from "@/shared/types/referral"

vi.mock("next-intl", () => ({
  useTranslations:
    (namespace?: string) =>
    (key: string) =>
      namespace ? `${namespace}.${key}` : key,
}))

const providerFixture: ProviderSnapshot = {
  expertise_domains: ["dev"],
  years_experience: 5,
}

const clientFixture: ClientSnapshot = {
  industry: "SaaS",
}

describe("AnonymizedProviderCard — masked variant (non-owner viewers)", () => {
  it("renders the masked snapshot with no display-name reveal", () => {
    render(<AnonymizedProviderCard snapshot={providerFixture} />)
    expect(
      screen.queryByTestId("anonymized-provider-revealed"),
    ).not.toBeInTheDocument()
    // The masked subtitle is rendered (mocked translator returns key path).
    expect(
      screen.getByText("referralIdentity.maskedSubtitle"),
    ).toBeInTheDocument()
  })

  it("does NOT render the legacy reveal-link or 'Identité visible' badge", () => {
    render(<AnonymizedProviderCard snapshot={providerFixture} />)
    // Legacy data-testid used by the previous design.
    expect(
      screen.queryByTestId("anonymized-provider-reveal-link"),
    ).not.toBeInTheDocument()
    // No "Voir le profil" CTA text leaks through.
    expect(screen.queryByText(/Voir le profil/i)).not.toBeInTheDocument()
    // No "Identité visible" eyebrow either.
    expect(screen.queryByText(/Identité visible/i)).not.toBeInTheDocument()
  })
})

describe("AnonymizedProviderCard — revealed variant (apporteur owner)", () => {
  it("renders ONLY the display name, role label, and no CTA", () => {
    render(
      <AnonymizedProviderCard
        snapshot={providerFixture}
        revealed
        displayName="Atelier Lumen"
      />,
    )
    const revealedCard = screen.getByTestId("anonymized-provider-revealed")
    expect(revealedCard).toBeInTheDocument()
    expect(screen.getByTestId("revealed-identity-name").textContent).toBe(
      "Atelier Lumen",
    )
    // Role label is the only secondary content.
    expect(
      screen.getByText("referralIdentity.providerTitle"),
    ).toBeInTheDocument()
    // Forbidden legacy bits.
    expect(
      screen.queryByTestId("anonymized-provider-reveal-link"),
    ).not.toBeInTheDocument()
    expect(screen.queryByText(/Voir le profil/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/Identité visible/i)).not.toBeInTheDocument()
    // Masked subtitle MUST NOT appear in revealed mode — the apporteur
    // already knows who they introduced.
    expect(
      screen.queryByText("referralIdentity.maskedSubtitle"),
    ).not.toBeInTheDocument()
  })

  it("renders the em-dash placeholder when displayName is empty", () => {
    render(
      <AnonymizedProviderCard
        snapshot={providerFixture}
        revealed
        displayName=""
      />,
    )
    expect(screen.getByTestId("revealed-identity-name").textContent).toBe("—")
  })
})

describe("AnonymizedClientCard — masked variant", () => {
  it("renders the masked snapshot with no reveal-link", () => {
    render(<AnonymizedClientCard snapshot={clientFixture} />)
    expect(
      screen.queryByTestId("anonymized-client-revealed"),
    ).not.toBeInTheDocument()
    expect(
      screen.queryByTestId("anonymized-client-reveal-link"),
    ).not.toBeInTheDocument()
    expect(
      screen.getByText("referralIdentity.maskedSubtitle"),
    ).toBeInTheDocument()
  })
})

describe("AnonymizedClientCard — revealed variant (apporteur owner)", () => {
  it("renders ONLY the display name + role label", () => {
    render(
      <AnonymizedClientCard
        snapshot={clientFixture}
        revealed
        displayName="Banque du Sud"
      />,
    )
    expect(
      screen.getByTestId("anonymized-client-revealed"),
    ).toBeInTheDocument()
    expect(screen.getByTestId("revealed-identity-name").textContent).toBe(
      "Banque du Sud",
    )
    expect(
      screen.getByText("referralIdentity.clientTitle"),
    ).toBeInTheDocument()
    expect(
      screen.queryByTestId("anonymized-client-reveal-link"),
    ).not.toBeInTheDocument()
    expect(screen.queryByText(/Voir le profil/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/Identité visible/i)).not.toBeInTheDocument()
    expect(
      screen.queryByText("referralIdentity.maskedSubtitle"),
    ).not.toBeInTheDocument()
  })

  it("renders the em-dash placeholder when displayName is empty", () => {
    render(
      <AnonymizedClientCard
        snapshot={clientFixture}
        revealed
        displayName={undefined}
      />,
    )
    expect(screen.getByTestId("revealed-identity-name").textContent).toBe("—")
  })
})
