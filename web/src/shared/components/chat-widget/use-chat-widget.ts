"use client"

import { useState, useCallback, useEffect } from "react"

type ChatWidgetView = "list" | "chat"

export type PendingRecipient = {
  userId: string
  displayName: string
}

type ChatWidgetState = {
  isOpen: boolean
  view: ChatWidgetView
  activeConversationId: string | null
  pendingRecipient: PendingRecipient | null
}

const INITIAL_STATE: ChatWidgetState = {
  isOpen: false,
  view: "list",
  activeConversationId: null,
  pendingRecipient: null,
}

// Custom event name for global "open chat with user" trigger
const OPEN_CHAT_EVENT = "chat-widget:open-with-user"

/** Dispatch from anywhere to open the chat widget with a specific user (lazy conversation). */
export function openChatWithUser(userId: string, displayName: string) {
  window.dispatchEvent(
    new CustomEvent(OPEN_CHAT_EVENT, { detail: { userId, displayName } }),
  )
}

export function useChatWidget() {
  const [state, setState] = useState<ChatWidgetState>(INITIAL_STATE)

  const open = useCallback(() => {
    setState((prev) => ({ ...prev, isOpen: true }))
  }, [])

  const close = useCallback(() => {
    setState(INITIAL_STATE)
  }, [])

  const selectConversation = useCallback((id: string) => {
    setState({ isOpen: true, view: "chat", activeConversationId: id, pendingRecipient: null })
  }, [])

  const goBack = useCallback(() => {
    setState((prev) => ({ ...prev, view: "list", activeConversationId: null, pendingRecipient: null }))
  }, [])

  const openWithRecipient = useCallback((recipient: PendingRecipient) => {
    setState({ isOpen: true, view: "chat", activeConversationId: null, pendingRecipient: recipient })
  }, [])

  // Resolve pending recipient: when the first message is sent and we get a conversation ID
  const resolvePendingConversation = useCallback((conversationId: string) => {
    setState((prev) => ({
      ...prev,
      activeConversationId: conversationId,
      pendingRecipient: null,
    }))
  }, [])

  // Listen for global "open chat with user" events
  useEffect(() => {
    function handler(e: Event) {
      const { userId, displayName } = (e as CustomEvent).detail
      openWithRecipient({ userId, displayName })
    }
    window.addEventListener(OPEN_CHAT_EVENT, handler)
    return () => window.removeEventListener(OPEN_CHAT_EVENT, handler)
  }, [openWithRecipient])

  return {
    ...state,
    open,
    close,
    selectConversation,
    goBack,
    openWithRecipient,
    resolvePendingConversation,
  }
}
