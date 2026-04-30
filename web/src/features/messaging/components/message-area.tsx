"use client"

import { useMemo } from "react"
import { MessageSquare } from "lucide-react"
import { useTranslations } from "next-intl"
import type { Message, ProposalMessageMetadata } from "../types"
import { useMessageScroll } from "../hooks/use-message-scroll"
import { MessageAreaSkeleton } from "./message-area-skeleton"
import { MessageBubble } from "./message-bubble"
import {
  computeResolvedCompletionIds,
  computeSupersededIds,
  isProposalMetadata,
} from "./message-area-utils"

interface MessageAreaProps {
  messages: Message[]
  currentUserId: string
  isLoading: boolean
  hasMore: boolean
  onLoadMore: () => void
  onEdit: (messageId: string, content: string) => void
  onDelete: (messageId: string) => void
  onReply: (message: Message) => void
  onReport?: (messageId: string) => void
  conversationId: string
  // participants are ORG ids from the proposal's system message
  // metadata. The page uses them together with the viewer's org to
  // derive the review side.
  onReview?: (
    proposalId: string,
    proposalTitle: string,
    participants: { clientOrganizationId: string; providerOrganizationId: string },
  ) => void
}

/**
 * MessageArea — the scrollable timeline of a single conversation.
 *
 * Owns:
 *   - scroll-to-bottom on append + intersection-observer for older
 *     messages (delegated to the `useMessageScroll` hook).
 *   - filtering of stale "completion_requested" cards once a
 *     subsequent system message resolves them.
 *   - dispatching each message to the right `MessageBubble` renderer.
 *
 * Does NOT own:
 *   - the actual rendering of any message type (delegated to
 *     `MessageBubble` and its sub-components).
 *   - the scroll/observer wiring (in `useMessageScroll`).
 */
export function MessageArea({
  messages,
  currentUserId,
  isLoading,
  hasMore,
  onLoadMore,
  onEdit,
  onDelete,
  onReply,
  onReport,
  conversationId,
  onReview,
}: MessageAreaProps) {
  const t = useTranslations("messaging")
  const { scrollRef, topSentinelRef } = useMessageScroll({
    messages,
    hasMore,
    onLoadMore,
  })

  // Stale-state derivations: BOTH useMemo calls must be declared
  // BEFORE any early return to satisfy Rules of Hooks.
  const supersededProposalIds = useMemo(
    () => computeSupersededIds(messages),
    [messages],
  )
  const resolvedCompletionProposalIds = useMemo(
    () => computeResolvedCompletionIds(messages),
    [messages],
  )

  if (isLoading) {
    return <MessageAreaSkeleton />
  }

  if (messages.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <div className="text-center">
          <MessageSquare
            className="mx-auto h-12 w-12 text-slate-200 dark:text-slate-700"
            strokeWidth={1}
          />
          <p className="mt-3 text-sm text-slate-400 dark:text-slate-500">
            {t("noMessages")}
          </p>
        </div>
      </div>
    )
  }

  const visibleMessages = messages.filter((message) => {
    if (
      message.type === "proposal_completion_requested" &&
      isProposalMetadata(message.metadata)
    ) {
      const meta = message.metadata as ProposalMessageMetadata
      return !resolvedCompletionProposalIds.has(meta.proposal_id)
    }
    return true
  })

  return (
    <div ref={scrollRef} className="flex-1 overflow-y-auto px-5 py-4">
      <div className="mx-auto flex max-w-4xl flex-col gap-3">
        {hasMore && (
          <div ref={topSentinelRef} className="flex justify-center py-2">
            <button
              onClick={onLoadMore}
              className="text-xs text-slate-400 transition-colors hover:text-slate-600 dark:text-slate-500 dark:hover:text-slate-300"
            >
              {t("loadMore")}
            </button>
          </div>
        )}

        {visibleMessages.map((message) => (
          <MessageBubble
            key={message.id}
            message={message}
            state={{
              isOwn:
                message.sender_id === currentUserId ||
                message.sender_id === "optimistic",
              currentUserId,
              conversationId,
              supersededProposalIds,
            }}
            actions={{
              onEdit,
              onDelete,
              onReply,
              onReport,
              onReview,
            }}
          />
        ))}
      </div>
    </div>
  )
}
