/**
 * DashboardShell tests — PERF-W-01.
 *
 * The shell is a thin composition layer; the heavy lifting lives in
 * Sidebar / Header / CallSlot / ChatWidget. Tests focus on the perf
 * contract:
 *   - the call event handler is registered via the WS bridge
 *   - children are rendered into the main content slot
 *   - the LiveKit-using runtime is NOT in the eager import graph
 */

import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"

vi.mock("../sidebar", () => ({
  Sidebar: () => <div data-testid="sidebar" />,
  SIDEBAR_STORAGE_KEY: "marketplace-sidebar-collapsed",
}))

vi.mock("../header", () => ({
  Header: () => <header data-testid="header" />,
}))

vi.mock("@/shared/hooks/use-user", () => ({
  useUser: () => ({ data: { id: "user-1", email: "user@example.com" }, isLoading: false }),
}))

vi.mock("@/shared/components/kyc-banner", () => ({
  KYCBanner: () => null,
}))

const registerCallEventHandler = vi.fn()
vi.mock("@/shared/hooks/use-global-ws", () => ({
  useGlobalWS: () => ({
    registerCallEventHandler,
    setMessagingPageActive: vi.fn(),
  }),
}))

vi.mock("@/shared/components/chat-widget/chat-widget", () => ({
  ChatWidget: () => <div data-testid="chat-widget" />,
}))

vi.mock("@/features/call/components/call-slot", () => ({
  CallSlot: ({
    children,
    registerCallEventHandler: register,
  }: {
    children?: React.ReactNode
    registerCallEventHandler: unknown
  }) => {
    // Surface the handler reference for assertions.
    ;(globalThis as Record<string, unknown>).__callSlotRegister = register
    return <div data-testid="call-slot">{children}</div>
  },
}))

import { DashboardShell } from "../dashboard-shell"

beforeEach(() => {
  registerCallEventHandler.mockReset()
  delete (globalThis as Record<string, unknown>).__callSlotRegister
})

describe("DashboardShell — PERF-W-01", () => {
  it("renders children inside the call slot", async () => {
    render(
      <DashboardShell>
        <span data-testid="page-content">page</span>
      </DashboardShell>,
    )

    await waitFor(() => {
      expect(screen.getByTestId("call-slot")).toBeInTheDocument()
    })
    expect(screen.getByTestId("page-content")).toBeInTheDocument()
  })

  it("forwards the WS register handler to CallSlot (no eager LiveKit)", async () => {
    render(
      <DashboardShell>
        <span>x</span>
      </DashboardShell>,
    )

    // The register reference handed to CallSlot must be the one
    // returned by useGlobalWS — proving the slot owns the lazy
    // registration rather than the shell calling useCall directly.
    await waitFor(() => {
      expect((globalThis as Record<string, unknown>).__callSlotRegister).toBe(
        registerCallEventHandler,
      )
    })
  })

  it("renders sidebar, header, and chat widget", async () => {
    render(
      <DashboardShell>
        <span>page</span>
      </DashboardShell>,
    )

    await waitFor(() => {
      expect(screen.getByTestId("chat-widget")).toBeInTheDocument()
    })
    expect(screen.getByTestId("sidebar")).toBeInTheDocument()
    expect(screen.getByTestId("header")).toBeInTheDocument()
  })

  it("does not import livekit-client eagerly (regression guard)", async () => {
    // Source-level check: the shell must not contain a static `from "livekit-client"`
    // import (the killer sequence that pulls 1.3 MB into the eager
    // dashboard bundle). Comments mentioning the package are fine.
    const fs = await import("node:fs")
    const pathMod = await import("node:path")
    const source = fs.readFileSync(
      pathMod.resolve(__dirname, "../dashboard-shell.tsx"),
      "utf-8",
    )
    // Strip line comments so a comment that mentions the package
    // (which is helpful for future maintainers) does not trip the
    // assertion.
    const codeOnly = source
      .replace(/\/\/.*$/gm, "")
      .replace(/\/\*[\s\S]*?\*\//g, "")
    expect(/from\s+['"]livekit-client['"]/.test(codeOnly)).toBe(false)
    expect(
      /from\s+['"]@\/features\/call\/hooks\/use-call['"]/.test(codeOnly),
    ).toBe(false)
  })
})
