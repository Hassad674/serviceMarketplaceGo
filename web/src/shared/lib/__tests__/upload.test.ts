/**
 * upload.test.ts
 *
 * Unit tests for the shared file-upload helper. The flow is
 *   1. POST to /api/v1/messaging/upload-url to get a presigned PUT URL
 *   2. PUT the file body to that URL
 *   3. Return a public URL + metadata envelope
 *
 * Tests mock both `apiClient` (presign step) and `fetch` (PUT step).
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { uploadFile, uploadFiles } from "../upload"

const mockApiClient = vi.fn()
const mockFetch = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...a: unknown[]) => mockApiClient(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockApiClient.mockResolvedValue({
    upload_url: "https://files/put?sig=abc",
    file_key: "uploads/abc.png",
    public_url: "https://files/uploads/abc.png",
  })
  vi.stubGlobal("fetch", mockFetch)
  mockFetch.mockResolvedValue({ ok: true, status: 200 })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("uploadFile", () => {
  it("requests a presigned URL with filename and content type", async () => {
    const file = new File(["bytes"], "screenshot.png", { type: "image/png" })
    await uploadFile(file)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/messaging/upload-url",
      {
        method: "POST",
        body: { filename: "screenshot.png", content_type: "image/png" },
      },
    )
  })

  it("PUTs the file body to the presigned URL with the right Content-Type", async () => {
    const file = new File(["bytes"], "doc.pdf", { type: "application/pdf" })
    await uploadFile(file)
    expect(mockFetch).toHaveBeenCalledWith(
      "https://files/put?sig=abc",
      expect.objectContaining({
        method: "PUT",
        body: file,
        headers: { "Content-Type": "application/pdf" },
      }),
    )
  })

  it("returns the public URL and file metadata", async () => {
    const file = new File(["bytes"], "doc.pdf", { type: "application/pdf" })
    const result = await uploadFile(file)
    expect(result).toEqual({
      filename: "doc.pdf",
      url: "https://files/uploads/abc.png",
      size: file.size,
      mime_type: "application/pdf",
    })
  })

  it("throws when the presign step fails (apiClient rejects)", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("presign failed"))
    const file = new File(["x"], "x.pdf", { type: "application/pdf" })
    await expect(uploadFile(file)).rejects.toThrow("presign failed")
    expect(mockFetch).not.toHaveBeenCalled()
  })

  it("throws when the PUT step returns a non-2xx status", async () => {
    mockFetch.mockResolvedValueOnce({ ok: false, status: 403 })
    const file = new File(["x"], "x.pdf", { type: "application/pdf" })
    await expect(uploadFile(file)).rejects.toThrow(/Upload failed: 403/)
  })
})

describe("uploadFiles", () => {
  it("uploads multiple files in parallel", async () => {
    const files = [
      new File(["a"], "a.pdf", { type: "application/pdf" }),
      new File(["b"], "b.pdf", { type: "application/pdf" }),
    ]
    const results = await uploadFiles(files)
    expect(results).toHaveLength(2)
    expect(mockApiClient).toHaveBeenCalledTimes(2)
    expect(mockFetch).toHaveBeenCalledTimes(2)
  })

  it("rejects the whole batch when one upload fails", async () => {
    mockFetch
      .mockResolvedValueOnce({ ok: true, status: 200 })
      .mockResolvedValueOnce({ ok: false, status: 500 })
    const files = [
      new File(["a"], "a.pdf", { type: "application/pdf" }),
      new File(["b"], "b.pdf", { type: "application/pdf" }),
    ]
    await expect(uploadFiles(files)).rejects.toThrow(/Upload failed: 500/)
  })

  it("returns an empty array for an empty input", async () => {
    const results = await uploadFiles([])
    expect(results).toEqual([])
    expect(mockApiClient).not.toHaveBeenCalled()
  })
})
