import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import messages from "@/../messages/en.json"
import { LoginForm } from "../login-form"

// Mock next-intl navigation (Link, useRouter)
const mockPush = vi.fn()
vi.mock("@i18n/navigation", () => ({
  Link: ({
    href,
    children,
    ...rest
  }: {
    href: string
    children: React.ReactNode
    className?: string
  }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
  useRouter: () => ({
    push: mockPush,
    replace: vi.fn(),
    back: vi.fn(),
    prefetch: vi.fn(),
  }),
}))

// Mock the login API function. We keep the real AuthApiError class so
// the component's `err instanceof AuthApiError` check keeps working
// when the mocked login throws.
vi.mock("@/features/auth/api/auth-api", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("@/features/auth/api/auth-api")>()
  return {
    ...actual,
    login: vi.fn(),
  }
})

import { login } from "@/features/auth/api/auth-api"
const mockLogin = vi.mocked(login)

// PERF-FIX-W-AUTH-ME-FANOUT: LoginForm now uses `useQueryClient()`
// from TanStack Query so it can `invalidateQueries(["session"])`
// after a successful login. The test render must therefore live
// inside a `QueryClientProvider`. A fresh client per test keeps
// state isolated.
function renderLoginForm() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <NextIntlClientProvider locale="en" messages={messages}>
        <LoginForm />
      </NextIntlClientProvider>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("LoginForm", () => {
  it("renders email and password inputs", () => {
    renderLoginForm()

    expect(screen.getByLabelText(messages.auth.email)).toBeInTheDocument()
    expect(screen.getByLabelText(messages.auth.password)).toBeInTheDocument()
  })

  it("renders sign in button", () => {
    renderLoginForm()

    expect(
      screen.getByRole("button", { name: messages.auth.loginTitle }),
    ).toBeInTheDocument()
  })

  it("renders forgot password link", () => {
    renderLoginForm()

    const link = screen.getByRole("link", {
      name: messages.auth.forgotPassword,
    })
    expect(link).toBeInTheDocument()
    expect(link).toHaveAttribute("href", "/forgot-password")
  })

  it("renders create account link", () => {
    renderLoginForm()

    const link = screen.getByRole("link", {
      name: messages.common.createAccount,
    })
    expect(link).toBeInTheDocument()
    expect(link).toHaveAttribute("href", "/register")
  })

  it("shows validation error for empty email submission", async () => {
    const user = userEvent.setup()
    renderLoginForm()

    // Fill password but leave email empty, then submit
    const passwordInput = screen.getByLabelText(messages.auth.password)
    await user.type(passwordInput, "Passw0rd!")

    const submitButton = screen.getByRole("button", {
      name: messages.auth.loginTitle,
    })
    await user.click(submitButton)

    // Zod validation: email is required
    await waitFor(() => {
      expect(screen.getByText(/invalid email/i)).toBeInTheDocument()
    })
  })

  it("shows validation error for short password", async () => {
    const user = userEvent.setup()
    renderLoginForm()

    const emailInput = screen.getByLabelText(messages.auth.email)
    const passwordInput = screen.getByLabelText(messages.auth.password)

    await user.type(emailInput, "test@example.com")
    await user.type(passwordInput, "short")

    const submitButton = screen.getByRole("button", {
      name: messages.auth.loginTitle,
    })
    await user.click(submitButton)

    await waitFor(() => {
      expect(
        screen.getByText(/password must contain at least 8 characters/i),
      ).toBeInTheDocument()
    })
  })

  it("toggles password visibility", async () => {
    const user = userEvent.setup()
    renderLoginForm()

    const passwordInput = screen.getByLabelText(messages.auth.password)
    expect(passwordInput).toHaveAttribute("type", "password")

    // Click the show password button
    const toggleButton = screen.getByRole("button", {
      name: messages.common.showPassword,
    })
    await user.click(toggleButton)

    expect(passwordInput).toHaveAttribute("type", "text")

    // Click the hide password button
    const hideButton = screen.getByRole("button", {
      name: messages.common.hidePassword,
    })
    await user.click(hideButton)

    expect(passwordInput).toHaveAttribute("type", "password")
  })

  it("shows loading state when submitting", async () => {
    // Make login hang (never resolves during this test)
    mockLogin.mockImplementation(
      () => new Promise<never>(() => {}),
    )

    const user = userEvent.setup()
    renderLoginForm()

    const emailInput = screen.getByLabelText(messages.auth.email)
    const passwordInput = screen.getByLabelText(messages.auth.password)

    await user.type(emailInput, "test@example.com")
    await user.type(passwordInput, "Passw0rd!")

    const submitButton = screen.getByRole("button", {
      name: messages.auth.loginTitle,
    })
    await user.click(submitButton)

    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: messages.auth.signingIn }),
      ).toBeInTheDocument()
    })
  })

  it("shows server error on failed login", async () => {
    mockLogin.mockRejectedValueOnce(new Error("Invalid credentials"))

    const user = userEvent.setup()
    renderLoginForm()

    const emailInput = screen.getByLabelText(messages.auth.email)
    const passwordInput = screen.getByLabelText(messages.auth.password)

    await user.type(emailInput, "test@example.com")
    await user.type(passwordInput, "Passw0rd!")

    const submitButton = screen.getByRole("button", {
      name: messages.auth.loginTitle,
    })
    await user.click(submitButton)

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(
        "Invalid credentials",
      )
    })
  })

  it("navigates to dashboard on successful login", async () => {
    mockLogin.mockResolvedValueOnce({
      id: "1",
      email: "test@example.com",
      first_name: "John",
      last_name: "Doe",
      display_name: "John Doe",
      role: "provider",
      referrer_enabled: false,
      email_verified: true,
      created_at: "2026-01-01",
    })

    const user = userEvent.setup()
    renderLoginForm()

    const emailInput = screen.getByLabelText(messages.auth.email)
    const passwordInput = screen.getByLabelText(messages.auth.password)

    await user.type(emailInput, "test@example.com")
    await user.type(passwordInput, "Passw0rd!")

    const submitButton = screen.getByRole("button", {
      name: messages.auth.loginTitle,
    })
    await user.click(submitButton)

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith("/dashboard")
    })
  })

  // PERF-FIX-W-AUTH-ME-FANOUT: the session hook uses
  // `retryOnMount: false` to prevent the 401 fan-out on public
  // pages. The trade-off is that a stale "logged out" verdict in
  // the cache survives the SPA navigation to /dashboard unless the
  // login flow explicitly invalidates ["session"]. This test pins
  // that contract — losing the invalidation would silently break
  // the login UX.
  it("invalidates the ['session'] query on successful login so the dashboard refetches /auth/me", async () => {
    mockLogin.mockResolvedValueOnce({
      id: "1",
      email: "test@example.com",
      first_name: "John",
      last_name: "Doe",
      display_name: "John Doe",
      role: "provider",
      referrer_enabled: false,
      email_verified: true,
      created_at: "2026-01-01",
    })

    // Build a custom render that lets us inspect the QueryClient.
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    // Pre-seed the cache with a stale 401 verdict — this is the
    // scenario the fix protects: the user landed on /login while
    // logged out, /auth/me 401'd, the cache holds an error state
    // with `data: undefined`. After login succeeds we expect the
    // form to invalidate the cache so the next /auth/me on
    // /dashboard fires for real.
    const sessionQuery = queryClient.getQueryCache().build(
      queryClient,
      { queryKey: ["session"] },
    )
    sessionQuery.setState({
      ...sessionQuery.state,
      status: "error",
      error: new Error("Not authenticated"),
      fetchStatus: "idle",
    })
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

    render(
      <QueryClientProvider client={queryClient}>
        <NextIntlClientProvider locale="en" messages={messages}>
          <LoginForm />
        </NextIntlClientProvider>
      </QueryClientProvider>,
    )

    const user = userEvent.setup()
    const emailInput = screen.getByLabelText(messages.auth.email)
    const passwordInput = screen.getByLabelText(messages.auth.password)
    await user.type(emailInput, "test@example.com")
    await user.type(passwordInput, "Passw0rd!")

    const submitButton = screen.getByRole("button", {
      name: messages.auth.loginTitle,
    })
    await user.click(submitButton)

    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["session"] })
    })
    // The navigation must happen AFTER the invalidation so the
    // post-redirect render sees a fresh fetch, not stale state.
    expect(mockPush).toHaveBeenCalledWith("/dashboard")
  })
})
