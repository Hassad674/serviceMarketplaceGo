import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useChangePassword } from "../use-change-password"

const changePasswordMock = vi.fn()

vi.mock("../../api/account-api", () => ({
  changePassword: (...args: unknown[]) => changePasswordMock(...args),
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  const Wrapper = ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
  Wrapper.displayName = "TestWrapper"
  return Wrapper
}

describe("useChangePassword", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls changePassword with the body and resolves with the response", async () => {
    changePasswordMock.mockResolvedValue({
      data: { ok: true },
      meta: { request_id: "rq-1" },
    })

    const { result } = renderHook(() => useChangePassword(), {
      wrapper: createWrapper(),
    })

    const body = {
      current_password: "OldPass1!aaa",
      new_password: "NewPass1!aaa",
    }

    await act(async () => {
      result.current.mutate(body)
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(changePasswordMock).toHaveBeenCalledWith(body)
    expect(result.current.data?.data.ok).toBe(true)
  })

  it("surfaces errors via the mutation error", async () => {
    changePasswordMock.mockRejectedValue(new Error("boom"))

    const { result } = renderHook(() => useChangePassword(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({
        current_password: "x",
        new_password: "Tooweak",
      })
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toBeInstanceOf(Error)
  })
})
