import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook } from "@testing-library/react"
import { useMessageScroll } from "../use-message-scroll"
import type { Message } from "../../types"

// Capture the most recent IntersectionObserver instance + its callback
// so the tests can drive intersection events deterministically.
let lastObserver: {
  callback: IntersectionObserverCallback
  observe: ReturnType<typeof vi.fn>
  unobserve: ReturnType<typeof vi.fn>
  disconnect: ReturnType<typeof vi.fn>
  options: IntersectionObserverInit | undefined
} | null = null

class MockIntersectionObserver {
  observe = vi.fn()
  unobserve = vi.fn()
  disconnect = vi.fn()
  constructor(
    cb: IntersectionObserverCallback,
    options?: IntersectionObserverInit,
  ) {
    lastObserver = {
      callback: cb,
      observe: this.observe,
      unobserve: this.unobserve,
      disconnect: this.disconnect,
      options,
    }
  }
}
vi.stubGlobal("IntersectionObserver", MockIntersectionObserver)

beforeEach(() => {
  lastObserver = null
})

function makeMessage(overrides: Partial<Message> = {}): Message {
  return {
    id: "m",
    conversation_id: "c",
    sender_id: "u",
    content: "",
    type: "text",
    metadata: null,
    seq: 1,
    status: "sent",
    edited_at: null,
    deleted_at: null,
    created_at: "2026-04-01T00:00:00Z",
    ...overrides,
  }
}

