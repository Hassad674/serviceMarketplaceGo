import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { SecuritySettings } from "../security-settings"

// next-intl: passes the key through. Locale defaults to "fr".
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
  useLocale: () => "fr",
}))

// The SecuritySettings tab mounts the 2FA toggle, which reads the
// current user via `useUser`. Stub both to keep this suite scoped
// to the section composition.
vi.mock("@/shared/hooks/use-user", () => ({
  useUser: () => ({ data: { two_factor_email_enabled: false } }),
}))
vi.mock("../two-factor-toggle", () => ({
  TwoFactorToggle: () => <div data-testid="two-factor-toggle" />,
}))
// SessionsList drives a TanStack query; stub it with a marker so the
// composition test stays focused on what SecuritySettings renders,
// without booting a QueryClient just to assert tree shape.
vi.mock("../sessions-list", () => ({
  SessionsList: () => <div data-testid="sessions-list" />,
}))

describe("SecuritySettings", () => {
  it("renders the title, the 2FA toggle and the active-sessions panel", () => {
    render(<SecuritySettings />)
    expect(screen.getByText("title")).toBeInTheDocument()
    expect(screen.getByText("subtitle")).toBeInTheDocument()
    expect(screen.getByTestId("two-factor-toggle")).toBeInTheDocument()
    expect(screen.getByTestId("sessions-list")).toBeInTheDocument()
  })
})
