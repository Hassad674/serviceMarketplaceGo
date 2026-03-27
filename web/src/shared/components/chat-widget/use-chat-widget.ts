"use client"

import { useState, useCallback } from "react"

type ChatWidgetView = "list" | "chat"

type ChatWidgetState = {
  isOpen: boolean
  view: ChatWidgetView
  activeConversationId: string | null
}

const INITIAL_STATE: ChatWidgetState = {
  isOpen: false,
  view: "list",
  activeConversationId: null,
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
    setState({ isOpen: true, view: "chat", activeConversationId: id })
  }, [])

  const goBack = useCallback(() => {
    setState((prev) => ({ ...prev, view: "list", activeConversationId: null }))
  }, [])

  return {
    ...state,
    open,
    close,
    selectConversation,
    goBack,
  }
}
