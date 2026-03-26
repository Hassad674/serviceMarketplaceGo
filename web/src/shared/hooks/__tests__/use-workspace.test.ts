import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { renderHook, act } from "@testing-library/react"
import { useWorkspace } from "../use-workspace"

function clearCookies() {
  document.cookie.split(";").forEach((c) => {
    const name = c.split("=")[0].trim()
    document.cookie = `${name}=; max-age=0; path=/`
  })
}

beforeEach(() => {
  clearCookies()
  // Mock fetch for switchToReferrer backend call
  vi.stubGlobal("fetch", vi.fn(() => Promise.resolve({ ok: true })))
})

afterEach(() => {
  clearCookies()
  vi.restoreAllMocks()
})

describe("useWorkspace", () => {
  it("defaults to isReferrerMode false when no cookie", () => {
    const { result } = renderHook(() => useWorkspace())
    expect(result.current.isReferrerMode).toBe(false)
  })

  it("reads isReferrerMode as true when workspace=referrer cookie exists", () => {
    document.cookie = "workspace=referrer; path=/; SameSite=Lax"
    const { result } = renderHook(() => useWorkspace())
    // After useEffect synchronizes with the cookie
    expect(result.current.isReferrerMode).toBe(true)
  })

  it("setReferrerMode(true) sets cookie and updates state", () => {
    const { result } = renderHook(() => useWorkspace())

    act(() => {
      result.current.setReferrerMode(true)
    })

    expect(result.current.isReferrerMode).toBe(true)
    expect(document.cookie).toContain("workspace=referrer")
  })

  it("setReferrerMode(false) clears cookie and updates state", () => {
    document.cookie = "workspace=referrer; path=/; SameSite=Lax"
    const { result } = renderHook(() => useWorkspace())

    act(() => {
      result.current.setReferrerMode(false)
    })

    expect(result.current.isReferrerMode).toBe(false)
    expect(document.cookie).not.toContain("workspace=referrer")
  })

  it("toggleMode switches from false to true", () => {
    const { result } = renderHook(() => useWorkspace())

    act(() => {
      result.current.toggleMode()
    })

    expect(result.current.isReferrerMode).toBe(true)
    expect(document.cookie).toContain("workspace=referrer")
  })

  it("toggleMode switches from true to false", () => {
    document.cookie = "workspace=referrer; path=/; SameSite=Lax"
    const { result } = renderHook(() => useWorkspace())

    // First, ensure state is synced to true via useEffect
    expect(result.current.isReferrerMode).toBe(true)

    act(() => {
      result.current.toggleMode()
    })

    expect(result.current.isReferrerMode).toBe(false)
    expect(document.cookie).not.toContain("workspace=referrer")
  })

  // --- switchToReferrer tests ---

  it("switchToReferrer sets referrer cookie and state to true", () => {
    const { result } = renderHook(() => useWorkspace())

    act(() => {
      result.current.switchToReferrer()
    })

    expect(result.current.isReferrerMode).toBe(true)
    expect(document.cookie).toContain("workspace=referrer")
  })

  it("switchToReferrer saves current path for freelance workspace", () => {
    // Set a known location
    Object.defineProperty(window, "location", {
      value: { pathname: "/dashboard" },
      writable: true,
    })

    const { result } = renderHook(() => useWorkspace())

    act(() => {
      result.current.switchToReferrer()
    })

    // The freelance path should be saved in a cookie
    expect(document.cookie).toContain("workspace_path_freelance=")
  })

  it("switchToReferrer returns the last referrer path (defaults to /dashboard)", () => {
    Object.defineProperty(window, "location", {
      value: { pathname: "/missions" },
      writable: true,
    })

    const { result } = renderHook(() => useWorkspace())

    let returnedPath: string | undefined
    act(() => {
      returnedPath = result.current.switchToReferrer()
    })

    expect(returnedPath).toBe("/dashboard")
  })

  it("switchToReferrer calls backend referrer-enable endpoint", () => {
    Object.defineProperty(window, "location", {
      value: { pathname: "/dashboard" },
      writable: true,
    })

    const { result } = renderHook(() => useWorkspace())

    act(() => {
      result.current.switchToReferrer()
    })

    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/v1/auth/referrer-enable"),
      expect.objectContaining({ method: "PUT", credentials: "include" }),
    )
  })

  // --- switchToFreelance tests ---

  it("switchToFreelance clears referrer cookie and state", () => {
    document.cookie = "workspace=referrer; path=/; SameSite=Lax"
    const { result } = renderHook(() => useWorkspace())

    // Ensure it starts in referrer mode
    expect(result.current.isReferrerMode).toBe(true)

    Object.defineProperty(window, "location", {
      value: { pathname: "/referrer/dashboard" },
      writable: true,
    })

    act(() => {
      result.current.switchToFreelance()
    })

    expect(result.current.isReferrerMode).toBe(false)
    expect(document.cookie).not.toContain("workspace=referrer")
  })

  it("switchToFreelance saves current path for referrer workspace", () => {
    document.cookie = "workspace=referrer; path=/; SameSite=Lax"
    Object.defineProperty(window, "location", {
      value: { pathname: "/referrer/clients" },
      writable: true,
    })

    const { result } = renderHook(() => useWorkspace())

    act(() => {
      result.current.switchToFreelance()
    })

    expect(document.cookie).toContain("workspace_path_referrer=")
  })

  it("switchToFreelance returns the last freelance path (defaults to /dashboard)", () => {
    document.cookie = "workspace=referrer; path=/; SameSite=Lax"
    Object.defineProperty(window, "location", {
      value: { pathname: "/referrer/dashboard" },
      writable: true,
    })

    const { result } = renderHook(() => useWorkspace())

    let returnedPath: string | undefined
    act(() => {
      returnedPath = result.current.switchToFreelance()
    })

    expect(returnedPath).toBe("/dashboard")
  })

  // --- path memory tests ---

  it("remembers freelance path when switching back from referrer", () => {
    Object.defineProperty(window, "location", {
      value: { pathname: "/missions" },
      writable: true,
    })

    const { result } = renderHook(() => useWorkspace())

    // Switch to referrer (saves /missions as freelance path)
    act(() => {
      result.current.switchToReferrer()
    })

    // Now change location to referrer workspace
    Object.defineProperty(window, "location", {
      value: { pathname: "/referrer/clients" },
      writable: true,
    })

    // Switch back to freelance
    let freelancePath: string | undefined
    act(() => {
      freelancePath = result.current.switchToFreelance()
    })

    expect(freelancePath).toBe("/missions")
  })
})
