import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"

// Mock the proposal API
const mockCreateProposal = vi.fn()
const mockAcceptProposal = vi.fn()
const mockDeclineProposal = vi.fn()
const mockModifyProposal = vi.fn()
const mockSimulatePayment = vi.fn()
const mockRequestCompletion = vi.fn()
const mockCompleteProposal = vi.fn()
const mockRejectCompletion = vi.fn()
const mockListProjects = vi.fn()

vi.mock("../../api/proposal-api", () => ({
  createProposal: (...args: unknown[]) => mockCreateProposal(...args),
  acceptProposal: (...args: unknown[]) => mockAcceptProposal(...args),
  declineProposal: (...args: unknown[]) => mockDeclineProposal(...args),
  modifyProposal: (...args: unknown[]) => mockModifyProposal(...args),
  simulatePayment: (...args: unknown[]) => mockSimulatePayment(...args),
  requestCompletion: (...args: unknown[]) => mockRequestCompletion(...args),
  completeProposal: (...args: unknown[]) => mockCompleteProposal(...args),
  rejectCompletion: (...args: unknown[]) => mockRejectCompletion(...args),
  listProjects: (...args: unknown[]) => mockListProjects(...args),
}))

// Mock the conversations and messages query key exports
vi.mock("@/features/messaging/hooks/use-conversations", () => ({
  CONVERSATIONS_QUERY_KEY: ["messaging", "conversations"],
}))
vi.mock("@/features/messaging/hooks/use-messages", () => ({
  MESSAGES_QUERY_KEY: "messaging-messages",
}))

import {
  useCreateProposal,
  useAcceptProposal,
  useDeclineProposal,
  useModifyProposal,
  useSimulatePayment,
  useRequestCompletion,
  useCompleteProposal,
  useRejectCompletion,
  useProjects,
  PROJECTS_QUERY_KEY,
} from "../use-proposals"

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })

  return function Wrapper({ children }: { children: React.ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children)
  }
}

describe("useProjects", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it("returns project data from the API", async () => {
    const mockData = {
      data: [
        {
          id: "proposal-1",
          title: "Website redesign",
          amount: 500000,
          status: "active",
        },
      ],
      next_cursor: "",
      has_more: false,
    }
    mockListProjects.mockResolvedValueOnce(mockData)

    const { result } = renderHook(() => useProjects(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data).toEqual(mockData)
    expect(mockListProjects).toHaveBeenCalledWith(undefined)
  })

  it("passes cursor to the API", async () => {
    mockListProjects.mockResolvedValueOnce({
      data: [],
      next_cursor: "",
      has_more: false,
    })

    const { result } = renderHook(() => useProjects("cursor-abc"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(mockListProjects).toHaveBeenCalledWith("cursor-abc")
  })

  it("handles API errors", async () => {
    mockListProjects.mockRejectedValueOnce(new Error("Network error"))

    const { result } = renderHook(() => useProjects(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })

    expect(result.current.error).toBeDefined()
  })
})

describe("useCreateProposal", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls createProposal with correct data", async () => {
    const mockResponse = { id: "proposal-new", title: "Test" }
    mockCreateProposal.mockResolvedValueOnce(mockResponse)

    const { result } = renderHook(() => useCreateProposal(), {
      wrapper: createWrapper(),
    })

    act(() => {
      result.current.mutate({
        recipient_id: "user-2",
        conversation_id: "conv-1",
        title: "Test proposal",
        description: "Test description",
        amount: 100000,
      })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(mockCreateProposal).toHaveBeenCalledWith({
      recipient_id: "user-2",
      conversation_id: "conv-1",
      title: "Test proposal",
      description: "Test description",
      amount: 100000,
    })
  })
})

describe("useAcceptProposal", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls acceptProposal with proposal ID", async () => {
    mockAcceptProposal.mockResolvedValueOnce(undefined)

    const { result } = renderHook(() => useAcceptProposal(), {
      wrapper: createWrapper(),
    })

    act(() => {
      result.current.mutate("proposal-1")
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(mockAcceptProposal).toHaveBeenCalledWith("proposal-1")
  })
})

describe("useDeclineProposal", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls declineProposal with proposal ID", async () => {
    mockDeclineProposal.mockResolvedValueOnce(undefined)

    const { result } = renderHook(() => useDeclineProposal(), {
      wrapper: createWrapper(),
    })

    act(() => {
      result.current.mutate("proposal-1")
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(mockDeclineProposal).toHaveBeenCalledWith("proposal-1")
  })
})

describe("useModifyProposal", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls modifyProposal with id and data", async () => {
    const mockResponse = { id: "proposal-modified", title: "Updated" }
    mockModifyProposal.mockResolvedValueOnce(mockResponse)

    const { result } = renderHook(() => useModifyProposal(), {
      wrapper: createWrapper(),
    })

    act(() => {
      result.current.mutate({
        id: "proposal-1",
        data: {
          title: "Updated proposal",
          description: "Updated description",
          amount: 200000,
        },
      })
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(mockModifyProposal).toHaveBeenCalledWith("proposal-1", {
      title: "Updated proposal",
      description: "Updated description",
      amount: 200000,
    })
  })
})

describe("useSimulatePayment", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls simulatePayment with proposal ID", async () => {
    mockSimulatePayment.mockResolvedValueOnce(undefined)

    const { result } = renderHook(() => useSimulatePayment(), {
      wrapper: createWrapper(),
    })

    act(() => {
      result.current.mutate("proposal-1")
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(mockSimulatePayment).toHaveBeenCalledWith("proposal-1")
  })
})

describe("useRequestCompletion", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls requestCompletion with proposal ID", async () => {
    mockRequestCompletion.mockResolvedValueOnce(undefined)

    const { result } = renderHook(() => useRequestCompletion(), {
      wrapper: createWrapper(),
    })

    act(() => {
      result.current.mutate("proposal-1")
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(mockRequestCompletion).toHaveBeenCalledWith("proposal-1")
  })
})

describe("useCompleteProposal", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls completeProposal with proposal ID", async () => {
    mockCompleteProposal.mockResolvedValueOnce(undefined)

    const { result } = renderHook(() => useCompleteProposal(), {
      wrapper: createWrapper(),
    })

    act(() => {
      result.current.mutate("proposal-1")
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(mockCompleteProposal).toHaveBeenCalledWith("proposal-1")
  })
})

describe("useRejectCompletion", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls rejectCompletion with proposal ID", async () => {
    mockRejectCompletion.mockResolvedValueOnce(undefined)

    const { result } = renderHook(() => useRejectCompletion(), {
      wrapper: createWrapper(),
    })

    act(() => {
      result.current.mutate("proposal-1")
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(mockRejectCompletion).toHaveBeenCalledWith("proposal-1")
  })
})

describe("PROJECTS_QUERY_KEY", () => {
  it("is defined as expected", () => {
    expect(PROJECTS_QUERY_KEY).toEqual(["projects"])
  })
})
