"use client"

import { useState, useCallback, useEffect, useRef, useMemo } from "react"
import { MessageSquare } from "lucide-react"
import { useTranslations } from "next-intl"
import { useSearchParams } from "next/navigation"
import { useRouter } from "@i18n/navigation"
import { useQueryClient } from "@tanstack/react-query"
import { cn } from "@/shared/lib/utils"
import { useUser, useOrganization } from "@/shared/hooks/use-user"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { useCallContext } from "@/shared/hooks/use-call-context"
import { ConversationList } from "./conversation-list"
import { ConversationHeader } from "./conversation-header"
import { MessageArea } from "./message-area"
import { MessageInput } from "./message-input"
import { useConversations, conversationsQueryKey } from "../hooks/use-conversations"
import { useMessages, useSendMessage, useEditMessage, useDeleteMessage } from "../hooks/use-messages"
import { useMessagingWS } from "../hooks/use-messaging-ws"
import { markAsRead } from "../api/messaging-api"
import { unreadCountQueryKey } from "@/shared/hooks/use-unread-count"
import { ReviewModal } from "@/shared/components/review/review-modal"
import { deriveReviewSide } from "@/shared/lib/review/derive-side"
import { ReportDialog } from "@/shared/components/reporting/report-dialog"
import type { Conversation, ConversationListResponse, Message } from "../types"
import type { ReviewSide } from "@/shared/types/review"

