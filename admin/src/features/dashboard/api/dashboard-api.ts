import { adminApi } from "@/shared/lib/api-client"

export type RecentSignup = {
  id: string
  display_name: string
  email: string
  role: string
  created_at: string
}

export type DashboardStats = {
  total_users: number
  users_by_role: Record<string, number>
  active_users: number
  suspended_users: number
  banned_users: number
  total_proposals: number
  active_proposals: number
  total_jobs: number
  open_jobs: number
  recent_signups: RecentSignup[]
}

export function getDashboardStats(): Promise<DashboardStats> {
  return adminApi<DashboardStats>("/api/v1/admin/dashboard/stats")
}
