"use client"

import { useEffect, useRef, useState, useCallback } from "react"
import { useQueryClient } from "@tanstack/react-query"
import type { WSServerFrame, WSClientFrame, Message, MessageListResponse, ConversationListResponse, Conversation, ProposalMessageMetadata } from "../types"
import { markAsRead } from "../api/messaging-api"
import { conversationsQueryKey } from "./use-conversations"
import { messagesQueryKey } from "./use-messages"
import { unreadCountQueryKey } from "@/shared/hooks/use-unread-count"
import { proposalQueryKey } from "@/features/proposal/hooks/use-proposals"

const HEARTBEAT_INTERVAL = 30_000
const TYPING_CLEAR_DELAY = 5_000
const MAX_RECONNECT_DELAY = 30_000

async function getWSUrl(): Promise<string> {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || ""
  // In dev (NEXT_PUBLIC_API_URL is set), connect directly — session cookie is same-origin.
  if (apiUrl) {
    return apiUrl.replace(/^http/, "ws") + "/api/v1/ws"
  }
  // Production: NEXT_PUBLIC_API_URL is empty (client uses proxy for HTTP).
  // For WS we need the real backend URL. Use NEXT_PUBLIC_WS_URL if set.
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
    // Fall through
  }
  return `${wsUrl}/api/v1/ws`
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

  // Ref to track the currently active (viewed) conversation.
  // Updated by the parent component via setActiveConversationId.
  // Used to suppress unread increments for messages in the active conversation.
  const activeConversationIdRef = useRef<string | null>(null)

  const [isConnected, setIsConnected] = useState(false)
  const [typingUsers, setTypingUsers] = useState<TypingState>({})
  const [totalUnread, setTotalUnread] = useState(0)

  // Keep userIdRef in sync
  useEffect(() => {
    userIdRef.current = userId
  }, [userId])

  const setActiveConversationId = useCallback((id: string | null) => {
    activeConversationIdRef.current = id
  }, [])

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
      const queryKey = messagesQueryKey(userId, message.conversation_id)
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
    [queryClient, userId],
  )

  // When a proposal status change message arrives (accepted/declined/paid/etc.),
  // update the proposal_status in the metadata of all proposal_sent and
  // proposal_modified messages with the same proposal_id in the cache.
  const syncProposalStatusInCache = useCallback(
    (message: Message) => {
      const meta = message.metadata as ProposalMessageMetadata | null
      if (!meta?.proposal_id) return

      const PROPOSAL_STATUS_TYPES = new Set([
        "proposal_accepted",
        "proposal_declined",
        "proposal_paid",
        "proposal_completion_requested",
        "proposal_completed",
        "proposal_completion_rejected",
      ])
      if (!PROPOSAL_STATUS_TYPES.has(message.type)) return

      const newStatus = meta.proposal_status
      const proposalId = meta.proposal_id
      const queryKey = messagesQueryKey(userId, message.conversation_id)

      queryClient.setQueryData(
        queryKey,
        (old: { pages: MessageListResponse[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old
          let changed = false
          const newPages = old.pages.map((page) => ({
            ...page,
            data: page.data.map((msg) => {
              if (
                (msg.type === "proposal_sent" || msg.type === "proposal_modified") &&
                msg.metadata &&
                "proposal_id" in msg.metadata &&
                (msg.metadata as ProposalMessageMetadata).proposal_id === proposalId
              ) {
                changed = true
                return {
                  ...msg,
                  metadata: { ...(msg.metadata as ProposalMessageMetadata), proposal_status: newStatus },
                }
              }
              return msg
            }),
          }))
          return changed ? { ...old, pages: newPages } : old
        },
      )

      // Also invalidate the proposal detail query so /projects/{id} refreshes
      queryClient.invalidateQueries({ queryKey: [...proposalQueryKey(userId), proposalId] })
    },
    [queryClient, userId],
  )

  // Use a ref for the frame handler so the WS onmessage callback
  // always calls the latest version without reconnecting.
  const handleFrameRef = useRef<(frame: WSServerFrame) => void>(() => {})

  useEffect(() => {
    handleFrameRef.current = (frame: WSServerFrame) => {
      const uid = userIdRef.current
      switch (frame.type) {
        case "new_message": {
          const incomingMsg = frame.payload
          addMessageToCache(incomingMsg)
          clearTyping(incomingMsg.conversation_id)
          syncProposalStatusInCache(incomingMsg)

          const isActiveConversation =
            activeConversationIdRef.current === incomingMsg.conversation_id

          if (isActiveConversation) {
            queryClient.setQueryData(
              conversationsQueryKey(uid),
              (old: ConversationListResponse | undefined) => {
                if (!old) return old
                return {
                  ...old,
                  data: old.data.map((c: Conversation) =>
                    c.id === incomingMsg.conversation_id
                      ? { ...c, last_message: incomingMsg.content, last_message_at: incomingMsg.created_at, unread_count: 0, last_message_seq: incomingMsg.seq }
                      : c,
                  ),
                }
              },
            )
            markAsRead(incomingMsg.conversation_id, incomingMsg.seq).catch(() => {})
          } else {
            queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
          }
          break
        }
        case "typing": {
          const { conversation_id, user_id } = frame.payload
          if (user_id === userIdRef.current) return

          if (typingTimersRef.current[conversation_id]) {
            clearTimeout(typingTimersRef.current[conversation_id])
          }

          typingTimersRef.current[conversation_id] = setTimeout(
            () => clearTyping(conversation_id),
            TYPING_CLEAR_DELAY,
          )

          setTypingUsers((prev) => ({
            ...prev,
            [conversation_id]: { userId: user_id },
          }))
          break
        }
        case "status_update": {
          const { conversation_id, up_to_seq, status } = frame.payload
          queryClient.setQueryData(
            messagesQueryKey(uid, conversation_id),
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
          queryClient.invalidateQueries({ queryKey: unreadCountQueryKey(uid) })
          break
        }
        case "message_edited": {
          queryClient.setQueryData(
            messagesQueryKey(uid, frame.payload.conversation_id),
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
          queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
          break
        }
        case "message_deleted": {
          const { message_id, conversation_id: delConvId } = frame.payload
          queryClient.setQueryData(
            messagesQueryKey(uid, delConvId),
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
          queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
          break
        }
        case "presence": {
          // WS presence frames are user-scoped, but the conversation
          // list is org-scoped post phase R4 (any org member online =
          // org online). Mapping a user → org requires membership data
          // the client doesn't cache, so the cheapest correct option
          // is to refetch the list and let the backend fan-out do its
          // work again.
          queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
          break
        }
      }
    }
  }, [queryClient, addMessageToCache, clearTyping, syncProposalStatusInCache])

  const connect = useCallback(async () => {
    if (!userId) return
    if (wsRef.current?.readyState === WebSocket.OPEN) return

    const url = await getWSUrl()
    const ws = new WebSocket(url)
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

  return { isConnected, typingUsers, sendTyping, totalUnread, setActiveConversationId }
}
