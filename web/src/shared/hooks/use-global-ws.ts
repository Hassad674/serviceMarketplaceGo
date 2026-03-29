"use client"

import { useEffect, useRef, useCallback } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { UNREAD_COUNT_QUERY_KEY } from "./use-unread-count"
import { UNREAD_NOTIF_COUNT_KEY } from "@/features/notification/hooks/use-unread-notification-count"
import { NOTIFICATIONS_QUERY_KEY } from "@/features/notification/hooks/use-notifications"

const HEARTBEAT_INTERVAL = 30_000
const MAX_RECONNECT_DELAY = 30_000

async function getWSUrl(): Promise<string> {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || ""
  // In dev (NEXT_PUBLIC_API_URL is set), connect directly — session cookie is same-origin.
  if (apiUrl) {
    return apiUrl.replace(/^http/, "ws") + "/api/v1/ws"
  }
  // Production: NEXT_PUBLIC_API_URL is empty (client uses proxy for HTTP).
  // For WS we need the real backend URL. Use NEXT_PUBLIC_WS_URL if set,
  // otherwise fetch a ws-token via the proxy and connect to the backend.
  const wsUrl = process.env.NEXT_PUBLIC_WS_URL || ""
  if (!wsUrl) {
    return "/api/v1/ws" // fallback: let the browser resolve relative to current origin
  }
  try {
    const res = await fetch("/api/v1/auth/ws-token", { credentials: "include" })
    if (res.ok) {
      const { token } = await res.json()
      return `${wsUrl}/api/v1/ws?ws_token=${token}`
    }
  } catch {
    // Fall through to direct connection attempt
  }
  return `${wsUrl}/api/v1/ws`
}

/**
 * Lightweight global WebSocket hook that maintains a persistent WS connection
 * at the layout level. Handles only global events (unread_count, presence)
 * so the sidebar badge updates in real time on every page.
 *
 * The full messaging WS handler (useMessagingWS) is mounted separately
 * on the messages page and handles message-specific events.
 *
 * When both are needed, the messaging page WS takes priority because
 * its connect call will find the existing connection via the shared
 * singleton ref.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type CallEventHandler = (payload: any) => void

export function useGlobalWS(userId: string | undefined, onCallEvent?: CallEventHandler) {
  const queryClient = useQueryClient()
  const wsRef = useRef<WebSocket | null>(null)
  const heartbeatRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const reconnectAttemptRef = useRef(0)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const isMessagingPageActiveRef = useRef(false)

  // Store the callback in a ref so that WS connection is not torn down
  // when the callback identity changes (e.g. when call state updates).
  const callEventHandlerRef = useRef<CallEventHandler | undefined>(onCallEvent)
  useEffect(() => {
    callEventHandlerRef.current = onCallEvent
  }, [onCallEvent])

  /**
   * When the messaging page mounts its own WS, it should suppress this
   * hook's connection to avoid duplicate WS connections. The messaging
   * page calls setMessagingPageActive(true) on mount and false on unmount.
   */
  const setMessagingPageActive = useCallback((active: boolean) => {
    isMessagingPageActiveRef.current = active
  }, [])

  const connect = useCallback(async () => {
    if (!userId) return
    if (wsRef.current?.readyState === WebSocket.OPEN) return

    const url = await getWSUrl()
    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onopen = () => {
      reconnectAttemptRef.current = 0
      heartbeatRef.current = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: "heartbeat" }))
        }
      }, HEARTBEAT_INTERVAL)
    }

    ws.onmessage = (event) => {
      try {
        const frame = JSON.parse(event.data)
        // Only handle global events at this level.
        // Message-specific events are handled by useMessagingWS on the messages page.
        if (frame.type === "unread_count") {
          queryClient.setQueryData(
            UNREAD_COUNT_QUERY_KEY,
            { count: frame.payload.count },
          )
        }
        if (frame.type === "notification") {
          queryClient.invalidateQueries({ queryKey: NOTIFICATIONS_QUERY_KEY })
          queryClient.setQueryData(UNREAD_NOTIF_COUNT_KEY, (old: unknown) => {
            const prev = old as { data?: { count?: number } } | undefined
            return {
              data: { count: ((prev?.data?.count ?? 0) + 1) },
            }
          })
        }
        if (frame.type === "notification_unread_count") {
          queryClient.setQueryData(UNREAD_NOTIF_COUNT_KEY, {
            data: { count: frame.payload?.count ?? 0 },
          })
        }
        if (frame.type === "call_event" && callEventHandlerRef.current) {
          callEventHandlerRef.current(frame.payload)
        }
      } catch {
        // Ignore malformed frames
      }
    }

    ws.onclose = () => {
      if (heartbeatRef.current) {
        clearInterval(heartbeatRef.current)
        heartbeatRef.current = null
      }
      const attempt = reconnectAttemptRef.current
      const delay = Math.min(1000 * Math.pow(2, attempt), MAX_RECONNECT_DELAY)
      reconnectAttemptRef.current = attempt + 1
      reconnectTimeoutRef.current = setTimeout(connect, delay)
    }

    ws.onerror = () => {
      ws.close()
    }
  }, [userId, queryClient])

  useEffect(() => {
    connect()

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (heartbeatRef.current) {
        clearInterval(heartbeatRef.current)
      }
      if (wsRef.current) {
        wsRef.current.onclose = null
        wsRef.current.close()
      }
    }
  }, [connect])

  return { setMessagingPageActive }
}
