/**
 * applicantProfileHref — persona-aware "View profile" routing.
 *
 * Pins the contract: the link prefix branches on applicant_kind and the
 * org id (NOT the user id) is appended. Regression for the bug where a
 * freelance candidate's "View profile" sent the enterprise to a 404
 * because /freelancers/<userId> never resolves — the public profile
 * route looks up organizations.id.
 */
import { describe, expect, it } from "vitest"
import { applicantProfileHref } from "../applicant-profile-href"

describe("applicantProfileHref", () => {
  const orgId = "org-uuid-123"

  it("routes agency candidates to /agencies/<orgId>", () => {
    expect(applicantProfileHref("agency", orgId)).toBe(`/agencies/${orgId}`)
  })

  it("routes freelance candidates to /freelancers/<orgId>", () => {
    expect(applicantProfileHref("freelance", orgId)).toBe(
      `/freelancers/${orgId}`,
    )
  })

  it("routes referrer candidates to /referrers/<orgId>", () => {
    expect(applicantProfileHref("referrer", orgId)).toBe(`/referrers/${orgId}`)
  })

  it("falls back to /freelancers/<orgId> for unknown kinds", () => {
    // Stale cached row pre-applicant_kind migration could surface here.
    expect(applicantProfileHref("", orgId)).toBe(`/freelancers/${orgId}`)
    expect(applicantProfileHref("provider_personal", orgId)).toBe(
      `/freelancers/${orgId}`,
    )
  })

  it("never appends an empty trailing id", () => {
    // Defensive — even if the org id is empty the URL shape stays valid
    // (/freelancers/) so the page boundary error.tsx renders rather
    // than a malformed URL crashing the router.
    expect(applicantProfileHref("agency", "")).toBe("/agencies/")
  })
})
