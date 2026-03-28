"use client"

import { useState, useCallback, useEffect, useRef, useMemo } from "react"
import { MessageSquare } from "lucide-react"
import { useTranslations } from "next-intl"
import { useSearchParams } from "next/navigation"
import { useRouter } from "@i18n/navigation"
import { useQueryClient } from "@tanstack/react-query"
import { cn } from "@/shared/lib/utils"
import { useUser } from "@/shared/hooks/use-user"
import { useCallContext } from "@/shared/hooks/use-call-context"
import { ConversationList } from "./conversation-list"
import { ConversationHeader } from "./conversation-header"
import { MessageArea } from "./message-area"
import { MessageInput } from "./message-input"
import { useConversations, CONVERSATIONS_QUERY_KEY } from "../hooks/use-conversations"
import { useMessages, useSendMessage, useEditMessage, useDeleteMessage } from "../hooks/use-messages"
import { useMessagingWS } from "../hooks/use-messaging-ws"
import { markAsRead } from "../api/messaging-api"
import { UNREAD_COUNT_QUERY_KEY } from "@/shared/hooks/use-unread-count"
import { ReviewModal } from "@/features/review/components/review-modal"
import type { Conversation, ConversationListResponse, Message } from "../types"

export function MessagingPage() {
  const t = useTranslations("messaging")
  const searchParams = useSearchParams()
  const router = useRouter()
  const { data: user } = useUser()
  const initializedFromUrl = useRef(false)

  const [activeId, setActiveId] = useState<string | null>(
    searchParams.get("id"),
  )
  const [roleFilter, setRoleFilter] = useState<"all" | string>("all")
  const [searchQuery, setSearchQuery] = useState("")
  const [mobileView, setMobileView] = useState<"list" | "chat">("list")
  const [replyTo, setReplyTo] = useState<{ id: string; senderId: string; senderName: string; content: string; type: string } | null>(null)
  const [reviewTarget, setReviewTarget] = useState<{ proposalId: string; proposalTitle: string } | null>(null)

  const callCtx = useCallContext()

  const { data: conversationsData, isLoading: conversationsLoading } = useConversations()
  const messagesQuery = useMessages(activeId)
  const sendMessage = useSendMessage(activeId)
  const editMessageMut = useEditMessage(activeId)
  const deleteMessageMut = useDeleteMessage(activeId)
  const { typingUsers, sendTyping, isConnected, setActiveConversationId } = useMessagingWS(user?.id)

  const queryClient = useQueryClient()

  const conversations = conversationsData?.data ?? []
  const activeConversation = conversations.find(
    (c: Conversation) => c.id === activeId,
  )

  // Keep the WS hook aware of which conversation is currently active,
  // so it can suppress unread increments for messages in this conversation.
  useEffect(() => {
    setActiveConversationId(activeId)
  }, [activeId, setActiveConversationId])

  // Gather all messages from infinite query pages.
  // Backend returns messages in DESC order (newest first) for cursor pagination.
  // We reverse pages and each page's data to get chronological order
  // (oldest at top, newest at bottom) for display.
  const allMessages = messagesQuery.data
    ? [...messagesQuery.data.pages].reverse().flatMap((page) => [...page.data].reverse())
    : []

  // Deep-link from query param — only on initial mount.
  // Subsequent conversation switches are handled by handleSelect
  // which updates both activeId and the URL.
  useEffect(() => {
    if (initializedFromUrl.current) return
    initializedFromUrl.current = true

    const paramId = searchParams.get("id")
    if (paramId && paramId !== activeId) {
      setActiveId(paramId)
      setMobileView("chat")
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Track the latest message seq in the active conversation for read receipts
  const latestSeq = useMemo(() => {
    if (allMessages.length === 0) return 0
    return allMessages[allMessages.length - 1]?.seq ?? 0
  }, [allMessages])

  // Mark as read when opening a conversation or when new messages arrive
  // while the conversation is already active.
  // Uses prevMarkedSeqRef to avoid re-calling markAsRead on every render
  // (BUG-04 fix: removed allMessages from deps, gate on seq change).
  const prevMarkedSeqRef = useRef(0)
  useEffect(() => {
    if (!activeId || !user?.id) return
    const seqToMark = Math.max(activeConversation?.last_message_seq ?? 0, latestSeq)
    if (seqToMark <= 0 || seqToMark <= prevMarkedSeqRef.current) return
    prevMarkedSeqRef.current = seqToMark
    markAsRead(activeId, seqToMark).catch(() => {
      // Silent fail — unread count will refresh via WS
    })
  }, [activeId, latestSeq, user?.id, activeConversation?.last_message_seq])

  const handleSelect = useCallback((id: string) => {
    setActiveId(id)
    setMobileView("chat")
    router.replace(`/messages?id=${id}`)

    // Find the conversation's current unread count before clearing
    const conv = conversations.find((c: Conversation) => c.id === id)
    const clearedUnread = conv?.unread_count ?? 0

    // Optimistically clear unread_count for the selected conversation
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
    // Optimistically subtract this conversation's unread from sidebar total
    // instead of invalidating (which would refetch before markAsRead completes)
    if (clearedUnread > 0) {
      queryClient.setQueryData(
        UNREAD_COUNT_QUERY_KEY,
        (old: { count: number } | undefined) => {
          if (!old) return old
          return { count: Math.max(0, old.count - clearedUnread) }
        },
      )
    }
  }, [router, queryClient, conversations])

  const handleBack = useCallback(() => {
    setMobileView("list")
  }, [])

  const handleSend = useCallback(
    (content: string, replyToId?: string) => {
      if (!activeId) return
      const replyToInfo = replyTo && replyToId
        ? { id: replyTo.id, sender_id: replyTo.senderId, content: replyTo.content, type: replyTo.type }
        : undefined
      sendMessage.mutate({ content, type: "text", replyToId, replyToInfo })
    },
    [activeId, sendMessage, replyTo],
  )

  const handleReply = useCallback(
    (message: Message) => {
      const senderName = message.sender_id === user?.id
        ? (user?.display_name ?? "You")
        : (activeConversation?.other_user_name ?? "")
      setReplyTo({ id: message.id, senderId: message.sender_id, senderName, content: message.content, type: message.type })
    },
    [user?.id, user?.display_name, activeConversation?.other_user_name],
  )

  const clearReply = useCallback(() => setReplyTo(null), [])

  const handleSendFile = useCallback(
    (content: string, metadata: { url: string; filename: string; size: number; mime_type: string }) => {
      if (!activeId) return
      sendMessage.mutate({ content, type: "file", metadata })
    },
    [activeId, sendMessage],
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
    if (activeId) sendTyping(activeId)
  }, [activeId, sendTyping])

  const handleStartCall = useCallback(() => {
    if (!activeConversation || !callCtx) return
    callCtx.startCall(
      activeConversation.id,
      activeConversation.other_user_id,
      activeConversation.other_user_name,
    )
  }, [activeConversation, callCtx])

  const handleSendVoice = useCallback(
    (content: string, metadata: { url: string; duration: number; size: number; mime_type: string }) => {
      if (!activeId) return
      sendMessage.mutate({ content, type: "voice", metadata })
    },
    [activeId, sendMessage],
  )

  const handleReview = useCallback(
    (proposalId: string, proposalTitle: string) => {
      setReviewTarget({ proposalId, proposalTitle })
    },
    [],
  )

  const typingUserForConversation = activeId ? typingUsers[activeId] : undefined

  return (
    <div className="-mx-5 -mt-5 flex h-[calc(100vh-3.5rem)] overflow-hidden bg-white dark:bg-gray-900">
      {/* Left panel: conversation list */}
      <div
        className={cn(
          "w-full shrink-0 border-r border-gray-100 bg-white dark:border-gray-800 dark:bg-gray-900",
          "lg:w-[400px] lg:block",
          mobileView === "list" ? "block" : "hidden lg:block",
        )}
      >
        {conversationsLoading ? (
          <ConversationListSkeleton />
        ) : (
          <ConversationList
            conversations={conversations}
            activeId={activeId}
            roleFilter={roleFilter}
            searchQuery={searchQuery}
            typingUsers={typingUsers}
            onSelect={handleSelect}
            onRoleFilterChange={setRoleFilter}
            onSearchChange={setSearchQuery}
          />
        )}
      </div>

      {/* Right panel: active conversation */}
      <div
        className={cn(
          "flex min-w-0 flex-1 flex-col",
          mobileView === "chat" ? "flex" : "hidden lg:flex",
        )}
      >
        {activeConversation ? (
          <>
            <ConversationHeader
              conversation={activeConversation}
              onBack={handleBack}
              typingUserName={typingUserForConversation ? activeConversation.other_user_name : undefined}
              isConnected={isConnected}
              onStartCall={handleStartCall}
            />
            <MessageArea
              messages={allMessages}
              currentUserId={user?.id ?? ""}
              isLoading={messagesQuery.isLoading}
              hasMore={messagesQuery.hasNextPage ?? false}
              onLoadMore={() => messagesQuery.fetchNextPage()}
              onEdit={handleEdit}
              onDelete={handleDelete}
              onReply={handleReply}
              conversationId={activeId ?? ""}
              onReview={handleReview}
            />
            <MessageInput
              conversationId={activeId ?? ""}
              otherUserId={activeConversation?.other_user_id ?? ""}
              onSend={handleSend}
              onSendFile={handleSendFile}
              onSendVoice={handleSendVoice}
              onTyping={handleTyping}
              isSending={sendMessage.isPending}
              replyTo={replyTo}
              onCancelReply={clearReply}
            />
          </>
        ) : (
          <EmptyState label={t("noConversations")} />
        )}
      </div>

      {/* Review modal — opens from evaluation_request messages */}
      {reviewTarget && (
        <ReviewModal
          proposalId={reviewTarget.proposalId}
          proposalTitle={reviewTarget.proposalTitle}
          isOpen
          onClose={() => setReviewTarget(null)}
        />
      )}
    </div>
  )
}

function EmptyState({ label }: { label: string }) {
  return (
    <div className="flex flex-1 items-center justify-center bg-gray-50/50 dark:bg-gray-950/50">
      <div className="text-center">
        <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-gray-100 dark:bg-gray-800">
          <MessageSquare
            className="h-8 w-8 text-gray-300 dark:text-gray-600"
            strokeWidth={1}
          />
        </div>
        <p className="mt-4 text-sm text-gray-400 dark:text-gray-500">
          {label}
        </p>
      </div>
    </div>
  )
}

function ConversationListSkeleton() {
  return (
    <div className="flex h-full flex-col">
      <div className="px-5 pt-5 pb-3">
        <div className="h-6 w-32 animate-pulse rounded-md bg-gray-200 dark:bg-gray-700" />
      </div>
      <div className="flex gap-1.5 px-5 pb-3">
        {[1, 2, 3, 4].map((i) => (
          <div
            key={i}
            className="h-7 w-16 animate-pulse rounded-full bg-gray-200 dark:bg-gray-700"
          />
        ))}
      </div>
      <div className="px-5 pb-3">
        <div className="h-9 w-full animate-pulse rounded-lg bg-gray-200 dark:bg-gray-700" />
      </div>
      <div className="flex-1 space-y-1 overflow-hidden">
        {[1, 2, 3, 4, 5].map((i) => (
          <div key={i} className="flex items-center gap-3 px-5 py-3">
            <div className="h-10 w-10 shrink-0 animate-pulse rounded-full bg-gray-200 dark:bg-gray-700" />
            <div className="min-w-0 flex-1 space-y-2">
              <div className="h-4 w-28 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
              <div className="h-3 w-40 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
