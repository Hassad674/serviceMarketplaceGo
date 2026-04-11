// Team feature types. Mirror the backend DTOs produced by the
// organization handlers (Phase 1-3) and the shared session shape from
// useSession/useOrganization. Deliberately narrow — if a field exists
// on the backend but isn't consumed here, we don't import it.

export type OrgRole = "owner" | "admin" | "member" | "viewer"

export type TeamMember = {
  id: string
  organization_id: string
  user_id: string
  role: OrgRole
  title: string
  joined_at: string
  // The users table is joined server-side so the UI can render an
  // avatar + name without a second round-trip. Absent for legacy rows
  // before the backend added the join.
  user?: {
    id: string
    email: string
    display_name: string
    first_name: string
    last_name: string
  }
}

export type TeamInvitationStatus = "pending" | "accepted" | "cancelled" | "expired"

export type TeamInvitation = {
  id: string
  organization_id: string
  email: string
  first_name: string
  last_name: string
  title: string
  role: "admin" | "member" | "viewer"
  invited_by_user_id: string
  status: TeamInvitationStatus
  expires_at: string
  accepted_at?: string | null
  cancelled_at?: string | null
  created_at: string
  updated_at: string
}

// API envelopes — the backend wraps list endpoints with `data` +
// cursor pagination metadata. For the team lists we only care about
// the items themselves (org size is capped at ~100 in V1).
export type TeamMembersListResponse = {
  data: TeamMember[]
  next_cursor?: string
}

export type TeamInvitationsListResponse = {
  data: TeamInvitation[]
  next_cursor?: string
}

// Mutation payloads — shape matches the request DTOs on the backend.
export type SendInvitationPayload = {
  email: string
  first_name: string
  last_name: string
  title: string
  role: "admin" | "member" | "viewer"
}

export type UpdateMemberPayload = {
  role?: OrgRole
  title?: string
}

export type InitiateTransferPayload = {
  target_user_id: string
}

// Public invitation preview returned by GET /invitations/validate.
// Used by the email-link landing page to show the invitee who is
// inviting them, into which org, and as what role before they set
// a password. Does not include the token itself — the page has it
// in the URL.
export type InvitationPreview = {
  id: string
  organization_id: string
  organization_name: string
  organization_type: "agency" | "enterprise"
  email: string
  first_name: string
  last_name: string
  title: string
  role: "owner" | "admin" | "member" | "viewer"
  expires_at: string
}

export type AcceptInvitationPayload = {
  token: string
  password: string
}
