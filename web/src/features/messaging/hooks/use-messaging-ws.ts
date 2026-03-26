"use client"

import { useEffect, useRef, useState, useCallback } from "react"
import { useQueryClient } from "@tanstack/react-query"
import type { WSServerFrame, WSClientFrame, Message, MessageListResponse } from "../types"
import { CONVERSATIONS_QUERY_KEY } from "./use-conversations"
import { MESSAGES_QUERY_KEY } from "./use-messages"
import { UNREAD_COUNT_QUERY_KEY } from "@/shared/hooks/use-unread-count"

const HEARTBEAT_INTERVAL = 30_000
const TYPING_CLEAR_DELAY = 3_000
const MAX_RECONNECT_DELAY = 30_000

function getWSUrl(): string {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8083"
  return apiUrl.replace(/^http/, "ws") + "/api/v1/ws"
}

type TypingState = Record<string, { userId: string; timeout: ReturnType<typeof setTimeout> }>

export function useMessagingWS(userId: string | undefined) {
  const queryClient = useQueryClient()
  const wsRef = useRef<WebSocket | null>(null)
  const heartbeatRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const reconnectAttemptRef = useRef(0)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const lastSeqMapRef = useRef<Record<string, number>>({})

  const [isConnected, setIsConnected] = useState(false)
  const [typingUsers, setTypingUsers] = useState<TypingState>({})
  const [totalUnread, setTotalUnread] = useState(0)

  const sendFrame = useCallback((frame: WSClientFrame) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(frame))
    }
  }, [])

  const sendTyping = useCallback(
    (conversationId: string) => {
      sendFrame({ type: "typing", conversation_id: conversationId })
    },
    [sendFrame],
  )

  const clearTyping = useCallback((conversationId: string) => {
    setTypingUsers((prev) => {
      const next = { ...prev }
      if (next[conversationId]) {
        clearTimeout(next[conversationId].timeout)
        delete next[conversationId]
      }
      return next
    })
  }, [])

  const addMessageToCache = useCallback(
    (message: Message) => {
      const queryKey = [MESSAGES_QUERY_KEY, message.conversation_id]
      queryClient.setQueryData(
        queryKey,
        (old: { pages: MessageListResponse[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old
          const allMessages = old.pages.flatMap((p) => p.data)
          if (allMessages.some((m) => m.id === message.id)) return old
          const newPages = [...old.pages]
          newPages[0] = {
            ...newPages[0],
            data: [...newPages[0].data, message],
          }
          return { ...old, pages: newPages }
        },
      )
      lastSeqMapRef.current[message.conversation_id] = message.seq
    },
    [queryClient],
  )

  const handleFrame = useCallback(
    (frame: WSServerFrame) => {
      switch (frame.type) {
        case "new_message": {
          addMessageToCache(frame.payload)
          queryClient.invalidateQueries({ queryKey: CONVERSATIONS_QUERY_KEY })
          clearTyping(frame.payload.conversation_id)
          break
        }
        case "typing": {
          if (frame.user_id === userId) return
          setTypingUsers((prev) => {
            const existing = prev[frame.conversation_id]
            if (existing) clearTimeout(existing.timeout)
            const timeout = setTimeout(
              () => clearTyping(frame.conversation_id),
              TYPING_CLEAR_DELAY,
            )
            return {
              ...prev,
              [frame.conversation_id]: { userId: frame.user_id, timeout },
            }
          })
          break
        }
        case "status_update": {
          queryClient.setQueriesData<{
            pages: MessageListResponse[]
            pageParams: (string | undefined)[]
          }>(
            { queryKey: [MESSAGES_QUERY_KEY] },
            (old) => {
              if (!old) return old
              return {
                ...old,
                pages: old.pages.map((page) => ({
                  ...page,
                  data: page.data.map((msg) =>
                    msg.id === frame.message_id
                      ? { ...msg, status: frame.status }
                      : msg,
                  ),
                })),
              }
            },
          )
          break
        }
        case "unread_count": {
          setTotalUnread(frame.count)
          queryClient.invalidateQueries({ queryKey: UNREAD_COUNT_QUERY_KEY })
          break
        }
        case "message_edited": {
          const queryKey = [MESSAGES_QUERY_KEY, frame.payload.conversation_id]
          queryClient.setQueryData(
            queryKey,
            (old: { pages: MessageListResponse[]; pageParams: (string | undefined)[] } | undefined) => {
              if (!old) return old
              return {
                ...old,
                pages: old.pages.map((page) => ({
                  ...page,
                  data: page.data.map((msg) =>
                    msg.id === frame.payload.id ? frame.payload : msg,
                  ),
                })),
              }
            },
          )
          queryClient.invalidateQueries({ queryKey: CONVERSATIONS_QUERY_KEY })
          break
        }
        case "message_deleted": {
          const msgQueryKey = [MESSAGES_QUERY_KEY, frame.conversation_id]
          queryClient.setQueryData(
            msgQueryKey,
            (old: { pages: MessageListResponse[]; pageParams: (string | undefined)[] } | undefined) => {
              if (!old) return old
              return {
                ...old,
                pages: old.pages.map((page) => ({
                  ...page,
                  data: page.data.map((msg) =>
                    msg.id === frame.message_id
                      ? { ...msg, deleted_at: new Date().toISOString(), content: "" }
                      : msg,
                  ),
                })),
              }
            },
          )
          queryClient.invalidateQueries({ queryKey: CONVERSATIONS_QUERY_KEY })
          break
        }
      }
    },
    [queryClient, userId, addMessageToCache, clearTyping],
  )

  const connect = useCallback(() => {
    if (!userId) return
    if (wsRef.current?.readyState === WebSocket.OPEN) return

    const ws = new WebSocket(getWSUrl())
    wsRef.current = ws

    ws.onopen = () => {
      setIsConnected(true)
      reconnectAttemptRef.current = 0

      heartbeatRef.current = setInterval(() => {
        sendFrame({ type: "heartbeat" })
      }, HEARTBEAT_INTERVAL)

      // Sync last known sequences on reconnect
      const seqMap = lastSeqMapRef.current
      if (Object.keys(seqMap).length > 0) {
        sendFrame({ type: "sync", conversations: seqMap })
      }
    }

    ws.onmessage = (event) => {
      try {
        const frame = JSON.parse(event.data) as WSServerFrame
        handleFrame(frame)
      } catch {
        // Ignore malformed frames
      }
    }

    ws.onclose = () => {
      setIsConnected(false)
      if (heartbeatRef.current) {
        clearInterval(heartbeatRef.current)
        heartbeatRef.current = null
      }

      // Exponential backoff reconnect
      const attempt = reconnectAttemptRef.current
      const delay = Math.min(1000 * Math.pow(2, attempt), MAX_RECONNECT_DELAY)
      reconnectAttemptRef.current = attempt + 1

      reconnectTimeoutRef.current = setTimeout(connect, delay)
    }

    ws.onerror = () => {
      ws.close()
    }
  }, [userId, sendFrame, handleFrame])

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

  return { isConnected, typingUsers, sendTyping, totalUnread }
}
