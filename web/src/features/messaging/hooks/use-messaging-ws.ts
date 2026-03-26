"use client"

import { useEffect, useRef, useState, useCallback } from "react"
import { useQueryClient } from "@tanstack/react-query"
import type { WSServerFrame, WSClientFrame, Message, MessageListResponse, ConversationListResponse } from "../types"
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

type TypingEntry = { userId: string }

type TypingState = Record<string, TypingEntry>

export function useMessagingWS(userId: string | undefined) {
  const queryClient = useQueryClient()
  const wsRef = useRef<WebSocket | null>(null)
  const heartbeatRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const reconnectAttemptRef = useRef(0)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const lastSeqMapRef = useRef<Record<string, number>>({})
  const userIdRef = useRef(userId)
  const typingTimersRef = useRef<Record<string, ReturnType<typeof setTimeout>>>({})

  const [isConnected, setIsConnected] = useState(false)
  const [typingUsers, setTypingUsers] = useState<TypingState>({})
  const [totalUnread, setTotalUnread] = useState(0)

  // Keep userIdRef in sync
  useEffect(() => {
    userIdRef.current = userId
  }, [userId])

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
      if (!prev[conversationId]) return prev
      const next = { ...prev }
      delete next[conversationId]
      return next
    })
    // Clean up the timer ref
    if (typingTimersRef.current[conversationId]) {
      clearTimeout(typingTimersRef.current[conversationId])
      delete typingTimersRef.current[conversationId]
    }
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
          // Prepend to page 0 (newest page, DESC order) so that after
          // chronological reversal the new message appears at the bottom.
          newPages[0] = {
            ...newPages[0],
            data: [message, ...newPages[0].data],
          }
          return { ...old, pages: newPages }
        },
      )
      lastSeqMapRef.current[message.conversation_id] = message.seq
    },
    [queryClient],
  )

  // Use a ref for the frame handler so the WS onmessage callback
  // always calls the latest version without reconnecting.
  const handleFrameRef = useRef<(frame: WSServerFrame) => void>(() => {})

  useEffect(() => {
    handleFrameRef.current = (frame: WSServerFrame) => {
      switch (frame.type) {
        case "new_message": {
          addMessageToCache(frame.payload)
          queryClient.invalidateQueries({ queryKey: CONVERSATIONS_QUERY_KEY })
          clearTyping(frame.payload.conversation_id)
          break
        }
        case "typing": {
          const { conversation_id, user_id } = frame.payload
          // Skip own typing events
          if (user_id === userIdRef.current) return

          // Clear any existing timer for this conversation
          if (typingTimersRef.current[conversation_id]) {
            clearTimeout(typingTimersRef.current[conversation_id])
          }

          // Set a new timer to clear the typing indicator
          typingTimersRef.current[conversation_id] = setTimeout(
            () => clearTyping(conversation_id),
            TYPING_CLEAR_DELAY,
          )

          // Update the typing state
          setTypingUsers((prev) => ({
            ...prev,
            [conversation_id]: { userId: user_id },
          }))
          break
        }
        case "status_update": {
          const { conversation_id, up_to_seq, status } = frame.payload
          const statusQueryKey = [MESSAGES_QUERY_KEY, conversation_id]
          queryClient.setQueryData(
            statusQueryKey,
            (old: { pages: MessageListResponse[]; pageParams: (string | undefined)[] } | undefined) => {
              if (!old) return old
              return {
                ...old,
                pages: old.pages.map((page) => ({
                  ...page,
                  data: page.data.map((msg) =>
                    msg.seq <= up_to_seq && msg.status !== "read"
                      ? { ...msg, status }
                      : msg,
                  ),
                })),
              }
            },
          )
          break
        }
        case "unread_count": {
          setTotalUnread(frame.payload.count)
          queryClient.invalidateQueries({ queryKey: UNREAD_COUNT_QUERY_KEY })
          break
        }
        case "message_edited": {
          const editedQueryKey = [MESSAGES_QUERY_KEY, frame.payload.conversation_id]
          queryClient.setQueryData(
            editedQueryKey,
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
          const { message_id, conversation_id: delConvId } = frame.payload
          const delQueryKey = [MESSAGES_QUERY_KEY, delConvId]
          queryClient.setQueryData(
            delQueryKey,
            (old: { pages: MessageListResponse[]; pageParams: (string | undefined)[] } | undefined) => {
              if (!old) return old
              return {
                ...old,
                pages: old.pages.map((page) => ({
                  ...page,
                  data: page.data.map((msg) =>
                    msg.id === message_id
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
        case "presence": {
          const { user_id: presenceUserId, online } = frame.payload
          queryClient.setQueryData(
            CONVERSATIONS_QUERY_KEY,
            (old: ConversationListResponse | undefined) => {
              if (!old) return old
              return {
                ...old,
                data: old.data.map((c) =>
                  c.other_user_id === presenceUserId
                    ? { ...c, online }
                    : c,
                ),
              }
            },
          )
          break
        }
      }
    }
  }, [queryClient, addMessageToCache, clearTyping])

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
        handleFrameRef.current(frame)
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
  }, [userId, sendFrame])

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
      // Clean up all typing timers
      for (const timer of Object.values(typingTimersRef.current)) {
        clearTimeout(timer)
      }
      typingTimersRef.current = {}
    }
  }, [connect])

  return { isConnected, typingUsers, sendTyping, totalUnread }
}
