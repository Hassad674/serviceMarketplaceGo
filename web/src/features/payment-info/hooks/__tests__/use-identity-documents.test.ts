import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import {
  useIdentityDocuments,
  useUploadIdentityDocument,
  useDeleteIdentityDocument,
} from "../use-identity-documents"

const mockListIdentityDocuments = vi.fn()
const mockUploadIdentityDocument = vi.fn()
const mockDeleteIdentityDocument = vi.fn()

vi.mock("../../api/identity-document-api", () => ({
  listIdentityDocuments: (...args: unknown[]) =>
    mockListIdentityDocuments(...args),
  uploadIdentityDocument: (...args: unknown[]) =>
    mockUploadIdentityDocument(...args),
  deleteIdentityDocument: (...args: unknown[]) =>
    mockDeleteIdentityDocument(...args),
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
}

describe("useIdentityDocuments", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("fetches identity documents on mount", async () => {
    const docs = [
      {
        id: "doc-1",
        category: "identity",
        document_type: "passport",
        side: "single",
        status: "pending",
      },
    ]
    mockListIdentityDocuments.mockResolvedValue(docs)

    const { result } = renderHook(() => useIdentityDocuments(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(docs)
    expect(mockListIdentityDocuments).toHaveBeenCalledOnce()
  })

  it("returns empty array when no documents", async () => {
    mockListIdentityDocuments.mockResolvedValue([])

    const { result } = renderHook(() => useIdentityDocuments(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual([])
  })

  it("handles fetch error", async () => {
    mockListIdentityDocuments.mockRejectedValue(new Error("Network error"))

    const { result } = renderHook(() => useIdentityDocuments(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})

describe("useUploadIdentityDocument", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls upload with correct params", async () => {
    const uploadedDoc = {
      id: "doc-new",
      category: "identity",
      document_type: "id_card",
      side: "front",
      status: "pending",
    }
    mockUploadIdentityDocument.mockResolvedValue(uploadedDoc)

    const { result } = renderHook(() => useUploadIdentityDocument(), {
      wrapper: createWrapper(),
    })

    const file = new File(["content"], "id.jpg", { type: "image/jpeg" })

    await act(async () => {
      result.current.mutate({
        file,
        category: "identity",
        documentType: "id_card",
        side: "front",
      })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockUploadIdentityDocument).toHaveBeenCalledWith(
      file,
      "identity",
      "id_card",
      "front",
    )
  })

  it("handles upload failure", async () => {
    mockUploadIdentityDocument.mockRejectedValue(new Error("Upload failed"))

    const { result } = renderHook(() => useUploadIdentityDocument(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({
        file: new File([""], "f.jpg"),
        category: "identity",
        documentType: "passport",
        side: "single",
      })
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})

describe("useDeleteIdentityDocument", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls delete with document id", async () => {
    mockDeleteIdentityDocument.mockResolvedValue(undefined)

    const { result } = renderHook(() => useDeleteIdentityDocument(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate("doc-1")
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockDeleteIdentityDocument).toHaveBeenCalledWith("doc-1")
  })

  it("handles delete failure", async () => {
    mockDeleteIdentityDocument.mockRejectedValue(new Error("Not found"))

    const { result } = renderHook(() => useDeleteIdentityDocument(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate("doc-999")
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})