export function MessagingPage() {
  const t = useTranslations("messaging")
  const searchParams = useSearchParams()
  const router = useRouter()
  const { data: user } = useUser()
  const { data: org } = useOrganization()

  const [activeId, setActiveId] = useState<string | null>(
    searchParams.get("id"),
  )
  const [orgTypeFilter, setOrgTypeFilter] = useState<"all" | string>("all")
  const [searchQuery, setSearchQuery] = useState("")
  // Mobile view defaults to "chat" when we deep-linked into a specific
  // conversation (so the user sees the chat, not the list, on small
  // screens). Computing this in the lazy initializer keeps the original
  // intent without a setState-in-effect bootstrap.
  const [mobileView, setMobileView] = useState<"list" | "chat">(() =>
    searchParams.get("id") ? "chat" : "list",
  )
  const [replyTo, setReplyTo] = useState<{ id: string; senderId: string; senderName: string; content: string; type: string } | null>(null)
  const [reviewTarget, setReviewTarget] = useState<{
    proposalId: string
    proposalTitle: string
    side: ReviewSide
  } | null>(null)
  const [reportTarget, setReportTarget] = useState<{ type: "message" | "user"; id: string } | null>(null)

  const canSendMessage = useHasPermission("messaging.send")
  const callCtx = useCallContext()

  const { data: conversationsData, isLoading: conversationsLoading } = useConversations()
  const messagesQuery = useMessages(activeId)
  const sendMessage = useSendMessage(activeId)
  const editMessageMut = useEditMessage(activeId)
  const deleteMessageMut = useDeleteMessage(activeId)
  const { typingUsers, sendTyping, isConnected, setActiveConversationId } = useMessagingWS(user?.id)

  const queryClient = useQueryClient()

  // Memo'd so the empty-array fallback keeps a stable identity and the
  // downstream useEffect/useCallback dep arrays don't churn.
  const conversations = useMemo(
    () => conversationsData?.data ?? [],
    [conversationsData?.data],
  )
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
  // Memo'd by the underlying pages reference so downstream useMemo dep
  // arrays don't churn on every parent render.
  const allMessages = useMemo(() => {
    if (!messagesQuery.data) return []
    return [...messagesQuery.data.pages]
      .reverse()
      .flatMap((page) => [...page.data].reverse())
  }, [messagesQuery.data])

  // Deep-link from query param is now handled entirely via lazy state
  // initializers above (activeId + mobileView read searchParams.get("id")
  // on mount). Subsequent conversation switches go through handleSelect,
  // which updates both activeId and the URL — no effect needed.

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
    // Optimistically subtract this conversation's unread from sidebar total
    // instead of invalidating (which would refetch before markAsRead completes)
    if (clearedUnread > 0) {
      queryClient.setQueryData(
        unreadCountQueryKey(user?.id),
        (old: { count: number } | undefined) => {
          if (!old) return old
          return { count: Math.max(0, old.count - clearedUnread) }
        },
      )
    }
  }, [router, queryClient, conversations, user?.id])

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
        : (activeConversation?.other_org_name ?? "")
      setReplyTo({ id: message.id, senderId: message.sender_id, senderName, content: message.content, type: message.type })
    },
    [user?.id, user?.display_name, activeConversation?.other_org_name],
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

  const handleStartCall = useCallback((callType: "audio" | "video" = "audio") => {
    if (!activeConversation || !callCtx) return
    // Calls still anchor on user ids — we target the other participant
    // user directly, not the org. Org-level routing is only for message
    // threads and display metadata.
    callCtx.startCall(
      activeConversation.id,
      activeConversation.other_user_id,
      activeConversation.other_org_name,
      callType,
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
    (
      proposalId: string,
      proposalTitle: string,
      participants: { clientOrganizationId: string; providerOrganizationId: string },
    ) => {
      // Double-blind reviews: both parties see the single shared
      // evaluation_request system message. We derive which side the
      // current viewer is on by comparing their organization id with
      // the proposal's client/provider organization ids (enriched by
      // the backend into the message metadata). If the viewer is
      // neither side (shouldn't normally happen outside of admin
      // snooping), we silently drop the click — the modal would just
      // show a forbidden error.
      const side = deriveReviewSide(org?.id, {
        client_id: participants.clientOrganizationId,
        provider_id: participants.providerOrganizationId,
      })
      if (!side) return
      setReviewTarget({ proposalId, proposalTitle, side })
    },
    [org?.id],
  )

  const handleReportMessage = useCallback(
    (messageId: string) => {
      setReportTarget({ type: "message", id: messageId })
    },
    [],
  )

  const handleReportUser = useCallback(() => {
    if (activeConversation) {
      setReportTarget({ type: "user", id: activeConversation.other_org_id })
    }
  }, [activeConversation])

  const typingUserForConversation = activeId ? typingUsers[activeId] : undefined

  return (
    <div className="-mx-5 -mt-5 flex h-[calc(100vh-3.5rem)] overflow-hidden bg-card">
      {/* Left panel: conversation list */}
      <div
        className={cn(
          "w-full shrink-0 border-r border-border bg-card",
          "lg:w-[360px] lg:block",
          mobileView === "list" ? "block" : "hidden lg:block",
        )}
      >
        {conversationsLoading ? (
          <ConversationListSkeleton />
        ) : (
          <ConversationList
            conversations={conversations}
            activeId={activeId}
            orgTypeFilter={orgTypeFilter}
            searchQuery={searchQuery}
            typingUsers={typingUsers}
            onSelect={handleSelect}
            onOrgTypeFilterChange={setOrgTypeFilter}
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
              currentOrgType={org?.type}
              onBack={handleBack}
              typingUserName={typingUserForConversation ? activeConversation.other_org_name : undefined}
              isConnected={isConnected}
              onStartCall={handleStartCall}
              onReportUser={handleReportUser}
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
              onReport={handleReportMessage}
              conversationId={activeId ?? ""}
              onReview={handleReview}
            />
            {canSendMessage ? (
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
            ) : (
              <NoSendPermissionBar />
            )}
          </>
        ) : (
          <EmptyState label={t("noConversations")} />
        )}
      </div>

      {/* Review modal — opens from evaluation_request messages.
          Since double-blind reviews, both client and provider can
          trigger this; `side` is derived from the viewer's org. */}
      {reviewTarget && (
        <ReviewModal
          proposalId={reviewTarget.proposalId}
          proposalTitle={reviewTarget.proposalTitle}
          side={reviewTarget.side}
          isOpen
          onClose={() => setReviewTarget(null)}
        />
      )}

      {/* Report dialog — opens from message context menu or header */}
      {reportTarget && (
        <ReportDialog
          open={!!reportTarget}
          onClose={() => setReportTarget(null)}
          targetType={reportTarget.type}
          targetId={reportTarget.id}
          conversationId={activeId ?? ""}
        />
      )}
    </div>
  )
}

function EmptyState({ label }: { label: string }) {
  const t = useTranslations("messaging")
  return (
    <div className="flex flex-1 items-center justify-center bg-background">
      <div className="mx-4 max-w-md rounded-2xl border border-border bg-card px-8 py-10 text-center shadow-[0_4px_24px_rgba(42,31,21,0.04)]">
        <span className="mx-auto flex h-16 w-16 items-center justify-center rounded-2xl bg-primary-soft text-primary-deep">
          <MessageSquare className="h-7 w-7" strokeWidth={1.5} />
        </span>
        <h2 className="mt-5 font-serif text-[22px] font-medium leading-tight text-foreground">
          {t("messaging_w21_emptyTitle")}
        </h2>
        <p className="mt-2 text-sm text-muted-foreground">
          {t("messaging_w21_emptyBody")}
        </p>
        <p className="mt-4 text-xs text-muted-foreground">{label}</p>
      </div>
    </div>
  )
}

function NoSendPermissionBar() {
  const t = useTranslations("permissions")
  return (
    <div className="flex items-center justify-center border-t border-border bg-background px-4 py-3">
      <p className="text-sm text-muted-foreground">{t("noMessagingSend")}</p>
    </div>
  )
}

function ConversationListSkeleton() {
  return (
    <div className="flex h-full flex-col bg-card">
      <div className="px-5 pb-3 pt-5">
        <div className="h-6 w-32 animate-pulse rounded-md bg-border" />
      </div>
      <div className="flex gap-1.5 px-5 pb-3">
        {[1, 2, 3, 4].map((i) => (
          <div
            key={i}
            className="h-7 w-16 animate-pulse rounded-full bg-border"
          />
        ))}
      </div>
      <div className="px-5 pb-3">
        <div className="h-10 w-full animate-pulse rounded-xl bg-border" />
      </div>
      <div className="flex-1 space-y-1 overflow-hidden">
        {[1, 2, 3, 4, 5].map((i) => (
          <div key={i} className="flex items-center gap-3 px-5 py-3">
            <div className="h-10 w-10 shrink-0 animate-pulse rounded-full bg-border" />
            <div className="min-w-0 flex-1 space-y-2">
              <div className="h-4 w-28 animate-pulse rounded bg-border" />
              <div className="h-3 w-40 animate-pulse rounded bg-border" />
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
