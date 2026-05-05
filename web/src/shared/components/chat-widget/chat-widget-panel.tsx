"use client"

import { useEffect, useCallback, useMemo, useRef } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { cn } from "@/shared/lib/utils"
import { useUser } from "@/shared/hooks/use-user"
import { unreadCountQueryKey } from "@/shared/hooks/use-unread-count"
import { useConversations, conversationsQueryKey } from "@/features/messaging/hooks/use-conversations"
import { useMessages, useSendMessage, useEditMessage, useDeleteMessage } from "@/features/messaging/hooks/use-messages"
import { useMessagingWS } from "@/features/messaging/hooks/use-messaging-ws"
import { markAsRead, getPresignedURL, startConversation } from "@/features/messaging/api/messaging-api"
import type { Conversation, ConversationListResponse } from "@/features/messaging/types"
import { ChatWidgetConversationList } from "./chat-widget-conversation-list"
import { ChatWidgetChatView } from "./chat-widget-chat-view"
import type { PendingRecipient } from "./use-chat-widget"

interface ChatWidgetPanelProps {
  view: "list" | "chat"
  activeConversationId: string | null
  pendingRecipient: PendingRecipient | null
  onSelectConversation: (id: string) => void
  onBack: () => void
  onClose: () => void
  onPendingConversationResolved: (conversationId: string) => void
}

export function ChatWidgetPanel({
  view,
  activeConversationId,
  pendingRecipient,
  onSelectConversation,
  onBack,
  onClose,
  onPendingConversationResolved,
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

  // If pendingRecipient matches an existing conversation, open it directly
  useEffect(() => {
    if (!pendingRecipient || activeConversationId) return
    const existing = conversations.find(
      (c: Conversation) => c.other_org_id === pendingRecipient.orgId,
    )
    if (existing) {
      onPendingConversationResolved(existing.id)
    }
  }, [pendingRecipient, conversations, activeConversationId, onPendingConversationResolved])

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
        conversationsQueryKey(user?.id),
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
          unreadCountQueryKey(user?.id),
          (old: { count: number } | undefined) => {
            if (!old) return old
            return { count: Math.max(0, old.count - clearedUnread) }
          },
        )
      }

      onSelectConversation(id)
    },
    [conversations, queryClient, onSelectConversation, user?.id],
  )

  const handleSend = useCallback(
    async (content: string) => {
      // Pending recipient: create conversation with first message
      if (!activeConversationId && pendingRecipient) {
        try {
          const result = await startConversation(pendingRecipient.orgId, content)
          queryClient.invalidateQueries({ queryKey: conversationsQueryKey(user?.id) })
          queryClient.invalidateQueries({ queryKey: unreadCountQueryKey(user?.id) })
          onPendingConversationResolved(result.conversation_id)
        } catch {
          // silently fail — user can retry
        }
        return
      }
      if (!activeConversationId) return
      sendMessageMut.mutate({ content, type: "text" })
    },
    [activeConversationId, pendingRecipient, sendMessageMut, queryClient, user?.id, onPendingConversationResolved],
  )

  const handleSendFile = useCallback(
    async (files: File[]) => {
      if (!activeConversationId) return
      for (const file of files) {
        const { upload_url, public_url } = await getPresignedURL(
          file.name,
          file.type,
        )
        await fetch(upload_url, {
          method: "PUT",
          body: file,
          headers: { "Content-Type": file.type },
        })
        sendMessageMut.mutate({
          content: file.name,
          type: "file",
          metadata: {
            url: public_url,
            filename: file.name,
            size: file.size,
            mime_type: file.type,
          },
        })
      }
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

  const handleSendVoice = useCallback(
    (content: string, metadata: { url: string; duration: number; size: number; mime_type: string }) => {
      if (!activeConversationId) return
      sendMessageMut.mutate({ content, type: "voice", metadata })
    },
    [activeConversationId, sendMessageMut],
  )

  const handleTyping = useCallback(() => {
    if (activeConversationId) sendTyping(activeConversationId)
  }, [activeConversationId, sendTyping])

  return (
    <div
      className={cn(
        "fixed bottom-6 right-6 z-50 flex w-[380px] flex-col overflow-hidden",
        "rounded-2xl border border-border bg-card",
        "shadow-[0_8px_32px_rgba(42,31,21,0.12)]",
        "transition-all duration-200 ease-out",
        "animate-slide-up",
      )}
      style={{ height: "min(540px, calc(100vh - 96px))" }}
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
          conversationId={activeConversationId}
          pendingRecipient={pendingRecipient}
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
          onSendFile={handleSendFile}
          onSendVoice={handleSendVoice}
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
