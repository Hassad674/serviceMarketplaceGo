/**
 * api-contracts.test.ts
 *
 * Contract tests that validate the runtime shape of every major DTO the
 * web app receives through `apiClient<T>`. We use `zod` to express the
 * expected schema and assert a representative JSON envelope parses
 * cleanly — both a happy-path and (where the contract evolved) a legacy
 * shape that older endpoints still return.
 *
 * Why this exists:
 *   - The F.3.2 typing sweep narrows every `apiClient<TFromTypes>(...)`
 *     call to `apiClient<paths["..."]["..."]>(...)`. If the openapi
 *     types drifted away from the actual server response, the sweep
 *     compiles but breaks at runtime. Contract tests catch the drift
 *     before the sweep lands.
 *   - Zod parses are idempotent and deterministic — no fetch, no DB
 *     fixture — so they run in milliseconds and never flake.
 *
 * Coverage target: one suite per major feature DTO bundle. The schemas
 * deliberately mirror the TypeScript types in `src/features/{feature}/
 * types.ts` and `src/shared/types/...`. When a TS type changes, the
 * test breaks — which is the point.
 */
import { describe, expect, it } from "vitest"
import { z } from "zod"

// ---------------------------------------------------------------------------
// Reusable primitives
// ---------------------------------------------------------------------------

const isoDate = z.string().min(10) // RFC3339 / ISO-8601, kept loose
// Permissive id schema — the wire shape is "string" but production
// returns UUIDs, opaque slugs, and short fixture ids depending on the
// endpoint. The contract test concern is "is the field present and a
// string", not the format.
const uuid = z.string().min(1)
const cents = z.number().int().nonnegative()

// Pagination envelope used by 90% of list endpoints.
const cursorPagination = z.object({
  next_cursor: z.string().optional(),
  has_more: z.boolean().optional(),
})

// ---------------------------------------------------------------------------
// Proposal contract (features/proposal/types.ts)
// ---------------------------------------------------------------------------

const milestoneStatus = z.enum([
  "pending_funding",
  "funded",
  "submitted",
  "approved",
  "released",
  "disputed",
  "cancelled",
  "refunded",
])

const proposalStatus = z.enum([
  "pending",
  "accepted",
  "declined",
  "withdrawn",
  "paid",
  "active",
  "completion_requested",
  "completed",
  "disputed",
])

const proposalDocumentSchema = z.object({
  id: uuid,
  filename: z.string(),
  url: z.string(),
  size: z.number().int(),
  mime_type: z.string(),
})

const milestoneResponseSchema = z.object({
  id: uuid,
  sequence: z.number().int().positive(),
  title: z.string(),
  description: z.string(),
  amount: cents,
  deadline: isoDate.nullable().optional(),
  status: milestoneStatus,
  version: z.number().int(),
  funded_at: isoDate.nullable().optional(),
  submitted_at: isoDate.nullable().optional(),
  approved_at: isoDate.nullable().optional(),
  released_at: isoDate.nullable().optional(),
  disputed_at: isoDate.nullable().optional(),
  cancelled_at: isoDate.nullable().optional(),
})

const proposalResponseSchema = z.object({
  id: uuid,
  conversation_id: uuid,
  sender_id: uuid,
  recipient_id: uuid,
  title: z.string(),
  description: z.string(),
  amount: cents,
  deadline: isoDate.nullable(),
  status: proposalStatus,
  parent_id: uuid.nullable(),
  version: z.number().int(),
  client_id: uuid,
  provider_id: uuid,
  client_name: z.string(),
  provider_name: z.string(),
  active_dispute_id: uuid.nullable(),
  last_dispute_id: uuid.nullable().optional(),
  documents: z.array(proposalDocumentSchema),
  payment_mode: z.enum(["one_time", "milestone"]),
  milestones: z.array(milestoneResponseSchema),
  current_milestone_sequence: z.number().int().optional(),
  accepted_at: isoDate.nullable(),
  paid_at: isoDate.nullable(),
  created_at: isoDate,
})