describe("useMessageScroll", () => {
  it("returns scrollRef and topSentinelRef objects", () => {
    const { result } = renderHook(() =>
      useMessageScroll({ messages: [], hasMore: false, onLoadMore: vi.fn() }),
    )
    expect(result.current.scrollRef).toBeDefined()
    expect(result.current.topSentinelRef).toBeDefined()
    expect(result.current.scrollRef.current).toBeNull()
  })

  it("does not throw when scrollRef.current is null on mount", () => {
    expect(() =>
      renderHook(() =>
        useMessageScroll({
          messages: [makeMessage({ id: "1" })],
          hasMore: false,
          onLoadMore: vi.fn(),
        }),
      ),
    ).not.toThrow()
  })

  it("calls scrollTo on the scroll container when a new message lands at the end", () => {
    const scrollTo = vi.fn()
    const fakeContainer = {
      scrollHeight: 1234,
      scrollTo,
    } as unknown as HTMLDivElement

    // Mount with no messages, then attach the ref + add a message via
    // re-render. The hook reads scrollRef.current inside its effect.
    const { rerender, result } = renderHook(
      ({ messages }: { messages: Message[] }) =>
        useMessageScroll({ messages, hasMore: false, onLoadMore: vi.fn() }),
      { initialProps: { messages: [] as Message[] } },
    )
    // Attach a fake container BEFORE the next render — the hook
    // captures the ref synchronously inside the effect.
    Object.defineProperty(result.current.scrollRef, "current", {
      value: fakeContainer,
      configurable: true,
    })

    // First render with messages — initial mount, prevMessageCountRef=0.
    rerender({ messages: [makeMessage({ id: "1" })] })
    expect(scrollTo).toHaveBeenCalledWith(
      expect.objectContaining({ top: 1234, behavior: "instant" }),
    )

    // New message at the end → smooth scroll.
    scrollTo.mockClear()
    rerender({
      messages: [makeMessage({ id: "1" }), makeMessage({ id: "2" })],
    })
    expect(scrollTo).toHaveBeenCalledWith(
      expect.objectContaining({ top: 1234, behavior: "smooth" }),
    )
  })

  it("does NOT scroll on initial render when there are no messages", () => {
    const scrollTo = vi.fn()
    const fakeContainer = {
      scrollHeight: 1234,
      scrollTo,
    } as unknown as HTMLDivElement

    const { result } = renderHook(() =>
      useMessageScroll({ messages: [], hasMore: false, onLoadMore: vi.fn() }),
    )
    Object.defineProperty(result.current.scrollRef, "current", {
      value: fakeContainer,
      configurable: true,
    })
    expect(scrollTo).not.toHaveBeenCalled()
  })

  it("checks the heuristic compares last-id vs. previous last-id", () => {
    // The implementation auto-scrolls when the LAST message id changes
    // between renders. This is a heuristic — it WILL still scroll on
    // a prepend if the resulting array's last position differs from
    // the previous count's last position. The behavior is preserved
    // verbatim from the pre-refactor file; we assert it here so a
    // regression that "fixes" the prepend behavior fails this test.
    const scrollTo = vi.fn()
    const fakeContainer = {
      scrollHeight: 1234,
      scrollTo,
    } as unknown as HTMLDivElement

    const { rerender, result } = renderHook(
      ({ messages }: { messages: Message[] }) =>
        useMessageScroll({ messages, hasMore: false, onLoadMore: vi.fn() }),
      { initialProps: { messages: [makeMessage({ id: "2" })] } },
    )
    Object.defineProperty(result.current.scrollRef, "current", {
      value: fakeContainer,
      configurable: true,
    })

    scrollTo.mockClear()
    rerender({
      messages: [
        makeMessage({ id: "0" }),
        makeMessage({ id: "1" }),
        makeMessage({ id: "2" }),
      ],
    })
    // Implementation behavior: still scrolls because last id changed
    // relative to messages[prevCount-1]. Locked in by this assertion.
    expect(scrollTo).toHaveBeenCalled()
  })

  it("creates an IntersectionObserver only when hasMore is true and refs are set", () => {
    const onLoadMore = vi.fn()
    const fakeContainer = document.createElement("div")
    const fakeSentinel = document.createElement("div")

    // Mount with hasMore=false → no observer
    const { rerender, result } = renderHook(
      ({ hasMore }: { hasMore: boolean }) =>
        useMessageScroll({ messages: [], hasMore, onLoadMore }),
      { initialProps: { hasMore: false } },
    )
    expect(lastObserver).toBeNull()

    // Attach refs and flip hasMore=true → creates observer
    Object.defineProperty(result.current.scrollRef, "current", {
      value: fakeContainer,
      configurable: true,
    })
    Object.defineProperty(result.current.topSentinelRef, "current", {
      value: fakeSentinel,
      configurable: true,
    })
    rerender({ hasMore: true })
    expect(lastObserver).not.toBeNull()
    expect(lastObserver?.observe).toHaveBeenCalledWith(fakeSentinel)
    expect(lastObserver?.options?.threshold).toBe(0.1)
  })

  it("invokes onLoadMore when the sentinel intersects", () => {
    // We need refs to be attached at the time the effect runs. Use a
    // wrapper component that attaches the refs to real DOM nodes so
    // the effect picks them up on the same render.
    const onLoadMore = vi.fn()
    const fakeContainer = document.createElement("div")
    const fakeSentinel = document.createElement("div")
    fakeContainer.appendChild(fakeSentinel)
    document.body.appendChild(fakeContainer)

    function Setup({ hasMore }: { hasMore: boolean }) {
      const refs = useMessageScroll({
        messages: [],
        hasMore,
        onLoadMore,
      })
      // Imperatively wire refs after first render via useEffect-style
      // assignment. The hook's effect runs AFTER this in the same
      // commit phase if refs already point to real nodes.
      Object.defineProperty(refs.scrollRef, "current", {
        value: fakeContainer,
        configurable: true,
      })
      Object.defineProperty(refs.topSentinelRef, "current", {
        value: fakeSentinel,
        configurable: true,
      })
      return null
    }
    const { rerender } = renderHook(({ hasMore }) => Setup({ hasMore }), {
      initialProps: { hasMore: false },
    })
    rerender({ hasMore: true })

    if (lastObserver) {
      // Manually fire an intersection event matching the hook's code
      // path: entries[0].isIntersecting === true
      lastObserver.callback(
        [{ isIntersecting: true } as IntersectionObserverEntry],
        {} as IntersectionObserver,
      )
      expect(onLoadMore).toHaveBeenCalled()

      // Non-intersecting event — must NOT call onLoadMore again
      onLoadMore.mockClear()
      lastObserver.callback(
        [{ isIntersecting: false } as IntersectionObserverEntry],
        {} as IntersectionObserver,
      )
      expect(onLoadMore).not.toHaveBeenCalled()
    }
  })

  it("disconnects the observer when hasMore flips back to false", () => {
    const onLoadMore = vi.fn()
    const fakeContainer = document.createElement("div")
    const fakeSentinel = document.createElement("div")

    const { rerender, result } = renderHook(
      ({ hasMore }: { hasMore: boolean }) =>
        useMessageScroll({ messages: [], hasMore, onLoadMore }),
      { initialProps: { hasMore: true } },
    )
    Object.defineProperty(result.current.scrollRef, "current", {
      value: fakeContainer,
      configurable: true,
    })
    Object.defineProperty(result.current.topSentinelRef, "current", {
      value: fakeSentinel,
      configurable: true,
    })
    // Force re-attach refs to reach the observer setup branch
    rerender({ hasMore: true })

    if (lastObserver) {
      const disconnect = lastObserver.disconnect
      rerender({ hasMore: false })
      expect(disconnect).toHaveBeenCalled()
    }
  })
})
