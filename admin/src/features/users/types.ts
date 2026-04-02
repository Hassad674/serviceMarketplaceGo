export type AdminUser = {
  id: string
  email: string
  first_name: string
  last_name: string
  display_name: string
  role: "agency" | "enterprise" | "provider"
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
  meta: {
    request_id: string
    pagination: {
      next_cursor: string
      has_more: boolean
      count: number
      total: number
    }
  }
}

export type AdminUserResponse = {
  data: AdminUser
  meta: { request_id: string }
}

export type UserFilters = {
  role: string
  status: string
  search: string
  cursor: string
}
