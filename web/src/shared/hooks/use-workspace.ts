"use client"

import { useCallback, useSyncExternalStore } from "react"
import { API_BASE_URL } from "@/shared/lib/api-client"

const COOKIE_NAME = "workspace"
const REFERRER_VALUE = "referrer"
const DEFAULT_PATH = "/dashboard"

/**
 * Read the workspace cookie synchronously.
 * Returns true if the cookie is set to "referrer".
 */
function readWorkspaceCookie(): boolean {
  if (typeof document === "undefined") return false
  const match = document.cookie
    .split("; ")
    .find((row) => row.startsWith(`${COOKIE_NAME}=`))
  return match?.split("=")[1] === REFERRER_VALUE
}

/**
 * Set the workspace cookie. SameSite=Lax, path=/, no expiry (session cookie).
 */
function setWorkspaceCookie(isReferrer: boolean): void {
  if (isReferrer) {
    document.cookie = `${COOKIE_NAME}=${REFERRER_VALUE}; path=/; SameSite=Lax`
  } else {
    // Delete the cookie by setting an expired date
    document.cookie = `${COOKIE_NAME}=; path=/; SameSite=Lax; max-age=0`
  }
}

/**
 * Save the last visited path for a given workspace (freelance or referrer).
 */
function stripLocale(path: string): string {
  // Remove locale prefix (/fr/, /en/) since next-intl router adds it automatically
  return path.replace(/^\/(fr|en)(\/|$)/, "/")
}

function saveLastPath(workspace: string, path: string): void {
  const cleanPath = stripLocale(path)
  document.cookie = `workspace_path_${workspace}=${encodeURIComponent(cleanPath)}; path=/; SameSite=Lax`
}

/**
 * Read the last visited path for a given workspace.
 * Returns "/dashboard" when no path has been saved yet.
 */
function getLastPath(workspace: string): string {
  if (typeof document === "undefined") return DEFAULT_PATH
  const match = document.cookie
    .split("; ")
    .find((row) => row.startsWith(`workspace_path_${workspace}=`))
  return match ? decodeURIComponent(match.split("=")[1]) : DEFAULT_PATH
}

// Set of subscribers that should be notified when the workspace cookie
// changes. We notify them via setWorkspaceCookie (the only mutation
// path we control); cross-tab cookie changes are not covered, which
// matches the previous behaviour.
const cookieSubscribers = new Set<() => void>()

function notifyCookieChange(): void {
  for (const listener of cookieSubscribers) {
    listener()
  }
}

function subscribeToWorkspaceCookie(onStoreChange: () => void): () => void {
  cookieSubscribers.add(onStoreChange)
  return () => {
    cookieSubscribers.delete(onStoreChange)
  }
}

/**
 * Hook to manage the referrer workspace mode via a cookie.
 *
 * Returns the current mode, a toggle function, and switch helpers
 * that save/restore the last visited path per workspace.
 */
export function useWorkspace() {
  // Source of truth is the cookie. useSyncExternalStore reads it on
  // every render that follows a notification, so we never need to mirror
  // it into local React state (which would require a setState-in-effect
  // bootstrap).
  const isReferrerMode = useSyncExternalStore(
    subscribeToWorkspaceCookie,
    readWorkspaceCookie,
    () => false, // SSR fallback — document is unavailable on the server
  )

  const setReferrerMode = useCallback((enabled: boolean) => {
    setWorkspaceCookie(enabled)
    notifyCookieChange()
  }, [])

  const toggleMode = useCallback(() => {
    setReferrerMode(!isReferrerMode)
  }, [isReferrerMode, setReferrerMode])

  const switchToReferrer = useCallback(() => {
    const currentPath = window.location.pathname
    saveLastPath("freelance", currentPath)
    setWorkspaceCookie(true)
    notifyCookieChange()

    // Sync referrer_enabled=true to the backend (once set, stays true permanently)
    fetch(`${API_BASE_URL}/api/v1/auth/referrer-enable`, {
      method: "PUT",
      credentials: "include",
    }).catch(() => {
      // Silent failure — the UI workspace switch works regardless of backend sync
    })

    return getLastPath("referrer")
  }, [])

  const switchToFreelance = useCallback(() => {
    const currentPath = window.location.pathname
    saveLastPath("referrer", currentPath)
    setWorkspaceCookie(false)
    notifyCookieChange()
    return getLastPath("freelance")
  }, [])

  return {
    isReferrerMode,
    setReferrerMode,
    toggleMode,
    switchToReferrer,
    switchToFreelance,
  } as const
}