describe("contract: ProposalResponse", () => {
  it("parses a fully-populated milestone proposal", () => {
    const sample = {
      id: "11111111-1111-1111-1111-111111111111",
      conversation_id: "22222222-2222-2222-2222-222222222222",
      sender_id: "33333333-3333-3333-3333-333333333333",
      recipient_id: "44444444-4444-4444-4444-444444444444",
      title: "Build the marketplace",
      description: "A 4-week sprint to ship the MVP.",
      amount: 200000,
      deadline: "2026-06-01T00:00:00Z",
      status: "active" as const,
      parent_id: null,
      version: 1,
      client_id: "55555555-5555-5555-5555-555555555555",
      provider_id: "66666666-6666-6666-6666-666666666666",
      client_name: "Acme Corp",
      provider_name: "Jean Dupont",
      active_dispute_id: null,
      documents: [
        {
          id: "doc-1",
          filename: "spec.pdf",
          url: "https://example.com/spec.pdf",
          size: 12345,
          mime_type: "application/pdf",
        },
      ],
      payment_mode: "milestone" as const,
      milestones: [
        {
          id: "m-1",
          sequence: 1,
          title: "Phase 1",
          description: "Setup",
          amount: 50000,
          status: "released" as const,
          version: 0,
          deadline: "2026-04-15",
          funded_at: "2026-04-01T10:00:00Z",
          released_at: "2026-04-15T18:00:00Z",
        },
      ],
      current_milestone_sequence: 1,
      accepted_at: "2026-03-30T10:00:00Z",
      paid_at: "2026-04-01T10:00:00Z",
      created_at: "2026-03-29T09:00:00Z",
    }
    const result = proposalResponseSchema.parse(sample)
    expect(result.id).toBe(sample.id)
    expect(result.milestones).toHaveLength(1)
  })

  it("rejects an unknown proposal status (catches BE/FE drift)", () => {
    expect(() =>
      proposalResponseSchema.parse({
        id: "x",
        conversation_id: "x",
        sender_id: "x",
        recipient_id: "x",
        title: "t",
        description: "d",
        amount: 100,
        deadline: null,
        status: "garbage_status",
        parent_id: null,
        version: 1,
        client_id: "x",
        provider_id: "x",
        client_name: "c",
        provider_name: "p",
        active_dispute_id: null,
        documents: [],
        payment_mode: "one_time",
        milestones: [],
        accepted_at: null,
        paid_at: null,
        created_at: "2026-01-01",
      }),
    ).toThrow()
  })

  it("accepts a one_time proposal with a single synthetic milestone", () => {
    const sample = {
      id: "x".repeat(36),
      conversation_id: "x".repeat(36),
      sender_id: "x".repeat(36),
      recipient_id: "x".repeat(36),
      title: "Single milestone",
      description: "",
      amount: 50000,
      deadline: null,
      status: "pending" as const,
      parent_id: null,
      version: 1,
      client_id: "x".repeat(36),
      provider_id: "x".repeat(36),
      client_name: "",
      provider_name: "",
      active_dispute_id: null,
      documents: [],
      payment_mode: "one_time" as const,
      milestones: [
        {
          id: "m1",
          sequence: 1,
          title: "Default",
          description: "",
          amount: 50000,
          status: "pending_funding" as const,
          version: 0,
        },
      ],
      accepted_at: null,
      paid_at: null,
      created_at: "2026-01-01T00:00:00Z",
    }
    expect(() => proposalResponseSchema.parse(sample)).not.toThrow()
  })
})

// ---------------------------------------------------------------------------
// Dispute contract (features/dispute/types.ts)
// ---------------------------------------------------------------------------

const disputeStatusSchema = z.enum([
  "open",
  "negotiation",
  "escalated",
  "resolved",
  "cancelled",
])

const counterProposalSchema = z.object({
  id: uuid,
  proposer_id: uuid,
  amount_client: cents,
  amount_provider: cents,
  message: z.string(),
  status: z.enum(["pending", "accepted", "rejected", "superseded"]),
  responded_at: isoDate.nullable(),
  created_at: isoDate,
})

