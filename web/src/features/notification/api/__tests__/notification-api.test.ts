import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  listNotifications,
  getUnreadNotificationCount,
  markNotificationAsRead,
  markAllNotificationsAsRead,
  deleteNotification,
  getNotificationPreferences,
  updateNotificationPreferences,
  registerDeviceToken,
} from "../notification-api"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...a: unknown[]) => mockApiClient(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockApiClient.mockResolvedValue({})
})

describe("notification-api", () => {
  it("listNotifications GETs with limit=20 and no cursor", () => {
    listNotifications()
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/notifications?limit=20",
    )
  })

  it("listNotifications appends cursor", () => {
    listNotifications("tok")
    const call = mockApiClient.mock.calls[0][0] as string
    expect(call).toContain("cursor=tok")
    expect(call).toContain("limit=20")
  })

  it("getUnreadNotificationCount GETs /unread-count", () => {
    getUnreadNotificationCount()
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/notifications/unread-count",
    )
  })

  it("markNotificationAsRead POSTs /:id/read", () => {
    markNotificationAsRead("n-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/notifications/n-1/read",
      { method: "POST" },
    )
  })

  it("markAllNotificationsAsRead POSTs /read-all", () => {
    markAllNotificationsAsRead()
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/notifications/read-all",
      { method: "POST" },
    )
  })

  it("deleteNotification DELETEs by id", () => {
    deleteNotification("n-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/notifications/n-1",
      { method: "DELETE" },
    )
  })

  it("getNotificationPreferences GETs /preferences", () => {
    getNotificationPreferences()
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/notifications/preferences",
    )
  })

  it("updateNotificationPreferences PUTs the array wrapped in { preferences }", () => {
    const prefs = [
      { type: "new_message", in_app: true, email: false, push: true },
    ]
    updateNotificationPreferences(prefs)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/notifications/preferences",
      { method: "PUT", body: { preferences: prefs } },
    )
  })

  it("registerDeviceToken POSTs the token + platform", () => {
    registerDeviceToken("tok-abc", "ios")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/notifications/device-token",
      {
        method: "POST",
        body: { token: "tok-abc", platform: "ios" },
      },
    )
  })

  it("propagates errors", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("403"))
    await expect(getUnreadNotificationCount()).rejects.toThrow("403")
  })
})
