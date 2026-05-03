/**
 * voice-message.test.tsx
 *
 * Component tests for the voice message bubble. Covers:
 *   - rendering the play button + duration
 *   - swap to pause when playing
 *   - progressbar role + aria-value attributes
 *   - own-vs-other styling toggles via the isOwn prop
 *   - duration formatting helper
 *
 * The HTMLMediaElement APIs we exercise (.play / .pause / currentTime /
 * duration) are not implemented in jsdom, so we stub the relevant
 * methods on the prototype before each test.
 */
import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { VoiceMessage } from "../voice-message"

// jsdom does not implement HTMLMediaElement playback APIs. Stub them so
// the component does not throw "play() is not a function".
beforeEach(() => {
  Object.defineProperty(HTMLMediaElement.prototype, "play", {
    configurable: true,
    value: vi.fn().mockResolvedValue(undefined),
  })
  Object.defineProperty(HTMLMediaElement.prototype, "pause", {
    configurable: true,
    value: vi.fn(),
  })
  Object.defineProperty(HTMLMediaElement.prototype, "duration", {
    configurable: true,
    get() {
      return 30 // seconds
    },
  })
  Object.defineProperty(HTMLMediaElement.prototype, "paused", {
    configurable: true,
    get() {
      return true
    },
  })
})

const SAMPLE_METADATA = {
  url: "https://files/audio.webm",
  duration: 35,
  size: 1024,
  mime_type: "audio/webm",
}

describe("VoiceMessage — rendering", () => {
  it("renders a Play button initially", () => {
    render(<VoiceMessage metadata={SAMPLE_METADATA} isOwn={false} />)
    expect(screen.getByRole("button", { name: "Play" })).toBeInTheDocument()
  })

  it("renders a progressbar with role=progressbar", () => {
    render(<VoiceMessage metadata={SAMPLE_METADATA} isOwn={false} />)
    const progressbar = screen.getByRole("progressbar")
    expect(progressbar).toBeInTheDocument()
    expect(progressbar.getAttribute("aria-valuemin")).toBe("0")
    expect(progressbar.getAttribute("aria-valuemax")).toBe("100")
    expect(progressbar.getAttribute("aria-valuenow")).toBe("0")
  })

  it("renders the duration formatted as M:SS", () => {
    render(<VoiceMessage metadata={SAMPLE_METADATA} isOwn={false} />)
    // 35 seconds -> 0:35
    expect(screen.getByText("0:35")).toBeInTheDocument()
  })

  it("formats sub-1-minute durations correctly", () => {
    render(
      <VoiceMessage
        metadata={{ ...SAMPLE_METADATA, duration: 5 }}
        isOwn={false}
      />,
    )
    expect(screen.getByText("0:05")).toBeInTheDocument()
  })

  it("formats over-1-minute durations correctly", () => {
    render(
      <VoiceMessage
        metadata={{ ...SAMPLE_METADATA, duration: 125 }}
        isOwn={false}
      />,
    )
    expect(screen.getByText("2:05")).toBeInTheDocument()
  })

  it("formats 0-second durations as 0:00", () => {
    render(
      <VoiceMessage
        metadata={{ ...SAMPLE_METADATA, duration: 0 }}
        isOwn={false}
      />,
    )
    expect(screen.getByText("0:00")).toBeInTheDocument()
  })
})

describe("VoiceMessage — playback toggle", () => {
  it("calls audio.play() when the play button is clicked", () => {
    const playSpy = HTMLMediaElement.prototype.play as unknown as ReturnType<typeof vi.fn>
    render(<VoiceMessage metadata={SAMPLE_METADATA} isOwn={false} />)
    fireEvent.click(screen.getByRole("button", { name: "Play" }))
    expect(playSpy).toHaveBeenCalled()
  })
})

describe("VoiceMessage — own vs other styling", () => {
  it("renders distinct styling on the play button when isOwn=true", () => {
    const { rerender } = render(
      <VoiceMessage metadata={SAMPLE_METADATA} isOwn={false} />,
    )
    const otherClasses = screen.getByRole("button", { name: "Play" }).className
    rerender(<VoiceMessage metadata={SAMPLE_METADATA} isOwn={true} />)
    const ownClasses = screen.getByRole("button", { name: "Play" }).className
    expect(otherClasses).not.toBe(ownClasses)
  })
})

describe("VoiceMessage — progressbar interaction", () => {
  it("clicking the progressbar updates audio.currentTime", () => {
    let recordedCurrentTime = 0
    Object.defineProperty(HTMLMediaElement.prototype, "currentTime", {
      configurable: true,
      get() {
        return recordedCurrentTime
      },
      set(v: number) {
        recordedCurrentTime = v
      },
    })
    render(<VoiceMessage metadata={SAMPLE_METADATA} isOwn={false} />)
    const progressbar = screen.getByRole("progressbar")
    // Mock the bounding client rect so we have a deterministic width.
    progressbar.getBoundingClientRect = () => ({
      x: 0,
      y: 0,
      width: 100,
      height: 6,
      top: 0,
      left: 0,
      right: 100,
      bottom: 6,
      toJSON: () => ({}),
    })
    fireEvent.click(progressbar, { clientX: 50, clientY: 3 })
    // 50% of 30s = 15s
    expect(recordedCurrentTime).toBeCloseTo(15)
  })
})
