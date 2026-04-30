import { describe, it, expect, vi, beforeEach } from "vitest"
import { getDashboardStats } from "../api/dashboard-api"

const mockAdminApi = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  adminApi: (...a: unknown[]) => mockAdminApi(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
})

describe("admin dashboard api / getDashboardStats", () => {
  it("GETs the stats endpoint", async () => {
    mockAdminApi.mockResolvedValue({
      total_users: 5,
      users_by_role: { agency: 2, enterprise: 1, provider: 2 },
      active_users: 4,
      suspended_users: 1,
      banned_users: 0,
      total_proposals: 0,
      active_proposals: 0,
      total_jobs: 0,
      open_jobs: 0,
      total_organizations: 3,
      pending_invitations: 1,
      recent_signups: [],
    })
    const stats = await getDashboardStats()
    expect(mockAdminApi).toHaveBeenCalledWith("/api/v1/admin/dashboard/stats")
    expect(stats.total_users).toBe(5)
    expect(stats.users_by_role.agency).toBe(2)
  })

  it("propagates errors", async () => {
    mockAdminApi.mockRejectedValueOnce(new Error("403"))
    await expect(getDashboardStats()).rejects.toThrow("403")
  })

  it("returns zero counts when backend hasn't shipped phase 6", async () => {
    mockAdminApi.mockResolvedValue({
      total_users: 0,
      users_by_role: {},
      active_users: 0,
      suspended_users: 0,
      banned_users: 0,
      total_proposals: 0,
      active_proposals: 0,
      total_jobs: 0,
      open_jobs: 0,
      total_organizations: 0,
      pending_invitations: 0,
      recent_signups: [],
    })
    const stats = await getDashboardStats()
    expect(stats.total_organizations).toBe(0)
  })
})
