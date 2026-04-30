import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  deleteReferrerVideo,
  deleteVideo,
  uploadPhoto,
  uploadReferrerVideo,
  uploadVideo,
} from "../upload-api"

// Mock global fetch — the upload helpers call window.fetch directly
// (intentionally bypassing the typed apiClient because the backend
// expects multipart/form-data, not JSON).
type FetchArgs = [RequestInfo | URL, RequestInit | undefined]

describe("shared/lib/upload-api", () => {
  let fetchMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    fetchMock = vi.fn()
    vi.stubGlobal("fetch", fetchMock)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  // ---------------------------------------------------------------------
  // uploadFile shared behaviour exercised through the public helpers
  // ---------------------------------------------------------------------

  describe("uploadPhoto", () => {
    it("posts the file as multipart/form-data with credentials and returns the parsed response", async () => {
      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ url: "https://cdn/test.jpg" }),
      })

      const file = new File(["x"], "avatar.png", { type: "image/png" })
      const result = await uploadPhoto(file)

      expect(result).toEqual({ url: "https://cdn/test.jpg" })

      const [url, init] = fetchMock.mock.calls[0] as FetchArgs
      expect(String(url)).toContain("/api/v1/upload/photo")
      expect(init?.method).toBe("POST")
      expect(init?.credentials).toBe("include")
      expect(init?.body).toBeInstanceOf(FormData)

      const form = init!.body as FormData
      expect(form.get("file")).toBe(file)
    })

    it("throws an Error with the backend-provided message when the server replies with !ok", async () => {
      fetchMock.mockResolvedValueOnce({
        ok: false,
        json: async () => ({ message: "File too large" }),
      })

      await expect(
        uploadPhoto(new File(["x"], "big.png", { type: "image/png" })),
      ).rejects.toThrow("File too large")
    })

    it("throws a generic Error when the error body cannot be parsed as JSON", async () => {
      fetchMock.mockResolvedValueOnce({
        ok: false,
        // simulate JSON parse failure
        json: async () => {
          throw new Error("invalid json")
        },
      })

      await expect(
        uploadPhoto(new File(["x"], "x.png", { type: "image/png" })),
      ).rejects.toThrow("Upload failed")
    })

    it("falls back to the generic message when the parsed body has no message field", async () => {
      fetchMock.mockResolvedValueOnce({
        ok: false,
        json: async () => ({}),
      })

      await expect(
        uploadPhoto(new File(["x"], "x.png", { type: "image/png" })),
      ).rejects.toThrow("Upload failed")
    })
  })

  describe("uploadVideo", () => {
    it("targets the video endpoint and forwards the file payload", async () => {
      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ url: "https://cdn/intro.mp4" }),
      })

      const file = new File(["v"], "intro.mp4", { type: "video/mp4" })
      const result = await uploadVideo(file)

      expect(result.url).toBe("https://cdn/intro.mp4")
      const [url] = fetchMock.mock.calls[0] as FetchArgs
      expect(String(url)).toContain("/api/v1/upload/video")
    })
  })

  describe("uploadReferrerVideo", () => {
    it("targets the referrer-specific video endpoint", async () => {
      fetchMock.mockResolvedValueOnce({
        ok: true,
        json: async () => ({ url: "https://cdn/ref.mp4" }),
      })

      const file = new File(["v"], "ref.mp4", { type: "video/mp4" })
      const result = await uploadReferrerVideo(file)

      expect(result.url).toBe("https://cdn/ref.mp4")
      const [url] = fetchMock.mock.calls[0] as FetchArgs
      expect(String(url)).toContain("/api/v1/upload/referrer-video")
    })
  })

  describe("deleteVideo", () => {
    it("issues a DELETE on /api/v1/upload/video with credentials and resolves on ok", async () => {
      fetchMock.mockResolvedValueOnce({ ok: true })

      await deleteVideo()

      const [url, init] = fetchMock.mock.calls[0] as FetchArgs
      expect(String(url)).toContain("/api/v1/upload/video")
      expect(init?.method).toBe("DELETE")
      expect(init?.credentials).toBe("include")
    })

    it("throws when the backend rejects the deletion", async () => {
      fetchMock.mockResolvedValueOnce({ ok: false })

      await expect(deleteVideo()).rejects.toThrow("Failed to delete video")
    })
  })

  describe("deleteReferrerVideo", () => {
    it("issues a DELETE on the referrer endpoint", async () => {
      fetchMock.mockResolvedValueOnce({ ok: true })

      await deleteReferrerVideo()

      const [url, init] = fetchMock.mock.calls[0] as FetchArgs
      expect(String(url)).toContain("/api/v1/upload/referrer-video")
      expect(init?.method).toBe("DELETE")
    })

    it("throws a referrer-specific error message on failure", async () => {
      fetchMock.mockResolvedValueOnce({ ok: false })

      await expect(deleteReferrerVideo()).rejects.toThrow(
        "Failed to delete referrer video",
      )
    })
  })
})
