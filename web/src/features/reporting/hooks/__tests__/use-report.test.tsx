import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useCreateReport } from "../use-report"

const mockCreateReport = vi.fn()

vi.mock("../../api/reporting-api", () => ({
  createReport: (...a: unknown[]) => mockCreateReport(...a),
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  const wrapper = ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
  return { queryClient, wrapper }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("useCreateReport", () => {
  it("forwards the mutation to createReport", async () => {
    mockCreateReport.mockResolvedValue({ id: "r-1" })
    const { wrapper } = createWrapper()
    const { result } = renderHook(() => useCreateReport(), { wrapper })

    await act(async () => {
      await result.current.mutateAsync({
        target_type: "user",
        target_id: "u-1",
        conversation_id: "c-1",
        reason: "harassment",
        description: "x",
      })
    })

    expect(mockCreateReport).toHaveBeenCalledWith(
      {
        target_type: "user",
        target_id: "u-1",
        conversation_id: "c-1",
        reason: "harassment",
        description: "x",
      },
      expect.anything(),
    )
  })

  it("propagates errors", async () => {
    mockCreateReport.mockRejectedValue(new Error("fail"))
    const { wrapper } = createWrapper()
    const { result } = renderHook(() => useCreateReport(), { wrapper })

    await expect(
      result.current.mutateAsync({
        target_type: "user",
        target_id: "u-1",
        conversation_id: "c-1",
        reason: "spam",
        description: "",
      }),
    ).rejects.toThrow("fail")
  })
})
