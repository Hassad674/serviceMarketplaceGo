/**
 * messaging-api.test.ts
 *
 * Unit tests for the messaging feature's HTTP wrapper. Each function
 * is asserted to hit the documented endpoint with the right method
 * and body envelope. Tests mock `apiClient` directly so we never round
 * trip to a real fetch — that surface is covered by the integration
 * tests in `src/shared/lib/__tests__/api-client.integration.test.ts`.
 */
import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  listMessages,
  sendMessage,
  startConversation,
  markAsRead,
  editMessage,
  deleteMessage,
  getPresignedURL,
  getUnreadCount,
  listConversations,
} from "../messaging-api"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...a: unknown[]) => mockApiClient(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockApiClient.mockResolvedValue({})
})

describe("messaging-api / listConversations", () => {
  it("calls /api/v1/messaging/conversations without a cursor", async () => {
    await listConversations()
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/messaging/conversations",
    )
  })

  it("URL-encodes the cursor", async () => {
    await listConversations("a/b")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/messaging/conversations?cursor=a%2Fb",
    )
  })

  it("propagates apiClient errors", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("boom"))
    await expect(listConversations()).rejects.toThrow("boom")
  })
})

describe("messaging-api / listMessages", () => {
  it("hits the per-conversation endpoint without cursor", async () => {
    await listMessages("c-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/messaging/conversations/c-1/messages",
    )
  })

  it("URL-encodes the cursor", async () => {
    await listMessages("c-1", "tok+1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/messaging/conversations/c-1/messages?cursor=tok%2B1",
    )
  })
})

describe("messaging-api / sendMessage", () => {
  it("POSTs a text message with default type", async () => {
    await sendMessage("c-1", "hello")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/messaging/conversations/c-1/messages",
      {
        method: "POST",
        body: {
          content: "hello",
          type: "text",
          metadata: undefined,
          reply_to_id: undefined,
        },
      },
    )
  })

  it("POSTs a file message with metadata", async () => {
    await sendMessage(
      "c-1",
      "spec.pdf",
      "file",
      { url: "u", filename: "spec.pdf", size: 100, mime_type: "application/pdf" },
    )
    const call = mockApiClient.mock.calls[0]
    expect(call[1].body.type).toBe("file")
    expect(call[1].body.metadata.filename).toBe("spec.pdf")
  })

  it("POSTs a voice message with duration metadata", async () => {
    await sendMessage(
      "c-1",
      "voice.webm",
      "voice",
      { url: "u", duration: 10, size: 100, mime_type: "audio/webm" },
    )
    const call = mockApiClient.mock.calls[0]
    expect(call[1].body.type).toBe("voice")
    expect(call[1].body.metadata.duration).toBe(10)
  })

  it("threads the reply_to_id when provided", async () => {
    await sendMessage("c-1", "hi", "text", undefined, "msg-parent")
    const call = mockApiClient.mock.calls[0]
    expect(call[1].body.reply_to_id).toBe("msg-parent")
  })
})

describe("messaging-api / startConversation", () => {
  it("POSTs the recipient_org_id and content", async () => {
    await startConversation("org-2", "hello")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/messaging/conversations",
      {
        method: "POST",
        body: { recipient_org_id: "org-2", content: "hello" },
      },
    )
  })
})

describe("messaging-api / markAsRead", () => {
  it("POSTs seq=0 by default", async () => {
    await markAsRead("c-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/messaging/conversations/c-1/read",
      { method: "POST", body: { seq: 0 } },
    )
  })

  it("POSTs the explicit seq when provided", async () => {
    await markAsRead("c-1", 42)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/messaging/conversations/c-1/read",
      { method: "POST", body: { seq: 42 } },
    )
  })
})

describe("messaging-api / editMessage", () => {
  it("PUTs the new content", async () => {
    await editMessage("m-1", "edited")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/messaging/messages/m-1",
      { method: "PUT", body: { content: "edited" } },
    )
  })
})

describe("messaging-api / deleteMessage", () => {
  it("DELETEs the message", async () => {
    await deleteMessage("m-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/messaging/messages/m-1",
      { method: "DELETE" },
    )
  })
})

describe("messaging-api / getPresignedURL", () => {
  it("POSTs filename and content_type", async () => {
    await getPresignedURL("doc.pdf", "application/pdf")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/messaging/upload-url",
      {
        method: "POST",
        body: { filename: "doc.pdf", content_type: "application/pdf" },
      },
    )
  })
})

describe("messaging-api / getUnreadCount", () => {
  it("hits /unread-count", async () => {
    await getUnreadCount()
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/messaging/unread-count",
    )
  })
})

describe("messaging-api / error propagation", () => {
  it("propagates errors from sendMessage", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("net"))
    await expect(sendMessage("c", "hi")).rejects.toThrow("net")
  })

  it("propagates errors from startConversation", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("net"))
    await expect(startConversation("o", "hi")).rejects.toThrow("net")
  })
})
