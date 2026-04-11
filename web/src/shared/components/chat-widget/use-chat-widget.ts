"use client"

import { useState, useCallback, useEffect } from "react"

type ChatWidgetView = "list" | "chat"

// A pending recipient is an organization the user is about to start a
// conversation with. The backend resolves it to the org's Owner user
// id at send time — operators of that org all share the thread.
export type PendingRecipient = {
  orgId: string
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

const OPEN_CHAT_EVENT = "chat-widget:open-with-org"

/** Dispatch from anywhere to open the chat widget with a specific organization (lazy conversation). */
export function openChatWithOrg(orgId: string, displayName: string) {
  window.dispatchEvent(
    new CustomEvent(OPEN_CHAT_EVENT, { detail: { orgId, displayName } }),
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

  useEffect(() => {
    function handler(e: Event) {
      const { orgId, displayName } = (e as CustomEvent).detail
      openWithRecipient({ orgId, displayName })
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
