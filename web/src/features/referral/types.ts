// Referral types live in `@/shared/types/referral` (P9 — the inline
// `ReferralSystemMessage` is shared with the messaging feature, so
// the types live in `shared/`). Re-exported here for back-compat.
export type {
  ReferralStatus,
  ReferralActorRole,
  ReferralNegotiationAction,
  IntroSnapshot,
  ProviderSnapshot,
  ClientSnapshot,
  Referral,
  ReferralListResponse,
  ReferralNegotiation,
  ReferralAttribution,
  ReferralCommission,
  SnapshotToggles,
  CreateReferralInput,
  RespondAction,
  RespondReferralInput,
  StatusTone,
} from "@/shared/types/referral"
export { statusTone, formatRatePct } from "@/shared/types/referral"
