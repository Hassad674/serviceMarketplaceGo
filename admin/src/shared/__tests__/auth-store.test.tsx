import { describe, it, expect, beforeEach, afterEach, vi } from "vitest"
import {
  useAuthStore,
  getAuthToken,
  clearAuthToken,
} from "@/shared/stores/auth-store"

// SECURITY (SEC-FINAL-07): the admin bearer token must NEVER be
// persisted in browser storage. These tests are the contract that
// any future contributor adding a `persist` middleware would have
// to break — they will fail loudly if the rule is broken.

describe("auth-store (SEC-FINAL-07: in-memory only)", () => {
  beforeEach(() => {
    useAuthStore.getState().clear()
    localStorage.clear()
    sessionStorage.clear()
  })

  afterEach(() => {
    useAuthStore.getState().clear()
    localStorage.clear()
    sessionStorage.clear()
  })

  it("starts with a null token (no auto-load from any storage)", () => {
    expect(useAuthStore.getState().token).toBeNull()
    expect(getAuthToken()).toBeNull()
  })

  it("setToken stores the bearer in memory and exposes it via getAuthToken", () => {
    useAuthStore.getState().setToken("admin-bearer-xyz")
    expect(getAuthToken()).toBe("admin-bearer-xyz")
  })

  it("clear empties the in-memory token", () => {
    useAuthStore.getState().setToken("temp")
    useAuthStore.getState().clear()
    expect(getAuthToken()).toBeNull()
  })

  it("clearAuthToken free function drops the token (used by 401 interceptor)", () => {
    useAuthStore.getState().setToken("dying-token")
    clearAuthToken()
    expect(getAuthToken()).toBeNull()
  })

  it("never writes the token to localStorage (XSS-readable surface)", () => {
    useAuthStore.getState().setToken("must-not-leak")
    // Sweep every key — Zustand's persist middleware uses the store
    // name as the key, but a future regression could pick anything.
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i)
      const value = key ? localStorage.getItem(key) : ""
      expect(value).not.toContain("must-not-leak")
    }
  })

  it("never writes the token to sessionStorage", () => {
    useAuthStore.getState().setToken("must-not-leak")
    for (let i = 0; i < sessionStorage.length; i++) {
      const key = sessionStorage.key(i)
      const value = key ? sessionStorage.getItem(key) : ""
      expect(value).not.toContain("must-not-leak")
    }
  })

  it("simulated reload (storage.clear + fresh module read) returns null token", () => {
    useAuthStore.getState().setToken("session-bearer")
    // Simulate a hard reload: persistent storages are wiped (they
    // shouldn't have anything anyway), and a fresh getter would have
    // to rehydrate from somewhere. Since nothing is persisted, the
    // expected behavior is that AFTER clear() the token is null.
    localStorage.clear()
    sessionStorage.clear()
    useAuthStore.getState().clear() // mimics a fresh module init
    expect(getAuthToken()).toBeNull()
  })

  it("isHydrated transitions from false to true on markHydrated", () => {
    expect(useAuthStore.getState().isHydrated).toBe(false)
    useAuthStore.getState().markHydrated()
    expect(useAuthStore.getState().isHydrated).toBe(true)
  })

  it("subscribers are notified on setToken (for React reactivity)", () => {
    const observer = vi.fn()
    const unsubscribe = useAuthStore.subscribe(observer)
    useAuthStore.getState().setToken("new-token")
    expect(observer).toHaveBeenCalled()
    unsubscribe()
  })
})
