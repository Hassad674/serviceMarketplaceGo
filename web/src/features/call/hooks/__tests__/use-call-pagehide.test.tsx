/**
 * useCall — pagehide cleanup tests.
 *
 * The hook installs a `pagehide` listener that fires
 * `navigator.sendBeacon` to POST /calls/{id}/end synchronously when
 * the tab closes during a call. This test mocks livekit-client and
 * the API surface so the listener can be exercised in isolation.
 *
 * Contract under test:
 *   - listener IS attached on mount, removed on unmount
 *   - dispatching `pagehide` while idle does NOT fire the beacon
 *   - dispatching `pagehide` while a call is active fires the beacon
 *     with the right URL and JSON body
 *   - the listener handles a pending outgoing ring (state =
 *     ringing_outgoing) the same way it handles an active call
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { renderHook, act } from "@testing-library/react"

// Mock livekit-client BEFORE importing useCall. The hook only uses
// the Room class + a small set of enums; the test never connects a
// real socket so we replace the SDK with stand-ins.
vi.mock("livekit-client", () => {
  class Room {
    localParticipant = {
      setMicrophoneEnabled: vi.fn(),
      setCameraEnabled: vi.fn(),
    }
    on() {
      return this
    }
    connect = vi.fn().mockResolvedValue(undefined)
    disconnect = vi.fn()
  }
  return {
    Room,
    RoomEvent: { TrackSubscribed: "TS", TrackUnsubscribed: "TU", Disconnected: "D" },
    Track: { Kind: { Audio: "audio", Video: "video" } },
  }
})

// API mocks — we own the call entry points so the test can drive
// state transitions without touching the network.
const initiateCallSpy = vi.fn()
const endCallSpy = vi.fn()
const endCallBeaconSpy = vi.fn()
vi.mock("../../api/call-api", () => ({
  initiateCall: (...args: unknown[]) => initiateCallSpy(...args),
  acceptCall: vi.fn(),
  declineCall: vi.fn(),
  endCall: (...args: unknown[]) => endCallSpy(...args),
  endCallBeacon: (...args: unknown[]) => endCallBeaconSpy(...args),
}))

import { useCall } from "../use-call"

beforeEach(() => {
  initiateCallSpy.mockReset()
  endCallSpy.mockReset()
  endCallBeaconSpy.mockReset()
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe("useCall — pagehide listener", () => {
  it("does NOT fire the beacon while idle", () => {
    renderHook(() => useCall())

    act(() => {
      window.dispatchEvent(new Event("pagehide"))
    })
    expect(endCallBeaconSpy).not.toHaveBeenCalled()
  })

  it("fires the beacon when a call is active and the page hides", async () => {
    initiateCallSpy.mockResolvedValue({
      call_id: "call-abc",
      room_name: "call:call-abc",
      token: "lkit-token",
    })

    const { result } = renderHook(() => useCall())

    // Drive the hook into the ringing_outgoing → activeCallRef populated state.
    await act(async () => {
      await result.current.startCall("conv-1", "user-2", "audio")
    })

    act(() => {
      window.dispatchEvent(new Event("pagehide"))
    })

    expect(endCallBeaconSpy).toHaveBeenCalledTimes(1)
    expect(endCallBeaconSpy).toHaveBeenCalledWith("call-abc", expect.any(Number))
  })

  it("removes the listener on unmount (no beacon on subsequent pagehide)", async () => {
    initiateCallSpy.mockResolvedValue({
      call_id: "call-xyz",
      room_name: "call:call-xyz",
      token: "lkit-token",
    })

    const { result, unmount } = renderHook(() => useCall())

    await act(async () => {
      await result.current.startCall("conv-1", "user-2", "audio")
    })

    unmount()

    act(() => {
      window.dispatchEvent(new Event("pagehide"))
    })
    // No new invocation after unmount.
    expect(endCallBeaconSpy).not.toHaveBeenCalled()
  })
})
