import { describe, it, expect, beforeEach, afterEach } from "vitest"
import { act } from "@testing-library/react"
import { useTheme } from "../use-theme"

// Zustand persist stores state in localStorage. We reset between tests.
beforeEach(() => {
  localStorage.clear()
  // Reset the Zustand store to default state
  act(() => {
    useTheme.setState({ theme: "light" })
  })
  document.documentElement.classList.remove("dark")
})

afterEach(() => {
  localStorage.clear()
  document.documentElement.classList.remove("dark")
})

describe("useTheme", () => {
  it("has light as default theme", () => {
    const state = useTheme.getState()
    expect(state.theme).toBe("light")
  })

  it("toggle switches from light to dark", () => {
    act(() => {
      useTheme.getState().toggle()
    })

    expect(useTheme.getState().theme).toBe("dark")
    expect(document.documentElement.classList.contains("dark")).toBe(true)
  })

  it("toggle switches from dark back to light", () => {
    act(() => {
      useTheme.getState().setTheme("dark")
    })

    act(() => {
      useTheme.getState().toggle()
    })

    expect(useTheme.getState().theme).toBe("light")
    expect(document.documentElement.classList.contains("dark")).toBe(false)
  })

  it("setTheme sets a specific theme to dark", () => {
    act(() => {
      useTheme.getState().setTheme("dark")
    })

    expect(useTheme.getState().theme).toBe("dark")
    expect(document.documentElement.classList.contains("dark")).toBe(true)
  })

  it("setTheme sets a specific theme to light", () => {
    // Start as dark
    act(() => {
      useTheme.getState().setTheme("dark")
    })

    // Switch to light
    act(() => {
      useTheme.getState().setTheme("light")
    })

    expect(useTheme.getState().theme).toBe("light")
    expect(document.documentElement.classList.contains("dark")).toBe(false)
  })

  it("toggle applies dark class to document element", () => {
    expect(document.documentElement.classList.contains("dark")).toBe(false)

    act(() => {
      useTheme.getState().toggle()
    })

    expect(document.documentElement.classList.contains("dark")).toBe(true)
  })
})
