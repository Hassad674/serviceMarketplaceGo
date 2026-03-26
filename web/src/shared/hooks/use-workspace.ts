"use client"

import { useState, useCallback } from "react"

const COOKIE_NAME = "workspace"
const REFERRER_VALUE = "referrer"

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
 * Hook to manage the referrer workspace mode via a cookie.
 *
 * Returns the current mode and a toggle function.
 * The cookie persists across navigations and page reloads without
 * polluting URLs with query parameters.
 */
export function useWorkspace() {
  const [isReferrerMode, setIsReferrerMode] = useState(readWorkspaceCookie)

  const setReferrerMode = useCallback((enabled: boolean) => {
    setWorkspaceCookie(enabled)
    setIsReferrerMode(enabled)
  }, [])

  const toggleMode = useCallback(() => {
    setReferrerMode(!isReferrerMode)
  }, [isReferrerMode, setReferrerMode])

  return { isReferrerMode, setReferrerMode, toggleMode } as const
}
