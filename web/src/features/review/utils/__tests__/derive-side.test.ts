import { describe, it, expect } from "vitest"
import { deriveReviewSide } from "../derive-side"

describe("deriveReviewSide", () => {
  const clientOrg = "org-client-1"
  const providerOrg = "org-provider-1"

  it("returns client_to_provider when the viewer's org matches the client", () => {
    const side = deriveReviewSide(clientOrg, {
      client_id: clientOrg,
      provider_id: providerOrg,
    })
    expect(side).toBe("client_to_provider")
  })

  it("returns provider_to_client when the viewer's org matches the provider", () => {
    const side = deriveReviewSide(providerOrg, {
      client_id: clientOrg,
      provider_id: providerOrg,
    })
    expect(side).toBe("provider_to_client")
  })

  it("returns null when the viewer is neither client nor provider", () => {
    const side = deriveReviewSide("org-bystander", {
      client_id: clientOrg,
      provider_id: providerOrg,
    })
    expect(side).toBeNull()
  })

  it("returns null when the viewer's org id is missing", () => {
    expect(
      deriveReviewSide(null, { client_id: clientOrg, provider_id: providerOrg }),
    ).toBeNull()
    expect(
      deriveReviewSide(undefined, { client_id: clientOrg, provider_id: providerOrg }),
    ).toBeNull()
  })

  it("returns null when the source is missing", () => {
    expect(deriveReviewSide(clientOrg, null)).toBeNull()
    expect(deriveReviewSide(clientOrg, undefined)).toBeNull()
  })

  it("accepts the messaging metadata shape with proposal_* prefixes", () => {
    const side = deriveReviewSide(providerOrg, {
      proposal_client_id: clientOrg,
      proposal_provider_id: providerOrg,
    })
    expect(side).toBe("provider_to_client")
  })
})