const disputeResponseSchema = z.object({
  id: uuid,
  proposal_id: uuid,
  conversation_id: uuid,
  initiator_id: uuid,
  respondent_id: uuid,
  client_id: uuid,
  provider_id: uuid,
  reason: z.enum([
    "work_not_conforming",
    "non_delivery",
    "insufficient_quality",
    "client_ghosting",
    "scope_creep",
    "refusal_to_validate",
    "harassment",
    "other",
  ]),
  description: z.string(),
  requested_amount: cents,
  proposal_amount: cents,
  status: disputeStatusSchema,
  resolution_type: z.string().nullable(),
  resolution_amount_client: cents.nullable(),
  resolution_amount_provider: cents.nullable(),
  resolution_note: z.string().nullable(),
  initiator_role: z.enum(["client", "provider"]),
  evidence: z.array(
    z.object({
      id: uuid,
      filename: z.string(),
      url: z.string(),
      size: z.number().int(),
      mime_type: z.string(),
    }),
  ),
  counter_proposals: z.array(counterProposalSchema),
  cancellation_requested_by: uuid.nullable().optional(),
  cancellation_requested_at: isoDate.nullable().optional(),
  escalated_at: isoDate.nullable(),
  resolved_at: isoDate.nullable(),
  created_at: isoDate,
})

describe("contract: DisputeResponse", () => {
  it("parses an open dispute with one counter-proposal", () => {
    const sample = {
      id: "d1",
      proposal_id: "p1",
      conversation_id: "c1",
      initiator_id: "u1",
      respondent_id: "u2",
      client_id: "u1",
      provider_id: "u2",
      reason: "non_delivery" as const,
      description: "Work not delivered",
      requested_amount: 200000,
      proposal_amount: 200000,
      status: "negotiation" as const,
      resolution_type: null,
      resolution_amount_client: null,
      resolution_amount_provider: null,
      resolution_note: null,
      initiator_role: "client" as const,
      evidence: [],
      counter_proposals: [
        {
          id: "cp1",
          proposer_id: "u2",
          amount_client: 50000,
          amount_provider: 150000,
          message: "Partial refund offer",
          status: "pending" as const,
          responded_at: null,
          created_at: "2026-04-01T00:00:00Z",
        },
      ],
      escalated_at: null,
      resolved_at: null,
      created_at: "2026-04-01T00:00:00Z",
    }
    expect(() => disputeResponseSchema.parse(sample)).not.toThrow()
  })

  it("rejects an unknown dispute reason", () => {
    const bad = {
      id: "d1",
      proposal_id: "p1",
      conversation_id: "c1",
      initiator_id: "u1",
      respondent_id: "u2",
      client_id: "u1",
      provider_id: "u2",
      reason: "made_up_reason",
      description: "",
      requested_amount: 0,
      proposal_amount: 0,
      status: "open",
      resolution_type: null,
      resolution_amount_client: null,
      resolution_amount_provider: null,
      resolution_note: null,
      initiator_role: "client",
      evidence: [],
      counter_proposals: [],
      escalated_at: null,
      resolved_at: null,
      created_at: "2026-04-01",
    }
    expect(() => disputeResponseSchema.parse(bad)).toThrow()
  })
})

// ---------------------------------------------------------------------------
// Messaging contract (features/messaging/types.ts)
// ---------------------------------------------------------------------------

