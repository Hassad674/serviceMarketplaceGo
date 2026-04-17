import { describe, expect, it, vi } from "vitest"
import { renderHook, act } from "@testing-library/react"
import { useDebouncedValue } from "../use-debounced-value"

describe("useDebouncedValue", () => {
  it("returns the initial value immediately", () => {
    const { result } = renderHook(() => useDebouncedValue("hello", 100))
    expect(result.current).toBe("hello")
  })

  it("debounces subsequent updates", () => {
    vi.useFakeTimers()
    const { result, rerender } = renderHook(
      ({ value }) => useDebouncedValue(value, 250),
      { initialProps: { value: "a" } },
    )
    rerender({ value: "b" })
    // before delay elapses
    act(() => {
      vi.advanceTimersByTime(100)
    })
    expect(result.current).toBe("a")
    act(() => {
      vi.advanceTimersByTime(200)
    })
    expect(result.current).toBe("b")
    vi.useRealTimers()
  })

  it("bypasses delay when delayMs is 0", () => {
    const { result, rerender } = renderHook(
      ({ value }) => useDebouncedValue(value, 0),
      { initialProps: { value: "a" } },
    )
    rerender({ value: "b" })
    expect(result.current).toBe("b")
  })
})
