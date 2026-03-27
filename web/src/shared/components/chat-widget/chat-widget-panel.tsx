"use client"

import { useEffect, useCallback, useMemo, useRef } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { cn } from "@/shared/lib/utils"
import { useUser } from "@/shared/hooks/use-user"
import { UNREAD_COUNT_QUERY_KEY } from "@/shared/hooks/use-unread-count"
import { useConversations, CONVERSATIONS_QUERY_KEY } from "@/features/messaging/hooks/use-conversations"
import { useMessages, useSendMessage, useEditMessage, useDeleteMessage } from "@/features/messaging/hooks/use-messages"
import { useMessagingWS } from "@/features/messaging/hooks/use-messaging-ws"
import { markAsRead } from "@/features/messaging/api/messaging-api"
import type { Conversation, ConversationListResponse } from "@/features/messaging/types"
import { ChatWidgetConversationList } from "./chat-widget-conversation-list"
import { ChatWidgetChatView } from "./chat-widget-chat-view"

interface ChatWidgetPanelProps {
  view: "list" | "chat"
  activeConversationId: string | null
  onSelectConversation: (id: string) => void
  onBack: () => void
  onClose: () => void
}

export function ChatWidgetPanel({
  view,
  activeConversationId,
  onSelectConversation,
  onBack,
  onClose,
}: ChatWidgetPanelProps) {
  const { data: user } = useUser()
  const queryClient = useQueryClient()
  const prevMarkedSeqRef = useRef(0)

  // Real-time WS connection (only while panel is open)
  const { typingUsers, sendTyping, setActiveConversationId } = useMessagingWS(user?.id)

  // Conversations list
  const { data: conversationsData, isLoading: conversationsLoading } = useConversations()
  const conversations = conversationsData?.data ?? []

  // Messages for active conversation
  const messagesQuery = useMessages(activeConversationId)
  const sendMessageMut = useSendMessage(activeConversationId)
  const editMessageMut = useEditMessage(activeConversationId)
  const deleteMessageMut = useDeleteMessage(activeConversationId)

  const activeConversation = conversations.find(
    (c: Conversation) => c.id === activeConversationId,
  )

  // Keep the WS hook aware of active conversation
  useEffect(() => {
    setActiveConversationId(activeConversationId)
  }, [activeConversationId, setActiveConversationId])

  // Reset marked seq when conversation changes
  useEffect(() => {
    prevMarkedSeqRef.current = 0
  }, [activeConversationId])

  // Flatten messages for display (reverse for chronological order)
  const allMessages = useMemo(() => {
    if (!messagesQuery.data) return []
    return [...messagesQuery.data.pages]
      .reverse()
      .flatMap((page) => [...page.data].reverse())
  }, [messagesQuery.data])

  // Track latest seq for read receipts
  const latestSeq = useMemo(() => {
    if (allMessages.length === 0) return 0
    return allMessages[allMessages.length - 1]?.seq ?? 0
  }, [allMessages])

  // Mark as read when opening a conversation or new messages arrive
  useEffect(() => {
    if (!activeConversationId || !user?.id) return
    const seqToMark = Math.max(
      activeConversation?.last_message_seq ?? 0,
      latestSeq,
    )
    if (seqToMark <= 0 || seqToMark <= prevMarkedSeqRef.current) return
    prevMarkedSeqRef.current = seqToMark
    markAsRead(activeConversationId, seqToMark).catch(() => {})
  }, [activeConversationId, latestSeq, user?.id, activeConversation?.last_message_seq])

  const handleSelect = useCallback(
    (id: string) => {
      const conv = conversations.find((c: Conversation) => c.id === id)
      const clearedUnread = conv?.unread_count ?? 0

      // Optimistically clear unread
      queryClient.setQueryData(
        CONVERSATIONS_QUERY_KEY,
        (old: ConversationListResponse | undefined) => {
          if (!old) return old
          return {
            ...old,
            data: old.data.map((c: Conversation) =>
              c.id === id ? { ...c, unread_count: 0 } : c,
            ),
          }
        },
      )
      if (clearedUnread > 0) {
        queryClient.setQueryData(
          UNREAD_COUNT_QUERY_KEY,
          (old: { count: number } | undefined) => {
            if (!old) return old
            return { count: Math.max(0, old.count - clearedUnread) }
          },
        )
      }

      onSelectConversation(id)
    },
    [conversations, queryClient, onSelectConversation],
  )

  const handleSend = useCallback(
    (content: string) => {
      if (!activeConversationId) return
      sendMessageMut.mutate({ content, type: "text" })
    },
    [activeConversationId, sendMessageMut],
  )

  const handleEdit = useCallback(
    (messageId: string, content: string) => {
      editMessageMut.mutate({ messageId, content })
    },
    [editMessageMut],
  )

  const handleDelete = useCallback(
    (messageId: string) => {
      deleteMessageMut.mutate(messageId)
    },
    [deleteMessageMut],
  )

  const handleTyping = useCallback(() => {
    if (activeConversationId) sendTyping(activeConversationId)
  }, [activeConversationId, sendTyping])

  return (
    <div
      className={cn(
        "fixed bottom-24 right-6 z-50 flex w-[380px] flex-col overflow-hidden",
        "rounded-2xl border border-gray-200 bg-white shadow-xl",
        "dark:border-gray-700 dark:bg-gray-900",
        "animate-scale-in origin-bottom-right",
      )}
      style={{ maxHeight: "500px" }}
    >
      {view === "list" ? (
        <ChatWidgetConversationList
          conversations={conversations}
          isLoading={conversationsLoading}
          typingUsers={typingUsers}
          onSelect={handleSelect}
          onClose={onClose}
        />
      ) : (
        <ChatWidgetChatView
          conversation={activeConversation ?? null}
          messages={allMessages}
          currentUserId={user?.id ?? ""}
          isLoading={messagesQuery.isLoading}
          hasMore={messagesQuery.hasNextPage ?? false}
          isSending={sendMessageMut.isPending}
          typingUser={
            activeConversationId ? typingUsers[activeConversationId] : undefined
          }
          onLoadMore={() => messagesQuery.fetchNextPage()}
          onSend={handleSend}
          onEdit={handleEdit}
          onDelete={handleDelete}
          onTyping={handleTyping}
          onBack={onBack}
          onClose={onClose}
        />
      )}
    </div>
  )
}
