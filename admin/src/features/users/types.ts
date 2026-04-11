export type AdminUser = {
  id: string
  email: string
  first_name: string
  last_name: string
  display_name: string
  role: "agency" | "enterprise" | "provider"
  // Operators are users invited into an organization. They have no
  // independent marketplace profile and get deleted when removed.
  account_type: "marketplace_owner" | "operator"
  // Set for Owners of an agency/enterprise and for invited operators.
  // Null for solo providers and for users who do not belong to an org.
  organization_id?: string | null
  referrer_enabled: boolean
  is_admin: boolean
  status: "active" | "suspended" | "banned"
  suspended_at?: string
  suspension_reason?: string
  suspension_expires_at?: string
  banned_at?: string
  ban_reason?: string
  email_verified: boolean
  created_at: string
  updated_at: string
}

export type AdminUserListResponse = {
  data: AdminUser[]
  next_cursor: string
  has_more: boolean
  total: number
  page: number
  total_pages: number
}

export type AdminUserResponse = {
  data: AdminUser
}

export type UserFilters = {
  role: string
  status: string
  search: string
  page: number
  reported: boolean
}

// Phase 6 — Team management aggregates returned by GET
// /api/v1/admin/users/{id}/organization. They bundle everything the
// admin UI needs to render the team section of a user detail page
// in a single round-trip.

export type AdminOrganization = {
  id: string
  type: "agency" | "enterprise"
  owner_user_id: string
  pending_transfer_to_user_id?: string
  pending_transfer_initiated_at?: string
  pending_transfer_expires_at?: string
  created_at: string
  updated_at: string
}

export type AdminOrganizationMember = {
  id: string
  organization_id: string
  user_id: string
  role: "owner" | "admin" | "member" | "viewer"
  title: string
  joined_at: string
}

export type AdminOrganizationInvitation = {
  id: string
  organization_id: string
  email: string
  first_name: string
  last_name: string
  title: string
  role: "admin" | "member" | "viewer"
  invited_by_user_id: string
  status: "pending" | "accepted" | "cancelled" | "expired"
  expires_at: string
  accepted_at?: string
  cancelled_at?: string
  created_at: string
}

export type AdminOrganizationDetail = {
  organization: AdminOrganization
  members: AdminOrganizationMember[]
  pending_invitations: AdminOrganizationInvitation[]
  viewing_role: "owner" | "admin" | "member" | "viewer"
}
