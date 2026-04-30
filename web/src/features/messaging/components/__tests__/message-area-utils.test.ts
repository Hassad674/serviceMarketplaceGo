import { describe, it, expect } from "vitest"
import {
  computeResolvedCompletionIds,
  computeSupersededIds,
  DISPUTE_SYSTEM_TYPES,
  isFileMetadata,
  isProposalMetadata,
  isVoiceMetadata,
  PROPOSAL_SYSTEM_TYPES,
  REFERRAL_SYSTEM_TYPES,
} from "../message-area-utils"
import type { Message, ProposalMessageMetadata } from "../../types"

function makeMessage(overrides: Partial<Message> = {}): Message {
  return {
    id: "m",
    conversation_id: "c",
    sender_id: "u",
    content: "",
    type: "text",
    metadata: null,
    seq: 1,
    status: "sent",
    edited_at: null,
    deleted_at: null,
    created_at: "2026-04-01T00:00:00Z",
    ...overrides,
  }
}

function makeProposalMeta(
  overrides: Partial<ProposalMessageMetadata> = {},
): ProposalMessageMetadata {
  return {
    proposal_id: "p1",
    proposal_title: "T",
    proposal_amount: 0,
    proposal_status: "pending",
    proposal_deadline: null,
    proposal_sender_name: "S",
    proposal_documents_count: 0,
    proposal_version: 1,
    proposal_parent_id: null,
    proposal_client_id: "c1",
    proposal_provider_id: "pr1",
    ...overrides,
  }
}

describe("type guards", () => {
  it("isProposalMetadata accepts only objects with proposal_id", () => {
    expect(isProposalMetadata({ proposal_id: "x" })).toBe(true)
    expect(isProposalMetadata({})).toBe(false)
    expect(isProposalMetadata(null)).toBe(false)
    expect(isProposalMetadata("nope")).toBe(false)
  })

  it("isFileMetadata accepts only objects with filename", () => {
    expect(isFileMetadata({ filename: "x.pdf" })).toBe(true)
    expect(isFileMetadata({})).toBe(false)
    expect(isFileMetadata(null)).toBe(false)
  })

  it("isVoiceMetadata accepts only objects with duration", () => {
    expect(isVoiceMetadata({ duration: 12 })).toBe(true)
    expect(isVoiceMetadata({})).toBe(false)
    expect(isVoiceMetadata(null)).toBe(false)
  })
})

describe("constant sets are non-empty and disjoint per category", () => {
  it("PROPOSAL_SYSTEM_TYPES has the canonical entries", () => {
    expect(PROPOSAL_SYSTEM_TYPES.has("proposal_accepted")).toBe(true)
    expect(PROPOSAL_SYSTEM_TYPES.has("milestone_released")).toBe(true)
  })

  it("DISPUTE_SYSTEM_TYPES has the canonical entries", () => {
    expect(DISPUTE_SYSTEM_TYPES.has("dispute_opened")).toBe(true)
    expect(DISPUTE_SYSTEM_TYPES.has("dispute_resolved")).toBe(true)
  })

  it("REFERRAL_SYSTEM_TYPES has the canonical entries", () => {
    expect(REFERRAL_SYSTEM_TYPES.has("referral_intro_sent")).toBe(true)
    expect(REFERRAL_SYSTEM_TYPES.has("referral_intro_activated")).toBe(true)
  })
})

describe("computeSupersededIds", () => {
  it("returns empty when no proposal_modified messages", () => {
    expect(computeSupersededIds([])).toEqual(new Set())
  })

  it("marks the original proposal_sent superseded by a proposal_modified", () => {
    const messages: Message[] = [
      makeMessage({
        id: "1",
        type: "proposal_sent",
        metadata: makeProposalMeta({
          proposal_id: "p-original",
          proposal_version: 1,
        }),
      }),
      makeMessage({
        id: "2",
        type: "proposal_modified",
        metadata: makeProposalMeta({
          proposal_id: "p-v2",
          proposal_parent_id: "p-original",
          proposal_version: 2,
        }),
      }),
    ]
    const result = computeSupersededIds(messages)
    expect(result.has("p-original")).toBe(true)
    expect(result.has("p-v2")).toBe(false)
  })

  it("marks earlier modifications superseded by a later modification", () => {
    const messages: Message[] = [
      makeMessage({
        id: "1",
        type: "proposal_sent",
        metadata: makeProposalMeta({
          proposal_id: "p-root",
          proposal_version: 1,
        }),
      }),
      makeMessage({
        id: "2",
        type: "proposal_modified",
        metadata: makeProposalMeta({
          proposal_id: "p-v2",
          proposal_parent_id: "p-root",
          proposal_version: 2,
        }),
      }),
      makeMessage({
        id: "3",
        type: "proposal_modified",
        metadata: makeProposalMeta({
          proposal_id: "p-v3",
          proposal_parent_id: "p-root",
          proposal_version: 3,
        }),
      }),
    ]
    const result = computeSupersededIds(messages)
    expect(result.has("p-root")).toBe(true)
    expect(result.has("p-v2")).toBe(true)
    expect(result.has("p-v3")).toBe(false)
  })

  it("ignores non-proposal messages with non-proposal metadata", () => {
    const messages: Message[] = [
      makeMessage({ id: "1", type: "text", content: "hi" }),
      makeMessage({ id: "2", type: "file", metadata: { filename: "x" } as never }),
    ]
    expect(computeSupersededIds(messages)).toEqual(new Set())
  })
})

describe("computeResolvedCompletionIds", () => {
  it("returns empty when no resolver messages", () => {
    expect(computeResolvedCompletionIds([])).toEqual(new Set())
  })

  it.each([
    "proposal_completed",
    "proposal_completion_rejected",
    "milestone_released",
    "milestone_auto_approved",
    "proposal_cancelled",
    "proposal_auto_closed",
  ] as const)("marks the proposal as resolved on %s", (type) => {
    const messages: Message[] = [
      makeMessage({
        id: "1",
        type,
        metadata: makeProposalMeta({ proposal_id: "p-x" }),
      }),
    ]
    expect(computeResolvedCompletionIds(messages).has("p-x")).toBe(true)
  })

  it("ignores resolver-typed messages without a proposal metadata", () => {
    const messages: Message[] = [
      makeMessage({ id: "1", type: "proposal_completed", metadata: null }),
    ]
    expect(computeResolvedCompletionIds(messages)).toEqual(new Set())
  })

  it("does not flag resolution for unrelated message types", () => {
    const messages: Message[] = [
      makeMessage({
        id: "1",
        type: "proposal_completion_requested",
        metadata: makeProposalMeta({ proposal_id: "p-x" }),
      }),
    ]
    expect(computeResolvedCompletionIds(messages)).toEqual(new Set())
  })
})
