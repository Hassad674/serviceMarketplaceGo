"use client"

import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { listMessages, sendMessage, editMessage, deleteMessage } from "../api/messaging-api"
import type { FileMessageMetadata, VoiceMessageMetadata } from "../api/messaging-api"
import type { Message, MessageListResponse } from "../types"
import { conversationsQueryKey } from "./use-conversations"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

export const MESSAGES_KEY_BASE = "messaging-messages"

export function messagesQueryKey(uid: string | undefined, conversationId: string | null) {
  return ["user", uid, MESSAGES_KEY_BASE, conversationId] as const
}

/** @deprecated Use messagesQueryKey(uid, conversationId) instead */
export const MESSAGES_QUERY_KEY = MESSAGES_KEY_BASE

export function useMessages(conversationId: string | null) {
  const uid = useCurrentUserId()

  return useInfiniteQuery({
    queryKey: messagesQueryKey(uid, conversationId),
    queryFn: ({ pageParam }) =>
      listMessages(conversationId!, pageParam as string | undefined),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage.has_more ? lastPage.next_cursor : undefined,
    enabled: !!conversationId,
    staleTime: 30 * 1000,
  })
}

export function useSendMessage(conversationId: string | null) {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: ({
      content,
      type,
      metadata,
      replyToId,
    }: {
      content: string
      type?: "text" | "file" | "voice"
      metadata?: FileMessageMetadata | VoiceMessageMetadata
      replyToId?: string
      replyToInfo?: { id: string; sender_id: string; content: string; type: string }
    }) => sendMessage(conversationId!, content, type, metadata, replyToId),

    onMutate: async ({ content, type = "text", metadata, replyToInfo }) => {
      const queryKey = messagesQueryKey(uid, conversationId)
      await queryClient.cancelQueries({ queryKey })

      const previous = queryClient.getQueryData<{
        pages: MessageListResponse[]
        pageParams: (string | undefined)[]
      }>(queryKey)

      const optimisticMessage: Message = {
        id: `temp-${Date.now()}`,
        conversation_id: conversationId!,
        sender_id: "optimistic",
        content,
        type,
        metadata: metadata ?? null,
        reply_to: replyToInfo ?? null,
        seq: 0,
        status: "sending",
        edited_at: null,
        deleted_at: null,
        created_at: new Date().toISOString(),
      }

      queryClient.setQueryData(queryKey, (old: typeof previous) => {
        if (!old) {
          return {
            pages: [{ data: [optimisticMessage], has_more: false }],
            pageParams: [undefined],
          }
        }
        // Prepend to page 0 (newest page, DESC order) so that after
        // chronological reversal the optimistic message appears at the bottom.
        const newPages = [...old.pages]
        newPages[0] = {
          ...newPages[0],
          data: [optimisticMessage, ...newPages[0].data],
        }
        return { ...old, pages: newPages }
      })

      return { previous }
    },

    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(
          messagesQueryKey(uid, conversationId),
          context.previous,
        )
      }
    },

    onSuccess: (newMessage) => {
      queryClient.setQueryData(
        messagesQueryKey(uid, conversationId),
        (old: { pages: MessageListResponse[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old
          const newPages = old.pages.map((page, idx) => {
            if (idx !== 0) return page
            return {
              ...page,
              data: page.data.map((msg) =>
                msg.id.startsWith("temp-") ? newMessage : msg,
              ),
            }
          })
          return { ...old, pages: newPages }
        },
      )
      queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
    },
  })
}

export function useEditMessage(conversationId: string | null) {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: ({ messageId, content }: { messageId: string; content: string }) =>
      editMessage(messageId, content),

    onSuccess: (updatedMessage) => {
      queryClient.setQueryData(
        messagesQueryKey(uid, conversationId),
        (old: { pages: MessageListResponse[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              data: page.data.map((msg) =>
                msg.id === updatedMessage.id ? updatedMessage : msg,
              ),
            })),
          }
        },
      )
      queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
    },
  })
}

export function useDeleteMessage(conversationId: string | null) {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (messageId: string) => deleteMessage(messageId),

    onSuccess: (_data, messageId) => {
      queryClient.setQueryData(
        messagesQueryKey(uid, conversationId),
        (old: { pages: MessageListResponse[]; pageParams: (string | undefined)[] } | undefined) => {
          if (!old) return old
          return {
            ...old,
            pages: old.pages.map((page) => ({
              ...page,
              data: page.data.map((msg) =>
                msg.id === messageId
                  ? { ...msg, deleted_at: new Date().toISOString(), content: "" }
                  : msg,
              ),
            })),
          }
        },
      )
      queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
    },
  })
}
