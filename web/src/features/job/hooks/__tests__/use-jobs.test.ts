import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useMyJobs, useCreateJob, useCloseJob, jobsQueryKey } from "../use-jobs"

const mockListMyJobs = vi.fn()
const mockCreateJob = vi.fn()
const mockCloseJob = vi.fn()

vi.mock("../../api/job-api", () => ({
  listMyJobs: (...args: unknown[]) => mockListMyJobs(...args),
  createJob: (...args: unknown[]) => mockCreateJob(...args),
  closeJob: (...args: unknown[]) => mockCloseJob(...args),
}))

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "test-user-id",
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

describe("useMyJobs", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls listMyJobs API on mount", async () => {
    mockListMyJobs.mockResolvedValue({ data: [], next_cursor: "", has_more: false })

    const { result } = renderHook(() => useMyJobs(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockListMyJobs).toHaveBeenCalledOnce()
  })

  it("returns job data from API", async () => {
    mockListMyJobs.mockResolvedValue({
      data: [
        {
          id: "job-1",
          creator_id: "test-user-id",
          title: "Senior Dev Needed",
          description: "Looking for a senior dev",
          skills: ["go", "react"],
          applicant_type: "all",
          budget_type: "one_shot",
          min_budget: 5000,
          max_budget: 10000,
          status: "open",
          created_at: "2026-03-25T10:00:00Z",
          updated_at: "2026-03-25T10:00:00Z",
          is_indefinite: false,
          description_type: "text",
        },
      ],
      next_cursor: "",
      has_more: false,
    })

    const { result } = renderHook(() => useMyJobs(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.data).toHaveLength(1)
    expect(result.current.data?.data[0].title).toBe("Senior Dev Needed")
  })

  it("passes cursor to API", async () => {
    mockListMyJobs.mockResolvedValue({ data: [], next_cursor: "", has_more: false })

    const { result } = renderHook(() => useMyJobs("cursor-abc"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockListMyJobs).toHaveBeenCalledWith("cursor-abc")
  })

  it("builds user-scoped query key", () => {
    expect(jobsQueryKey("uid-123")).toEqual(["user", "uid-123", "jobs"])
  })
})

describe("useCreateJob", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls createJob API on mutate", async () => {
    mockCreateJob.mockResolvedValue({ id: "new-job-1", title: "New Job" })
    mockListMyJobs.mockResolvedValue({ data: [], next_cursor: "", has_more: false })

    const { result } = renderHook(() => useCreateJob(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({
        title: "New Job",
        description: "Description",
        skills: ["react"],
        applicant_type: "all",
        budget_type: "one_shot",
        min_budget: 1000,
        max_budget: 5000,
        is_indefinite: false,
        description_type: "text",
      })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockCreateJob).toHaveBeenCalledOnce()
    expect(mockCreateJob).toHaveBeenCalledWith(
      expect.objectContaining({ title: "New Job" }),
    )
  })
})

describe("useCloseJob", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls closeJob API on mutate", async () => {
    mockCloseJob.mockResolvedValue(undefined)
    mockListMyJobs.mockResolvedValue({ data: [], next_cursor: "", has_more: false })

    const { result } = renderHook(() => useCloseJob(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate("job-1")
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockCloseJob).toHaveBeenCalledWith("job-1")
  })
})
