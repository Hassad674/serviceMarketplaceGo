import type { ReviewSide } from "@/shared/types/review"

/**
 * Minimal shape of a proposal (or proposal-message metadata) needed
 * to figure out which side of a double-blind review the current
 * viewer is on.
 *
 * Since the phase-4 org refactor, both fields are ORG ids
 * (`CurrentOrganization.id`), NOT user ids. A single org may have
 * multiple operators — any of them, acting on behalf of the org,
 * shares the same "side" on the review.
 */
export type ReviewSideSource = {
  client_id: string
  provider_id: string
}

/**
 * Alternative shape used by the messaging feature where the proposal
 * metadata keys are prefixed with `proposal_`.
 */
export type ReviewSideMessageSource = {
  proposal_client_id: string
  proposal_provider_id: string
}

/**
 * Returns which side of the double-blind review the current viewer
 * is on, given their organization id and the proposal participants.
 *
 * Returns `null` when the viewer's org matches neither side — in that
 * case the UI must hide any review CTA entirely: only participants of
 * a mission can review its counterpart.
 *
 * All inputs are optional strings so callers can forward values
 * straight from query hooks that may still be loading (undefined)
 * without having to guard every call site.
 */
export function deriveReviewSide(
  viewerOrgId: string | null | undefined,
  source: ReviewSideSource | ReviewSideMessageSource | null | undefined,
): ReviewSide | null {
  if (!viewerOrgId || !source) return null

  const clientId = "client_id" in source ? source.client_id : source.proposal_client_id
  const providerId = "provider_id" in source ? source.provider_id : source.proposal_provider_id

  if (viewerOrgId === clientId) return "client_to_provider"
  if (viewerOrgId === providerId) return "provider_to_client"
  return null
}
