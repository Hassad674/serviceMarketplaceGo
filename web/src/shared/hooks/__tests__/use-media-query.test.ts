import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { renderHook, act } from "@testing-library/react"
import { useMediaQuery } from "../use-media-query"

type ChangeListener = (e: MediaQueryListEvent) => void

let listeners: Map<string, ChangeListener[]>
let matchesMap: Map<string, boolean>

function createMockMQL(query: string): MediaQueryList {
  return {
    matches: matchesMap.get(query) ?? false,
    media: query,
    onchange: null,
    addEventListener: vi.fn(((event: string, cb: EventListenerOrEventListenerObject) => {
      if (event === "change" && typeof cb === "function") {
        const list = listeners.get(query) ?? []
        list.push(cb as unknown as ChangeListener)
        listeners.set(query, list)
      }
    }) as MediaQueryList["addEventListener"]),
    removeEventListener: vi.fn(((event: string, cb: EventListenerOrEventListenerObject) => {
      if (event === "change" && typeof cb === "function") {
        const list = listeners.get(query) ?? []
        listeners.set(query, list.filter((l) => l !== cb))
        }
    }) as MediaQueryList["removeEventListener"]),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }
}

function fireMediaChange(query: string, newMatches: boolean) {
  matchesMap.set(query, newMatches)
  const cbs = listeners.get(query) ?? []
  for (const cb of cbs) {
    cb({ matches: newMatches, media: query } as MediaQueryListEvent)
  }
}

beforeEach(() => {
  listeners = new Map()
  matchesMap = new Map()
  vi.stubGlobal("matchMedia", vi.fn((query: string) => createMockMQL(query)))
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe("useMediaQuery", () => {
  it("returns false initially (SSR-safe default)", () => {
    const { result } = renderHook(() => useMediaQuery("(min-width: 768px)"))
    // Before useEffect runs on initial render, matches is false
    // After effect, it reads the mock which also defaults to false
    expect(result.current).toBe(false)
  })

  it("returns true when media query matches", () => {
    matchesMap.set("(min-width: 768px)", true)

    const { result } = renderHook(() => useMediaQuery("(min-width: 768px)"))

    expect(result.current).toBe(true)
  })

  it("updates when media query changes from false to true", () => {
    matchesMap.set("(max-width: 640px)", false)

    const { result } = renderHook(() => useMediaQuery("(max-width: 640px)"))
    expect(result.current).toBe(false)

    act(() => {
      fireMediaChange("(max-width: 640px)", true)
    })

    expect(result.current).toBe(true)
  })

  it("updates when media query changes from true to false", () => {
    matchesMap.set("(prefers-color-scheme: dark)", true)

    const { result } = renderHook(() => useMediaQuery("(prefers-color-scheme: dark)"))
    expect(result.current).toBe(true)

    act(() => {
      fireMediaChange("(prefers-color-scheme: dark)", false)
    })

    expect(result.current).toBe(false)
  })

  it("cleans up event listener on unmount", () => {
    const query = "(min-width: 1024px)"
    const { unmount } = renderHook(() => useMediaQuery(query))

    const mql = (window.matchMedia as ReturnType<typeof vi.fn>).mock.results[0].value
    expect(mql.addEventListener).toHaveBeenCalledWith("change", expect.any(Function))

    unmount()

    expect(mql.removeEventListener).toHaveBeenCalledWith("change", expect.any(Function))
  })

  it("re-evaluates when query string changes", () => {
    matchesMap.set("(min-width: 768px)", true)
    matchesMap.set("(min-width: 1024px)", false)

    const { result, rerender } = renderHook(
      ({ query }) => useMediaQuery(query),
      { initialProps: { query: "(min-width: 768px)" } },
    )

    expect(result.current).toBe(true)

    rerender({ query: "(min-width: 1024px)" })

    expect(result.current).toBe(false)
  })
})
