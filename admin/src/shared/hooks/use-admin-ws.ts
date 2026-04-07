import { useEffect, useRef, useCallback } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { useAuth } from "@/shared/hooks/use-auth"

const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8083"
const HEARTBEAT_INTERVAL = 30_000
const MAX_RECONNECT_DELAY = 30_000

function getWSUrl(token: string): string {
  const wsBase = API_URL.replace(/^http/, "ws")
  return `${wsBase}/api/v1/ws?token=${token}`
}

/**
 * Lightweight WebSocket hook for the admin panel.
 * Listens for admin_notification_update events and invalidates
 * the notification counters query for instant UI updates.
 */
export function useAdminWS() {
  const { token } = useAuth()
  const queryClient = useQueryClient()
  const wsRef = useRef<WebSocket | null>(null)
  const heartbeatRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const reconnectAttemptRef = useRef(0)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const connect = useCallback(() => {
    if (!token) return
    if (wsRef.current?.readyState === WebSocket.OPEN) return

    const url = getWSUrl(token)
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
        if (frame.type === "admin_notification_update") {
          queryClient.invalidateQueries({ queryKey: ["admin", "notifications"] })
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
  }, [token, queryClient])

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
}
