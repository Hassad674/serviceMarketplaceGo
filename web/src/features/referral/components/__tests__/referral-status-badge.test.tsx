import { describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"

import { ReferralStatusBadge } from "../referral-status-badge"

describe("ReferralStatusBadge", () => {
  it("renders the French label for each status", () => {
    const cases: Array<[string, string]> = [
      ["pending_provider", "En attente du prestataire"],
      ["pending_referrer", "En attente de l'apporteur"],
      ["pending_client", "En attente du client"],
      ["active", "Active"],
      ["rejected", "Refusée"],
      ["expired", "Expirée"],
      ["cancelled", "Annulée"],
      ["terminated", "Terminée"],
    ]
    for (const [status, label] of cases) {
      const { unmount } = render(
        // @ts-expect-error — narrowing via the test loop
        <ReferralStatusBadge status={status} />,
      )
      expect(screen.getByText(label)).toBeInTheDocument()
      unmount()
    }
  })

  it("applies the amber tone classes for pending statuses", () => {
    render(<ReferralStatusBadge status="pending_provider" />)
    const badge = screen.getByText("En attente du prestataire")
    expect(badge.className).toContain("bg-amber-50")
  })

  it("applies the emerald tone classes for active status", () => {
    render(<ReferralStatusBadge status="active" />)
    const badge = screen.getByText("Active")
    expect(badge.className).toContain("bg-emerald-50")
  })
})