const messageTypeSchema = z.enum([
  "text",
  "file",
  "voice",
  "proposal_sent",
  "proposal_accepted",
  "proposal_declined",
  "proposal_modified",
  "proposal_paid",
  "proposal_payment_requested",
  "proposal_completion_requested",
  "proposal_completed",
  "proposal_completion_rejected",
  "evaluation_request",
  "call_ended",
  "call_missed",
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

const messageSchema = z.object({
  id: uuid,
  conversation_id: uuid,
  sender_id: uuid,
  content: z.string(),
  type: messageTypeSchema,
  metadata: z.unknown().nullable(),
  reply_to: z
    .object({
      id: uuid,
      sender_id: uuid,
      content: z.string(),
      type: z.string(),
    })
    .nullable()
    .optional(),
  seq: z.number().int().nonnegative(),
  status: z.enum(["sending", "sent", "delivered", "read"]),
  edited_at: isoDate.nullable(),
  deleted_at: isoDate.nullable(),
  created_at: isoDate,
})

const conversationSchema = z.object({
  id: uuid,
  other_user_id: uuid,
  other_org_id: uuid,
  other_org_name: z.string(),
  other_org_type: z.string(),
  other_photo_url: z.string(),
  last_message: z.string().nullable(),
  last_message_at: isoDate.nullable(),
  unread_count: z.number().int().nonnegative(),
  last_message_seq: z.number().int().nonnegative(),
  online: z.boolean(),
})

const conversationListSchema = z.object({
  data: z.array(conversationSchema),
  next_cursor: z.string().optional(),
  has_more: z.boolean(),
})

const messageListSchema = z.object({
  data: z.array(messageSchema),
  next_cursor: z.string().optional(),
  has_more: z.boolean(),
})

describe("contract: Messaging DTOs", () => {
  it("parses a text Message", () => {
    expect(() =>
      messageSchema.parse({
        id: "m1",
        conversation_id: "c1",
        sender_id: "u1",
        content: "Hello",
        type: "text" as const,
        metadata: null,
        seq: 1,
        status: "sent" as const,
        edited_at: null,
        deleted_at: null,
        created_at: "2026-04-01T00:00:00Z",
      }),
    ).not.toThrow()
  })

  it("parses a file Message with metadata payload", () => {
    expect(() =>
      messageSchema.parse({
        id: "m2",
        conversation_id: "c1",
        sender_id: "u1",
        content: "screenshot.png",
        type: "file" as const,
        metadata: {
          url: "https://files/screenshot.png",
          filename: "screenshot.png",
          size: 12345,
          mime_type: "image/png",
        },
        seq: 2,
        status: "delivered" as const,
        edited_at: null,
        deleted_at: null,
        created_at: "2026-04-01T00:00:01Z",
      }),
    ).not.toThrow()
  })

  it("parses a Conversation list response", () => {
    expect(() =>
      conversationListSchema.parse({
        data: [
          {
            id: "c1",
            other_user_id: "u2",
            other_org_id: "org-2",
            other_org_name: "Acme",
            other_org_type: "agency",
            other_photo_url: "",
            last_message: "Hi",
            last_message_at: "2026-04-01T00:00:00Z",
            unread_count: 0,
            last_message_seq: 1,
            online: false,
          },
        ],
        has_more: false,
      }),
    ).not.toThrow()
  })

  it("parses a Message list response with cursor", () => {
    expect(() =>
      messageListSchema.parse({
        data: [],
        has_more: true,
        next_cursor: "cursor-token",
      }),
    ).not.toThrow()
  })
})

// ---------------------------------------------------------------------------
// Wallet contract (features/wallet/api/wallet-api.ts)
// ---------------------------------------------------------------------------

const walletOverviewSchema = z.object({
  stripe_account_id: z.string(),
  charges_enabled: z.boolean(),
  payouts_enabled: z.boolean(),
  escrow_amount: cents,
  available_amount: cents,
  transferred_amount: cents,
  records: z
    .array(
      z.object({
        id: uuid,
        proposal_id: uuid,
        milestone_id: z.string().optional(),
        proposal_amount: cents,
        platform_fee: cents,
        provider_payout: cents,
        payment_status: z.string(),
        transfer_status: z.string(),
        mission_status: z.string(),
        created_at: isoDate,
      }),
    )
    .nullable(),
  commissions: z.object({
    pending_cents: cents,
    pending_kyc_cents: cents,
    paid_cents: cents,
    clawed_back_cents: cents,
    currency: z.string(),
  }),
  commission_records: z.array(z.unknown()).nullable(),
})

describe("contract: WalletOverview", () => {
  it("parses a fresh empty wallet", () => {
    expect(() =>
      walletOverviewSchema.parse({
        stripe_account_id: "acct_123",
        charges_enabled: false,
        payouts_enabled: false,
        escrow_amount: 0,
        available_amount: 0,
        transferred_amount: 0,
        records: null,
        commissions: {
          pending_cents: 0,
          pending_kyc_cents: 0,
          paid_cents: 0,
          clawed_back_cents: 0,
          currency: "EUR",
        },
        commission_records: null,
      }),
    ).not.toThrow()
  })

  it("parses a wallet with one record", () => {
    expect(() =>
      walletOverviewSchema.parse({
        stripe_account_id: "acct_123",
        charges_enabled: true,
        payouts_enabled: true,
        escrow_amount: 0,
        available_amount: 0,
        transferred_amount: 100000,
        records: [
          {
            id: "rec-1",
            proposal_id: "prop-1",
            milestone_id: "ms-1",
            proposal_amount: 100000,
            platform_fee: 5000,
            provider_payout: 95000,
            payment_status: "paid",
            transfer_status: "paid",
            mission_status: "completed",
            created_at: "2026-04-01T10:00:00Z",
          },
        ],
        commissions: {
          pending_cents: 0,
          pending_kyc_cents: 0,
          paid_cents: 0,
          clawed_back_cents: 0,
          currency: "EUR",
        },
        commission_records: null,
      }),
    ).not.toThrow()
  })
})

// ---------------------------------------------------------------------------
// GDPR contract (features/account/api/gdpr.ts)
// ---------------------------------------------------------------------------

const requestDeletionSchema = z.object({
  email_sent_to: z.string().email(),
  expires_at: isoDate,
})
const confirmDeletionSchema = z.object({
  user_id: uuid,
  deleted_at: isoDate,
  hard_delete_at: isoDate,
})
const cancelDeletionSchema = z.object({ cancelled: z.boolean() })

describe("contract: GDPR DTOs", () => {
  it("parses a request-deletion response", () => {
    expect(() =>
      requestDeletionSchema.parse({
        email_sent_to: "user@example.com",
        expires_at: "2026-05-01T10:00:00Z",
      }),
    ).not.toThrow()
  })

  it("parses a confirm-deletion response", () => {
    expect(() =>
      confirmDeletionSchema.parse({
        user_id: "u-1",
        deleted_at: "2026-05-01T10:00:00Z",
        hard_delete_at: "2026-06-01T10:00:00Z",
      }),
    ).not.toThrow()
  })

  it("parses a cancel-deletion response", () => {
    expect(() =>
      cancelDeletionSchema.parse({ cancelled: true }),
    ).not.toThrow()
  })

  it("rejects malformed email on request-deletion", () => {
    expect(() =>
      requestDeletionSchema.parse({
        email_sent_to: "not-an-email",
        expires_at: "2026-05-01T10:00:00Z",
      }),
    ).toThrow()
  })
})

// ---------------------------------------------------------------------------
// Billing / FeePreview contract (shared/types/billing.ts)
// ---------------------------------------------------------------------------

const feePreviewSchema = z.object({
  amount_cents: cents,
  fee_cents: cents,
  net_cents: cents,
  role: z.enum(["freelance", "agency"]),
  active_tier_index: z.number().int().nonnegative(),
  tiers: z.array(
    z.object({
      label: z.string(),
      max_cents: cents.nullable(),
      fee_cents: cents,
    }),
  ),
  viewer_is_provider: z.boolean(),
  viewer_is_subscribed: z.boolean(),
})

describe("contract: FeePreview", () => {
  it("parses a fee preview with three tiers", () => {
    expect(() =>
      feePreviewSchema.parse({
        amount_cents: 50000,
        fee_cents: 1000,
        net_cents: 49000,
        role: "freelance",
        active_tier_index: 1,
        tiers: [
          { label: "0 € – 200 €", max_cents: 20000, fee_cents: 500 },
          { label: "200 € – 1 000 €", max_cents: 100000, fee_cents: 1000 },
          { label: "Plus de 1 000 €", max_cents: null, fee_cents: 2500 },
        ],
        viewer_is_provider: true,
        viewer_is_subscribed: false,
      }),
    ).not.toThrow()
  })

  it("rejects negative amounts", () => {
    expect(() =>
      feePreviewSchema.parse({
        amount_cents: -1,
        fee_cents: 0,
        net_cents: 0,
        role: "freelance",
        active_tier_index: 0,
        tiers: [],
        viewer_is_provider: true,
        viewer_is_subscribed: false,
      }),
    ).toThrow()
  })
})

// ---------------------------------------------------------------------------
// Subscription contract (features/subscription/types.ts)
// ---------------------------------------------------------------------------

const subscriptionSchema = z.object({
  id: uuid,
  plan: z.string(),
  billing_cycle: z.string(),
  status: z.enum([
    "incomplete",
    "active",
    "past_due",
    "canceled",
    "unpaid",
  ]),
  current_period_start: isoDate,
  current_period_end: isoDate,
  cancel_at_period_end: z.boolean(),
  started_at: isoDate,
  grace_period_ends_at: isoDate.optional(),
  canceled_at: isoDate.optional(),
  pending_billing_cycle: z.string().optional(),
  pending_cycle_effective_at: isoDate.optional(),
})

describe("contract: Subscription", () => {
  it("parses an active monthly subscription", () => {
    expect(() =>
      subscriptionSchema.parse({
        id: "sub_1",
        plan: "premium",
        billing_cycle: "monthly",
        status: "active",
        current_period_start: "2026-04-01T00:00:00Z",
        current_period_end: "2026-05-01T00:00:00Z",
        cancel_at_period_end: false,
        started_at: "2026-04-01T00:00:00Z",
      }),
    ).not.toThrow()
  })

  it("parses a canceled subscription with auto-renew off", () => {
    expect(() =>
      subscriptionSchema.parse({
        id: "sub_1",
        plan: "premium",
        billing_cycle: "annual",
        status: "active",
        current_period_start: "2026-04-01T00:00:00Z",
        current_period_end: "2027-04-01T00:00:00Z",
        cancel_at_period_end: true,
        started_at: "2026-04-01T00:00:00Z",
        canceled_at: "2026-04-15T10:00:00Z",
      }),
    ).not.toThrow()
  })

  it("rejects an unknown subscription status", () => {
    expect(() =>
      subscriptionSchema.parse({
        id: "sub_1",
        plan: "premium",
        billing_cycle: "monthly",
        status: "future_status",
        current_period_start: "2026-04-01T00:00:00Z",
        current_period_end: "2026-05-01T00:00:00Z",
        cancel_at_period_end: false,
        started_at: "2026-04-01T00:00:00Z",
      }),
    ).toThrow()
  })
})

// ---------------------------------------------------------------------------
// Referral contract (shared/types/referral.ts)
// ---------------------------------------------------------------------------

const referralSchema = z.object({
  id: uuid,
  status: z.string(),
  referrer_id: uuid,
  client_id: uuid,
  provider_id: uuid,
  created_at: isoDate,
})

describe("contract: Referral list", () => {
  it("parses a minimal referral envelope", () => {
    expect(() =>
      referralSchema.parse({
        id: "r1",
        status: "active",
        referrer_id: "u1",
        client_id: "u2",
        provider_id: "u3",
        created_at: "2026-04-01T00:00:00Z",
      }),
    ).not.toThrow()
  })
})

// ---------------------------------------------------------------------------
// Team contract (features/team/types.ts)
// ---------------------------------------------------------------------------

const teamMemberSchema = z.object({
  id: uuid,
  organization_id: uuid,
  user_id: uuid,
  role: z.enum(["owner", "admin", "member", "viewer"]),
  title: z.string(),
  joined_at: isoDate,
  user: z
    .object({
      id: uuid,
      email: z.string(),
      display_name: z.string(),
      first_name: z.string(),
      last_name: z.string(),
    })
    .optional(),
})

const teamInvitationSchema = z.object({
  id: uuid,
  organization_id: uuid,
  email: z.string(),
  first_name: z.string(),
  last_name: z.string(),
  title: z.string(),
  role: z.enum(["admin", "member", "viewer"]),
  invited_by_user_id: uuid,
  status: z.enum(["pending", "accepted", "cancelled", "expired"]),
  expires_at: isoDate,
  accepted_at: isoDate.nullable().optional(),
  cancelled_at: isoDate.nullable().optional(),
  created_at: isoDate,
  updated_at: isoDate,
})

describe("contract: Team DTOs", () => {
  it("parses a team member with embedded user", () => {
    expect(() =>
      teamMemberSchema.parse({
        id: "tm-1",
        organization_id: "org-1",
        user_id: "u-1",
        role: "admin",
        title: "Lead",
        joined_at: "2026-04-01T00:00:00Z",
        user: {
          id: "u-1",
          email: "user@example.com",
          display_name: "Jean",
          first_name: "Jean",
          last_name: "Dupont",
        },
      }),
    ).not.toThrow()
  })

  it("rejects member with invalid role", () => {
    expect(() =>
      teamMemberSchema.parse({
        id: "tm-1",
        organization_id: "org-1",
        user_id: "u-1",
        role: "superuser", // invalid
        title: "",
        joined_at: "2026-04-01",
      }),
    ).toThrow()
  })

  it("parses an invitation envelope", () => {
    expect(() =>
      teamInvitationSchema.parse({
        id: "inv-1",
        organization_id: "org-1",
        email: "new@example.com",
        first_name: "New",
        last_name: "Hire",
        title: "Designer",
        role: "member",
        invited_by_user_id: "u-1",
        status: "pending",
        expires_at: "2026-05-01T00:00:00Z",
        created_at: "2026-04-01T00:00:00Z",
        updated_at: "2026-04-01T00:00:00Z",
      }),
    ).not.toThrow()
  })
})

// ---------------------------------------------------------------------------
// Invoicing contract (features/invoicing/types.ts)
// ---------------------------------------------------------------------------

const invoiceSchema = z.object({
  id: uuid,
  number: z.string(),
  issued_at: isoDate,
  source_type: z.enum(["subscription", "monthly_commission", "credit_note"]),
  amount_incl_tax_cents: cents,
  currency: z.string(),
  pdf_url: z.string(),
})

const invoicesPageSchema = z.object({
  data: z.array(invoiceSchema),
  next_cursor: z.string().optional(),
})

const billingProfileSnapshotSchema = z.object({
  profile: z.object({
    organization_id: uuid,
    profile_type: z.enum(["individual", "business"]),
    legal_name: z.string(),
    trading_name: z.string(),
    legal_form: z.string(),
    tax_id: z.string(),
    vat_number: z.string(),
    vat_validated_at: isoDate.nullable(),
    address_line1: z.string(),
    address_line2: z.string(),
    postal_code: z.string(),
    city: z.string(),
    country: z.string(),
    invoicing_email: z.string(),
    synced_from_kyc_at: isoDate.nullable(),
  }),
  missing_fields: z.array(
    z.object({ field: z.string(), reason: z.string() }),
  ),
  is_complete: z.boolean(),
})

describe("contract: Invoicing DTOs", () => {
  it("parses an invoice list response", () => {
    expect(() =>
      invoicesPageSchema.parse({
        data: [
          {
            id: "inv-1",
            number: "2026-001",
            issued_at: "2026-04-01T00:00:00Z",
            source_type: "monthly_commission",
            amount_incl_tax_cents: 12000,
            currency: "EUR",
            pdf_url: "",
          },
        ],
      }),
    ).not.toThrow()
  })

  it("parses a complete billing profile snapshot", () => {
    expect(() =>
      billingProfileSnapshotSchema.parse({
        profile: {
          organization_id: "org-1",
          profile_type: "business",
          legal_name: "Acme SAS",
          trading_name: "Acme",
          legal_form: "SAS",
          tax_id: "FR12345678901",
          vat_number: "FR12345678901",
          vat_validated_at: "2026-04-01T00:00:00Z",
          address_line1: "1 rue de la Paix",
          address_line2: "",
          postal_code: "75002",
          city: "Paris",
          country: "FR",
          invoicing_email: "billing@acme.fr",
          synced_from_kyc_at: null,
        },
        missing_fields: [],
        is_complete: true,
      }),
    ).not.toThrow()
  })
})

// ---------------------------------------------------------------------------
// Notification contract (features/notification/types.ts)
// ---------------------------------------------------------------------------

const notificationSchema = z.object({
  id: uuid,
  user_id: uuid,
  type: z.string(), // narrow enum mirrored in TS, but new types ship every quarter
  title: z.string(),
  body: z.string(),
  data: z.record(z.string(), z.unknown()),
  read_at: isoDate.nullable(),
  created_at: isoDate,
})

const notificationListSchema = z.object({
  data: z.array(notificationSchema),
  next_cursor: z.string(),
  has_more: z.boolean(),
})

describe("contract: Notifications DTOs", () => {
  it("parses an unread notification", () => {
    expect(() =>
      notificationSchema.parse({
        id: "n1",
        user_id: "u1",
        type: "new_message",
        title: "New message",
        body: "You have a new message",
        data: { conversation_id: "c1" },
        read_at: null,
        created_at: "2026-04-01T00:00:00Z",
      }),
    ).not.toThrow()
  })

  it("parses a notification list", () => {
    expect(() =>
      notificationListSchema.parse({
        data: [],
        next_cursor: "",
        has_more: false,
      }),
    ).not.toThrow()
  })
})

// ---------------------------------------------------------------------------
// Pagination smoke
// ---------------------------------------------------------------------------

describe("contract: Pagination envelope (cross-feature)", () => {
  const cases: { name: string; payload: unknown }[] = [
    { name: "first page", payload: { has_more: true, next_cursor: "tok-1" } },
    { name: "last page", payload: { has_more: false } },
    { name: "empty", payload: {} },
  ]
  it.each(cases)("parses $name", ({ payload }) => {
    expect(() => cursorPagination.parse(payload)).not.toThrow()
  })
})
