import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, act } from "@testing-library/react"
import { RoomEvent, Track } from "livekit-client"
import { useVideoTracks } from "../use-video-tracks"
import type { CallType } from "../../types"

// ---------------------------------------------------------------------------
// Helpers — minimal mock objects that satisfy the hook's contract
// ---------------------------------------------------------------------------

type Listener = (...args: unknown[]) => void

function createMockRoom({
  remoteVideoTrack,
  localVideoTrack,
}: {
  remoteVideoTrack?: unknown
  localVideoTrack?: unknown
} = {}) {
  const listeners = new Map<string, Set<Listener>>()

  // Build remote participants map
  const remoteParticipants = new Map()
  if (remoteVideoTrack) {
    const pub = {
      track: remoteVideoTrack,
      kind: Track.Kind.Video,
      isSubscribed: true,
    }
    const pubs = new Map([["TR_video", pub]])
    remoteParticipants.set("remote-1", { trackPublications: pubs })
  }

  // Build local participant
  const localPubs = new Map()
  if (localVideoTrack) {
    localPubs.set("TL_video", {
      track: localVideoTrack,
      kind: Track.Kind.Video,
    })
  }

  return {
    remoteParticipants,
    localParticipant: { trackPublications: localPubs },

    on(event: string, fn: Listener) {
      if (!listeners.has(event)) listeners.set(event, new Set())
      listeners.get(event)!.add(fn)
      return this
    },

    off(event: string, fn: Listener) {
      listeners.get(event)?.delete(fn)
      return this
    },

    emit(event: string, ...args: unknown[]) {
      listeners.get(event)?.forEach((fn) => fn(...args))
    },

    /** Expose listener count for assertions */
    listenerCount(event: string) {
      return listeners.get(event)?.size ?? 0
    },
  }
}

function createMockTrack(kind: string, sid: string) {
  return {
    kind,
    sid,
    attach: vi.fn(),
    detach: vi.fn(() => []),
  }
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("useVideoTracks", () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it("returns null for both tracks when callType is audio", () => {
    const room = createMockRoom()
    const { result } = renderHook(() =>
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      useVideoTracks(room as any, "audio"),
    )

    expect(result.current.remoteVideoTrack).toBeNull()
    expect(result.current.localVideoTrack).toBeNull()
  })

  it("returns null when room is null", () => {
    const { result } = renderHook(() => useVideoTracks(null, "video"))

    expect(result.current.remoteVideoTrack).toBeNull()
    expect(result.current.localVideoTrack).toBeNull()
  })

  it("finds existing remote video track on mount via scan", () => {
    const remoteTrack = createMockTrack(Track.Kind.Video, "RT_1")
    const room = createMockRoom({ remoteVideoTrack: remoteTrack })

    const { result } = renderHook(() =>
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      useVideoTracks(room as any, "video"),
    )

    expect(result.current.remoteVideoTrack).toBe(remoteTrack)
  })

  it("finds existing local video track on mount via scan", () => {
    const localTrack = createMockTrack(Track.Kind.Video, "LT_1")
    const room = createMockRoom({ localVideoTrack: localTrack })

    const { result } = renderHook(() =>
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      useVideoTracks(room as any, "video"),
    )

    expect(result.current.localVideoTrack).toBe(localTrack)
  })

  it("picks up remote video track via TrackSubscribed event", () => {
    const room = createMockRoom()
    const { result } = renderHook(() =>
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      useVideoTracks(room as any, "video"),
    )

    const newTrack = createMockTrack(Track.Kind.Video, "RT_new")
    act(() => {
      room.emit(
        RoomEvent.TrackSubscribed,
        newTrack, // track
        { track: newTrack, isSubscribed: true }, // pub
        { sid: "remote-2" }, // participant
      )
    })

    expect(result.current.remoteVideoTrack).toBe(newTrack)
  })

  it("ignores audio tracks in TrackSubscribed event", () => {
    const room = createMockRoom()
    const { result } = renderHook(() =>
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      useVideoTracks(room as any, "video"),
    )

    const audioTrack = createMockTrack(Track.Kind.Audio, "AT_1")
    act(() => {
      room.emit(
        RoomEvent.TrackSubscribed,
        audioTrack,
        { track: audioTrack, isSubscribed: true },
        { sid: "remote-2" },
      )
    })

    expect(result.current.remoteVideoTrack).toBeNull()
  })

  it("picks up local video track via LocalTrackPublished event", () => {
    const room = createMockRoom()
    const { result } = renderHook(() =>
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      useVideoTracks(room as any, "video"),
    )

    const localTrack = createMockTrack(Track.Kind.Video, "LT_new")
    act(() => {
      room.emit(RoomEvent.LocalTrackPublished, { track: localTrack })
    })

    expect(result.current.localVideoTrack).toBe(localTrack)
  })

  it("clears remote video track on TrackUnsubscribed", () => {
    const remoteTrack = createMockTrack(Track.Kind.Video, "RT_1")
    const room = createMockRoom({ remoteVideoTrack: remoteTrack })

    const { result } = renderHook(() =>
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      useVideoTracks(room as any, "video"),
    )
    expect(result.current.remoteVideoTrack).toBe(remoteTrack)

    act(() => {
      room.emit(RoomEvent.TrackUnsubscribed, remoteTrack)
    })

    expect(result.current.remoteVideoTrack).toBeNull()
    expect(remoteTrack.detach).toHaveBeenCalled()
  })

  it("clears local video track on LocalTrackUnpublished", () => {
    const localTrack = createMockTrack(Track.Kind.Video, "LT_1")
    const room = createMockRoom({ localVideoTrack: localTrack })

    const { result } = renderHook(() =>
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      useVideoTracks(room as any, "video"),
    )
    expect(result.current.localVideoTrack).toBe(localTrack)

    act(() => {
      room.emit(RoomEvent.LocalTrackUnpublished, { track: localTrack })
    })

    expect(result.current.localVideoTrack).toBeNull()
    expect(localTrack.detach).toHaveBeenCalled()
  })

  it("removes event listeners on unmount", () => {
    const room = createMockRoom()
    const { unmount } = renderHook(() =>
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      useVideoTracks(room as any, "video"),
    )

    expect(room.listenerCount(RoomEvent.TrackSubscribed)).toBe(1)
    expect(room.listenerCount(RoomEvent.LocalTrackPublished)).toBe(1)

    unmount()

    expect(room.listenerCount(RoomEvent.TrackSubscribed)).toBe(0)
    expect(room.listenerCount(RoomEvent.LocalTrackPublished)).toBe(0)
  })

  it("does not register listeners when callType switches to audio", () => {
    const room = createMockRoom()
    const { rerender } = renderHook(
      ({ callType }) =>
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        useVideoTracks(room as any, callType),
      { initialProps: { callType: "audio" as CallType } },
    )

    expect(room.listenerCount(RoomEvent.TrackSubscribed)).toBe(0)

    rerender({ callType: "video" as const })
    expect(room.listenerCount(RoomEvent.TrackSubscribed)).toBe(1)

    rerender({ callType: "audio" as const })
    expect(room.listenerCount(RoomEvent.TrackSubscribed)).toBe(0)
  })
})
