import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  uploadReferrerVideo,
  deleteReferrerVideo,
} from "../referrer-video-api"

interface FetchCall {
  url: string
  init?: RequestInit
}

const calls: FetchCall[] = []
const originalFetch = globalThis.fetch

beforeEach(() => {
  calls.length = 0
  globalThis.fetch = vi.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
    calls.push({ url: String(url), init })
    return new Response(
      JSON.stringify({ video_url: "https://storage.example.com/r.mp4" }),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )
  }) as typeof fetch
})

afterEach(() => {
  globalThis.fetch = originalFetch
})

describe("referrer-video-api", () => {
  it("uploadReferrerVideo POSTs to /api/v1/referrer-profile/video", async () => {
    const file = new File(["dummy"], "referrer.mp4", { type: "video/mp4" })
    const result = await uploadReferrerVideo(file)

    expect(calls).toHaveLength(1)
    expect(calls[0].url).toContain("/api/v1/referrer-profile/video")
    expect(calls[0].init?.method).toBe("POST")
    expect(calls[0].init?.credentials).toBe("include")
    expect(result.video_url).toBe("https://storage.example.com/r.mp4")
  })

  it("deleteReferrerVideo DELETEs /api/v1/referrer-profile/video", async () => {
    await deleteReferrerVideo()

    expect(calls).toHaveLength(1)
    expect(calls[0].url).toContain("/api/v1/referrer-profile/video")
    expect(calls[0].init?.method).toBe("DELETE")
    expect(calls[0].init?.credentials).toBe("include")
  })

  it("uploadReferrerVideo throws on non-OK response", async () => {
    globalThis.fetch = vi.fn(async () => {
      return new Response(JSON.stringify({ message: "boom" }), { status: 500 })
    }) as typeof fetch

    const file = new File(["dummy"], "referrer.mp4", { type: "video/mp4" })
    await expect(uploadReferrerVideo(file)).rejects.toThrow("boom")
  })
})
