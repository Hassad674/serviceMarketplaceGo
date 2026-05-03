/**
 * use-voice-recorder.test.ts
 *
 * Hook tests for the voice-message recorder. Covers:
 *   - idle state on mount
 *   - startRecording transitions to "recording" + ticks duration
 *   - stopRecording resolves a Blob and returns to idle
 *   - cancelRecording clears state without surfacing a Blob
 *   - setUploading toggles state to "uploading"
 *   - graceful no-op when stop is called with no recorder running
 *
 * The browser's MediaRecorder + getUserMedia are not available in
 * jsdom, so we install minimal fakes on the globals before each test.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { act, renderHook } from "@testing-library/react"
import { useVoiceRecorder } from "../use-voice-recorder"

class FakeMediaRecorder {
  static isTypeSupported(type: string): boolean {
    return type.includes("audio/webm")
  }
  state: "inactive" | "recording" = "inactive"
  ondataavailable: ((e: { data: Blob }) => void) | null = null
  onstop: (() => void) | null = null
  mimeType: string
  constructor(_stream: MediaStream, opts?: MediaRecorderOptions) {
    this.mimeType = opts?.mimeType ?? "audio/webm"
  }
  start() {
    this.state = "recording"
  }
  stop() {
    this.state = "inactive"
    // Push a chunk THEN fire onstop — mirrors the real browser API.
    this.ondataavailable?.({ data: new Blob(["audio"], { type: this.mimeType }) })
    this.onstop?.()
  }
}

class FakeMediaStreamTrack {
  stop = vi.fn()
}

class FakeMediaStream {
  getTracks() {
    return [new FakeMediaStreamTrack()]
  }
}

const getUserMediaMock = vi.fn().mockResolvedValue(new FakeMediaStream())

beforeEach(() => {
  vi.useFakeTimers()
  vi.stubGlobal("MediaRecorder", FakeMediaRecorder)
  Object.defineProperty(navigator, "mediaDevices", {
    configurable: true,
    value: { getUserMedia: getUserMediaMock },
  })
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})

describe("useVoiceRecorder — initial state", () => {
  it("starts in idle state with duration=0", () => {
    const { result } = renderHook(() => useVoiceRecorder())
    expect(result.current.state).toBe("idle")
    expect(result.current.duration).toBe(0)
  })
})

describe("useVoiceRecorder — startRecording", () => {
  it("transitions to recording and requests microphone access", async () => {
    const { result } = renderHook(() => useVoiceRecorder())
    await act(async () => {
      await result.current.startRecording()
    })
    expect(getUserMediaMock).toHaveBeenCalledWith({ audio: true })
    expect(result.current.state).toBe("recording")
  })

  it("ticks the duration counter once per second", async () => {
    const { result } = renderHook(() => useVoiceRecorder())
    await act(async () => {
      await result.current.startRecording()
    })
    act(() => {
      vi.advanceTimersByTime(3000)
    })
    expect(result.current.duration).toBe(3)
  })
})

describe("useVoiceRecorder — stopRecording", () => {
  it("returns a Blob on success and goes back to idle", async () => {
    const { result } = renderHook(() => useVoiceRecorder())
    await act(async () => {
      await result.current.startRecording()
    })
    let blob: Blob | null = null
    await act(async () => {
      blob = await result.current.stopRecording()
    })
    expect(blob).toBeInstanceOf(Blob)
    expect(result.current.state).toBe("idle")
  })

  it("returns null when called without an active recording", async () => {
    const { result } = renderHook(() => useVoiceRecorder())
    let blob: Blob | null = null
    await act(async () => {
      blob = await result.current.stopRecording()
    })
    expect(blob).toBeNull()
    expect(result.current.state).toBe("idle")
  })
})

describe("useVoiceRecorder — cancelRecording", () => {
  it("returns to idle and resets duration without producing a Blob", async () => {
    const { result } = renderHook(() => useVoiceRecorder())
    await act(async () => {
      await result.current.startRecording()
    })
    act(() => {
      vi.advanceTimersByTime(2000)
    })
    expect(result.current.duration).toBe(2)
    act(() => {
      result.current.cancelRecording()
    })
    expect(result.current.state).toBe("idle")
    expect(result.current.duration).toBe(0)
  })
})

describe("useVoiceRecorder — setUploading", () => {
  it("toggles to uploading and back to idle", () => {
    const { result } = renderHook(() => useVoiceRecorder())
    act(() => {
      result.current.setUploading(true)
    })
    expect(result.current.state).toBe("uploading")
    act(() => {
      result.current.setUploading(false)
    })
    expect(result.current.state).toBe("idle")
  })
})
