import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { SecuritySettings } from "../security-settings"

// next-intl: passes the key through. Locale defaults to "fr".
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
  useLocale: () => "fr",
}))

type QueryShape = {
  data?: { pages: { data: unknown[]; next_cursor?: string }[] }
  isLoading: boolean
  isError: boolean
  hasNextPage: boolean
  isFetchingNextPage: boolean
  fetchNextPage: () => void
  refetch: () => void
}

let queryShape: QueryShape

vi.mock("../../hooks/use-security-activity", () => ({
  useSecurityActivity: () => queryShape,
}))

// The SecuritySettings tab now mounts the 2FA toggle, which reads
// the current user via `useUser`. Stub it so this suite stays
// scoped to the activity feed it was originally written for.
vi.mock("@/shared/hooks/use-user", () => ({
  useUser: () => ({ data: { two_factor_email_enabled: false } }),
}))

// Replace the toggle with a plain marker so we don't pull the API
// hooks into a suite that mocks `next-intl` to a key passthrough.
vi.mock("../two-factor-toggle", () => ({
  TwoFactorToggle: () => <div data-testid="two-factor-toggle" />,
}))

const fetchNextPageMock = vi.fn()
const refetchMock = vi.fn()

beforeEach(() => {
  fetchNextPageMock.mockReset()
  refetchMock.mockReset()
  queryShape = {
    data: undefined,
    isLoading: false,
    isError: false,
    hasNextPage: false,
    isFetchingNextPage: false,
    fetchNextPage: fetchNextPageMock,
    refetch: refetchMock,
  }
})

describe("SecuritySettings", () => {
  it("renders the loading skeleton while fetching", () => {
    queryShape.isLoading = true
    render(<SecuritySettings />)
    expect(screen.getByText("title")).toBeInTheDocument()
    // 3 skeleton rows are aria-busy
    expect(document.querySelector('[aria-busy="true"]')).toBeInTheDocument()
  })

  it("renders the empty state when there are no events", () => {
    queryShape.data = { pages: [{ data: [], next_cursor: "" }] }
    render(<SecuritySettings />)
    expect(screen.getByText("empty")).toBeInTheDocument()
  })

  it("renders the error state and supports retry", () => {
    queryShape.isError = true
    render(<SecuritySettings />)
    expect(screen.getByRole("alert")).toBeInTheDocument()
    expect(screen.getByText("error")).toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: "retry" }))
    expect(refetchMock).toHaveBeenCalledTimes(1)
  })

  it("renders one row per event with device label and action", () => {
    queryShape.data = {
      pages: [
        {
          data: [
            {
              id: "evt-1",
              action: "auth.login_success",
              ip_address: "203.0.113.4",
              user_agent_summary: "Ordinateur (Chrome 120)",
              access_kind: "desktop",
              created_at: "2026-05-08T12:00:00Z",
            },
            {
              id: "evt-2",
              action: "auth.logout",
              ip_address: "198.51.100.7",
              user_agent_summary: "Mobile (Safari 16)",
              access_kind: "mobile",
              created_at: "2026-05-08T08:30:00Z",
            },
          ],
          next_cursor: "",
        },
      ],
    }
    const { container } = render(<SecuritySettings />)
    expect(screen.getByText("Ordinateur (Chrome 120)")).toBeInTheDocument()
    expect(screen.getByText("Mobile (Safari 16)")).toBeInTheDocument()
    expect(screen.getByText("203.0.113.4")).toBeInTheDocument()
    expect(screen.getByText("198.51.100.7")).toBeInTheDocument()
    // Action labels share their <p> with the IP separator, so look at
    // the rendered HTML to assert the label was emitted.
    expect(container.innerHTML).toContain("actions.loginSuccess")
    expect(container.innerHTML).toContain("actions.logout")
  })

  it("renders the load-more button when hasNextPage and triggers fetchNextPage", () => {
    queryShape.data = {
      pages: [
        {
          data: [
            {
              id: "evt-1",
              action: "auth.login_success",
              ip_address: "",
              user_agent_summary: "",
              access_kind: "unknown",
              created_at: "2026-05-08T12:00:00Z",
            },
          ],
          next_cursor: "next-token",
        },
      ],
    }
    queryShape.hasNextPage = true
    render(<SecuritySettings />)
    fireEvent.click(screen.getByRole("button", { name: "loadMore" }))
    expect(fetchNextPageMock).toHaveBeenCalledTimes(1)
  })

  it("falls back to 'unknownDevice' when the user_agent_summary is empty", () => {
    queryShape.data = {
      pages: [
        {
          data: [
            {
              id: "evt-1",
              action: "auth.login_success",
              ip_address: "",
              user_agent_summary: "",
              access_kind: "unknown",
              created_at: "2026-05-08T12:00:00Z",
            },
          ],
          next_cursor: "",
        },
      ],
    }
    render(<SecuritySettings />)
    expect(screen.getByText("unknownDevice")).toBeInTheDocument()
  })

  it("maps unrecognised actions to the generic label", () => {
    queryShape.data = {
      pages: [
        {
          data: [
            {
              id: "evt-1",
              action: "auth.something_new",
              ip_address: "",
              user_agent_summary: "Mobile (Safari 16)",
              access_kind: "mobile",
              created_at: "2026-05-08T12:00:00Z",
            },
          ],
          next_cursor: "",
        },
      ],
    }
    const { container } = render(<SecuritySettings />)
    expect(container.innerHTML).toContain("actions.unknown")
  })
})
