/**
 * useReconcileCallOnMount tests.
 *
 * The hook reconciles orphan Redis state ("user is already in a call"
 * bug). The contract under test:
 *   - on mount, GET /api/v1/calls/me/active is called
 *   - non-null response triggers a Sonner toast with a hangup action
 *   - clicking the action fires endCall + clears the cache
 *   - null response keeps the toast UI silent
 *   - the toast is never shown twice for the same response
 *   - the hook honours `enabled=false` (no fetch fired)
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { render, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement, useState, type ReactNode } from "react"

// next-intl: pass keys through verbatim for predictable assertions.
vi.mock("next-intl", () => ({
  useTranslations: () => (k: string) => k,
}))

// sonner: capture toast() invocations so the test can drive the
// action click without rendering the real Toaster portal.
type ToastAction = { label: string; onClick: () => void }
type ToastOptions = {
  id?: string
  description?: string
  duration?: number
  action?: ToastAction
}
const toastSpy = vi.fn()
const toastDismissSpy = vi.fn()
vi.mock("sonner", () => ({
  toast: Object.assign(
    (...args: [string, ToastOptions?]) => toastSpy(...args),
    { dismiss: (id?: string) => toastDismissSpy(id) },
  ),
}))

// API mocks — the hook only touches getMyActiveCall + endCall.
const getMyActiveCallSpy = vi.fn()
const endCallSpy = vi.fn()
vi.mock("../../api/call-api", () => ({
  getMyActiveCall: (signal?: AbortSignal) => getMyActiveCallSpy(signal),
  endCall: (id: string, duration: number) => endCallSpy(id, duration),
}))

import { useReconcileCallOnMount } from "../use-reconcile-call-on-mount"

function makeWrapper(client: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client }, children)
  }
}

function makeClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0, staleTime: 0 },
    },
  })
}

function HookHarness({ enabled }: { enabled: boolean }) {
  useReconcileCallOnMount({ enabled })
  return null
}

beforeEach(() => {
  toastSpy.mockReset()
  toastDismissSpy.mockReset()
  getMyActiveCallSpy.mockReset()
  endCallSpy.mockReset()
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe("useReconcileCallOnMount", () => {
  it("does NOT fetch when disabled", async () => {
    getMyActiveCallSpy.mockResolvedValue(null)
    const client = makeClient()

    render(
      createElement(makeWrapper(client), null, createElement(HookHarness, { enabled: false })),
    )

    // Give React a tick so any spurious effect fires.
    await new Promise((r) => setTimeout(r, 0))
    expect(getMyActiveCallSpy).not.toHaveBeenCalled()
    expect(toastSpy).not.toHaveBeenCalled()
  })

  it("fetches on mount and stays silent when there is no active call", async () => {
    getMyActiveCallSpy.mockResolvedValue(null)
    const client = makeClient()

    render(
      createElement(makeWrapper(client), null, createElement(HookHarness, { enabled: true })),
    )

    await waitFor(() => {
      expect(getMyActiveCallSpy).toHaveBeenCalledTimes(1)
    })
    // Null payload must not surface any UI.
    expect(toastSpy).not.toHaveBeenCalled()
  })

  it("shows a toast with a hangup action when an orphan call is found", async () => {
    getMyActiveCallSpy.mockResolvedValue({
      call_id: "call-1",
      conversation_id: "conv-1",
      room_name: "call:call-1",
      type: "audio",
      status: "active",
      other_participant_id: "user-2",
    })
    const client = makeClient()

    render(
      createElement(makeWrapper(client), null, createElement(HookHarness, { enabled: true })),
    )

    await waitFor(() => {
      expect(toastSpy).toHaveBeenCalled()
    })

    const [title, options] = toastSpy.mock.calls[0] as [string, ToastOptions]
    expect(title).toBe("orphanCallTitle")
    expect(options.description).toBe("orphanCallDescription")
    expect(options.action?.label).toBe("hangup")
    expect(typeof options.action?.onClick).toBe("function")
  })

  it("clicking the toast action calls endCall with duration=0 and clears the toast", async () => {
    getMyActiveCallSpy.mockResolvedValue({
      call_id: "call-42",
      conversation_id: "conv-42",
      room_name: "call:call-42",
      type: "video",
      status: "active",
      other_participant_id: "user-99",
    })
    endCallSpy.mockResolvedValue(undefined)
    const client = makeClient()

    render(
      createElement(makeWrapper(client), null, createElement(HookHarness, { enabled: true })),
    )

    await waitFor(() => {
      expect(toastSpy).toHaveBeenCalled()
    })
    const action = (toastSpy.mock.calls[0] as [string, ToastOptions])[1]?.action
    expect(action).toBeDefined()

    await act(async () => {
      action!.onClick()
      // Allow the promise inside onClick to settle.
      await Promise.resolve()
      await Promise.resolve()
    })

    expect(endCallSpy).toHaveBeenCalledWith("call-42", 0)
    await waitFor(() => {
      expect(toastDismissSpy).toHaveBeenCalled()
    })
  })

  it("clicking the action still dismisses the toast even if endCall rejects", async () => {
    getMyActiveCallSpy.mockResolvedValue({
      call_id: "call-9",
      conversation_id: "conv-9",
      room_name: "call:call-9",
      type: "audio",
      status: "active",
      other_participant_id: "user-1",
    })
    endCallSpy.mockRejectedValue(new Error("already gone"))
    const client = makeClient()

    render(
      createElement(makeWrapper(client), null, createElement(HookHarness, { enabled: true })),
    )

    await waitFor(() => {
      expect(toastSpy).toHaveBeenCalled()
    })
    const action = (toastSpy.mock.calls[0] as [string, ToastOptions])[1]?.action

    await act(async () => {
      action!.onClick()
      await Promise.resolve()
      await Promise.resolve()
    })

    expect(endCallSpy).toHaveBeenCalledWith("call-9", 0)
    await waitFor(() => {
      expect(toastDismissSpy).toHaveBeenCalled()
    })
  })

  it("does NOT show the toast twice on re-renders of the same hook instance", async () => {
    getMyActiveCallSpy.mockResolvedValue({
      call_id: "call-1",
      conversation_id: "conv-1",
      room_name: "call:call-1",
      type: "audio",
      status: "active",
      other_participant_id: "user-2",
    })
    const client = makeClient()

    // Re-renderable harness that bumps a state to force the hook to
    // re-render WITHOUT being unmounted. The dismissed-ref guards
    // against a second toast for the same orphan payload.
    let trigger = (() => {}) as () => void
    function StableHarness() {
      const [, setBump] = useState(0)
      trigger = () => setBump((n) => n + 1)
      useReconcileCallOnMount({ enabled: true })
      return null
    }

    render(createElement(makeWrapper(client), null, createElement(StableHarness)))

    await waitFor(() => {
      expect(toastSpy).toHaveBeenCalledTimes(1)
    })

    await act(async () => {
      trigger()
      await Promise.resolve()
    })

    expect(toastSpy).toHaveBeenCalledTimes(1)
  })
})
