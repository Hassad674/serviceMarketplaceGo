import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useChangeEmail } from "../use-change-email"

const changeEmailMock = vi.fn()

vi.mock("../../api/account-api", () => ({
  changeEmail: (...args: unknown[]) => changeEmailMock(...args),
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

describe("useChangeEmail", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls changeEmail with the body and resolves with the response", async () => {
    changeEmailMock.mockResolvedValue({
      data: { email: "new@example.com" },
      meta: { request_id: "rq-1" },
    })

    const { result } = renderHook(() => useChangeEmail(), {
      wrapper: createWrapper(),
    })

    const body = {
      current_password: "OldPass1!aaa",
      new_email: "new@example.com",
    }

    await act(async () => {
      result.current.mutate(body)
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(changeEmailMock).toHaveBeenCalledWith(body)
    expect(result.current.data?.data.email).toBe("new@example.com")
  })

  it("surfaces errors via the mutation error", async () => {
    changeEmailMock.mockRejectedValue(new Error("boom"))

    const { result } = renderHook(() => useChangeEmail(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({
        current_password: "x",
        new_email: "y@example.com",
      })
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error).toBeInstanceOf(Error)
  })
})
