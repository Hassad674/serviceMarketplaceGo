import { describe, it, expect, beforeEach, afterEach } from "vitest"
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
})

afterEach(() => {
  clearCookies()
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
})
