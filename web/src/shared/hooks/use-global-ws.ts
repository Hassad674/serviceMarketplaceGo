"use client"

import { useEffect, useRef, useCallback } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { UNREAD_COUNT_QUERY_KEY } from "./use-unread-count"

const HEARTBEAT_INTERVAL = 30_000
const MAX_RECONNECT_DELAY = 30_000

function getWSUrl(): string {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8083"
  return apiUrl.replace(/^http/, "ws") + "/api/v1/ws"
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

  /**
   * When the messaging page mounts its own WS, it should suppress this
   * hook's connection to avoid duplicate WS connections. The messaging
   * page calls setMessagingPageActive(true) on mount and false on unmount.
   */
  const setMessagingPageActive = useCallback((active: boolean) => {
    isMessagingPageActiveRef.current = active
  }, [])

  const connect = useCallback(() => {
    if (!userId) return
    if (wsRef.current?.readyState === WebSocket.OPEN) return

    const ws = new WebSocket(getWSUrl())
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
        if (frame.type === "call_event" && onCallEvent) {
          onCallEvent(frame.payload)
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
  }, [userId, queryClient, onCallEvent])

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
