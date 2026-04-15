import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  uploadFreelanceVideo,
  deleteFreelanceVideo,
} from "../freelance-video-api"

// Verifies the freelance video api hits the per-persona endpoints
// added when the split-profiles refactor stranded the legacy upload
// path. Mocks `fetch` so the suite stays free of network IO.

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
      JSON.stringify({ video_url: "https://storage.example.com/v.mp4" }),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )
  }) as typeof fetch
})

afterEach(() => {
  globalThis.fetch = originalFetch
})

describe("freelance-video-api", () => {
  it("uploadFreelanceVideo POSTs to /api/v1/freelance-profile/video", async () => {
    const file = new File(["dummy"], "intro.mp4", { type: "video/mp4" })
    const result = await uploadFreelanceVideo(file)

    expect(calls).toHaveLength(1)
    expect(calls[0].url).toContain("/api/v1/freelance-profile/video")
    expect(calls[0].init?.method).toBe("POST")
    expect(calls[0].init?.credentials).toBe("include")
    expect(result.video_url).toBe("https://storage.example.com/v.mp4")
  })

  it("deleteFreelanceVideo DELETEs against /api/v1/freelance-profile/video", async () => {
    await deleteFreelanceVideo()

    expect(calls).toHaveLength(1)
    expect(calls[0].url).toContain("/api/v1/freelance-profile/video")
    expect(calls[0].init?.method).toBe("DELETE")
    expect(calls[0].init?.credentials).toBe("include")
  })

  it("uploadFreelanceVideo throws on non-OK response", async () => {
    globalThis.fetch = vi.fn(async () => {
      return new Response(JSON.stringify({ message: "boom" }), { status: 500 })
    }) as typeof fetch

    const file = new File(["dummy"], "intro.mp4", { type: "video/mp4" })
    await expect(uploadFreelanceVideo(file)).rejects.toThrow("boom")
  })
})
