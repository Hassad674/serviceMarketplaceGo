import type { FileMetadata, Message, ProposalMessageMetadata, VoiceMetadata } from "../types"

// ---------------------------------------------------------------------------
// Metadata type guards — narrow the polymorphic `message.metadata`
// shape based on a discriminating field. Used by the bubble switch
// and the message area's filter step.
// ---------------------------------------------------------------------------

export function isProposalMetadata(metadata: unknown): metadata is ProposalMessageMetadata {
  return (
    metadata !== null &&
    typeof metadata === "object" &&
    "proposal_id" in (metadata as Record<string, unknown>)
  )
}

export function isFileMetadata(metadata: unknown): metadata is FileMetadata {
  return (
    metadata !== null &&
    typeof metadata === "object" &&
    "filename" in (metadata as Record<string, unknown>)
  )
}

export function isVoiceMetadata(metadata: unknown): metadata is VoiceMetadata {
  return (
    metadata !== null &&
    typeof metadata === "object" &&
    "duration" in (metadata as Record<string, unknown>)
  )
}

// ---------------------------------------------------------------------------
// System message type sets — used by MessageBubble to dispatch to the
// right renderer. Kept here so changes to the lifecycle (new event
// types) only have to be added once.
// ---------------------------------------------------------------------------

export const PROPOSAL_SYSTEM_TYPES = new Set([
  "proposal_accepted",
  "proposal_declined",
  "proposal_paid",
  "proposal_completed",
  "proposal_completion_rejected",
  "proposal_modified",
  // Phase 12 milestone-scoped types — emitted by the proposal
  // service (release notifications) and the scheduler worker
  // (auto-approval, auto-close).
  "milestone_released",
  "milestone_auto_approved",
  "proposal_cancelled",
  "proposal_auto_closed",
])

export const DISPUTE_SYSTEM_TYPES = new Set([
  "dispute_opened",
  "dispute_counter_proposal",
  "dispute_counter_accepted",
  "dispute_counter_rejected",
  "dispute_escalated",
  "dispute_resolved",
  "dispute_cancelled",
  "dispute_auto_resolved",
  "dispute_cancellation_requested",
  "dispute_cancellation_refused",
])

// Referral (apport d'affaires) system messages. Every lifecycle event
// posted by the Go referral service lands here — the widget renders
// role-appropriate accept / reject / negotiate buttons from the
// current user's perspective.
export const REFERRAL_SYSTEM_TYPES = new Set([
  "referral_intro_sent",
  "referral_intro_negotiated",
  "referral_intro_activated",
  "referral_intro_closed",
])

// ---------------------------------------------------------------------------
// Pure timeline helpers — derive sets of "stale" proposal ids that
// should be visually downplayed or filtered out of the rendered list.
// ---------------------------------------------------------------------------

export function computeSupersededIds(messages: Message[]): Set<string> {
  const superseded = new Set<string>()
  const parentIds = new Set<string>()
  for (const msg of messages) {
    if (msg.type === "proposal_modified" && isProposalMetadata(msg.metadata)) {
      const meta = msg.metadata as ProposalMessageMetadata
      if (meta.proposal_parent_id) {
        parentIds.add(meta.proposal_parent_id)
      }
    }
  }
  // Any proposal_sent whose proposal_id is a parent_id of a modified
  // version is superseded.
  for (const msg of messages) {
    if (
      (msg.type === "proposal_sent" || msg.type === "proposal_modified") &&
      isProposalMetadata(msg.metadata)
    ) {
      const meta = msg.metadata as ProposalMessageMetadata
      if (parentIds.has(meta.proposal_id)) {
        superseded.add(meta.proposal_id)
      }
    }
  }
  // Also mark older modified versions as superseded (keep only the latest).
  const versionMap = new Map<string, number>()
  for (const msg of messages) {
    if (
      (msg.type === "proposal_sent" || msg.type === "proposal_modified") &&
      isProposalMetadata(msg.metadata)
    ) {
      const meta = msg.metadata as ProposalMessageMetadata
      const rootId = meta.proposal_parent_id ?? meta.proposal_id
      const current = versionMap.get(rootId) ?? 0
      if (meta.proposal_version > current) {
        versionMap.set(rootId, meta.proposal_version)
      }
    }
  }
  for (const msg of messages) {
    if (
      (msg.type === "proposal_sent" || msg.type === "proposal_modified") &&
      isProposalMetadata(msg.metadata)
    ) {
      const meta = msg.metadata as ProposalMessageMetadata
      const rootId = meta.proposal_parent_id ?? meta.proposal_id
      const maxVersion = versionMap.get(rootId) ?? 1
      if (meta.proposal_version < maxVersion) {
        superseded.add(meta.proposal_id)
      }
    }
  }
  return superseded
}

// computeResolvedCompletionIds returns the set of proposal ids whose
// "completion_requested" state has been resolved by a subsequent
// system message. The message types listed below all signal that the
// client has already acted on (or moved past) the completion request,
// making the earlier yellow card stale.
//
// We deliberately include milestone_released / milestone_auto_approved
// so that approving milestone N of a multi-milestone proposal hides
// the old "Complétion demandée" bubble for THAT proposal, even though
// the proposal as a whole will see more completion requests for
// milestones N+1, N+2, etc. Each new request gets its own fresh card
// after the provider re-submits.
export function computeResolvedCompletionIds(messages: Message[]): Set<string> {
  const resolved = new Set<string>()
  const resolverTypes = new Set([
    "proposal_completed",
    "proposal_completion_rejected",
    "milestone_released",
    "milestone_auto_approved",
    "proposal_cancelled",
    "proposal_auto_closed",
  ])
  for (const msg of messages) {
    if (resolverTypes.has(msg.type) && isProposalMetadata(msg.metadata)) {
      const meta = msg.metadata as ProposalMessageMetadata
      resolved.add(meta.proposal_id)
    }
  }
  return resolved
}
