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

// Mock the login API and the verifyTwoFactor call. Real AuthApiError
// + isTwoFactorChallenge are kept so the component branches on them.
vi.mock("@/features/auth/api/auth-api", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("@/features/auth/api/auth-api")>()
  return {
    ...actual,
    login: vi.fn(),
  }
})

vi.mock("@/features/auth/api/two-factor-api", () => ({
  verifyTwoFactor: vi.fn(),
}))

import { login } from "@/features/auth/api/auth-api"
import { verifyTwoFactor } from "@/features/auth/api/two-factor-api"
const mockLogin = vi.mocked(login)
const mockVerify = vi.mocked(verifyTwoFactor)

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

describe("LoginForm — 2FA gating", () => {
  it("swaps to the verification UI when login responds with requires_2fa", async () => {
    mockLogin.mockResolvedValueOnce({
      requires_2fa: true,
      user_id: "user-123",
      challenge_id: "challenge-456",
    })

    const user = userEvent.setup()
    renderLoginForm()

    await user.type(
      screen.getByLabelText(messages.auth.email),
      "test@example.com",
    )
    await user.type(
      screen.getByLabelText(messages.auth.password),
      "Passw0rd!",
    )
    await user.click(
      screen.getByRole("button", { name: messages.auth.loginTitle }),
    )

    // The verification step takes over the form — we're still on the
    // same page, no router push has happened.
    await waitFor(() => {
      expect(
        screen.getByLabelText(messages.twoFactor.codeLabel),
      ).toBeInTheDocument()
    })
    expect(mockPush).not.toHaveBeenCalled()
  })

  it("calls verifyTwoFactor with the cached user_id + challenge_id then redirects", async () => {
    mockLogin.mockResolvedValueOnce({
      requires_2fa: true,
      user_id: "user-123",
      challenge_id: "challenge-456",
    })
    mockVerify.mockResolvedValueOnce(undefined)

    const user = userEvent.setup()
    renderLoginForm()

    await user.type(
      screen.getByLabelText(messages.auth.email),
      "test@example.com",
    )
    await user.type(
      screen.getByLabelText(messages.auth.password),
      "Passw0rd!",
    )
    await user.click(
      screen.getByRole("button", { name: messages.auth.loginTitle }),
    )

    const codeInput = await screen.findByLabelText(messages.twoFactor.codeLabel)
    await user.type(codeInput, "654321")
    await user.click(
      screen.getByRole("button", { name: messages.twoFactor.verifyCta }),
    )

    await waitFor(() => {
      expect(mockVerify).toHaveBeenCalledWith({
        user_id: "user-123",
        challenge_id: "challenge-456",
        code: "654321",
      })
    })
    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith("/dashboard")
    })
  })

  it("preserves the non-2FA happy path when the response is the regular AuthUser", async () => {
    mockLogin.mockResolvedValueOnce({
      id: "user-1",
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

    await user.type(
      screen.getByLabelText(messages.auth.email),
      "test@example.com",
    )
    await user.type(
      screen.getByLabelText(messages.auth.password),
      "Passw0rd!",
    )
    await user.click(
      screen.getByRole("button", { name: messages.auth.loginTitle }),
    )

    await waitFor(() => {
      expect(mockPush).toHaveBeenCalledWith("/dashboard")
    })
    expect(mockVerify).not.toHaveBeenCalled()
  })
})
