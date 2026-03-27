"use client"

import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { listMessages, sendMessage, editMessage, deleteMessage } from "../api/messaging-api"
import type { FileMessageMetadata, VoiceMessageMetadata } from "../api/messaging-api"
import type { Message, MessageListResponse } from "../types"
import { CONVERSATIONS_QUERY_KEY } from "./use-conversations"

export const MESSAGES_QUERY_KEY = "messaging-messages"

export function useMessages(conversationId: string | null) {
  return useInfiniteQuery({
    queryKey: [MESSAGES_QUERY_KEY, conversationId],
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
    }) => sendMessage(conversationId!, content, type, metadata, replyToId),

    onMutate: async ({ content, type = "text", metadata }) => {
      const queryKey = [MESSAGES_QUERY_KEY, conversationId]
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
          [MESSAGES_QUERY_KEY, conversationId],
          context.previous,
        )
      }
    },

    onSuccess: (newMessage) => {
      queryClient.setQueryData(
        [MESSAGES_QUERY_KEY, conversationId],
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
      queryClient.invalidateQueries({ queryKey: CONVERSATIONS_QUERY_KEY })
    },
  })
}

export function useEditMessage(conversationId: string | null) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ messageId, content }: { messageId: string; content: string }) =>
      editMessage(messageId, content),

    onSuccess: (updatedMessage) => {
      queryClient.setQueryData(
        [MESSAGES_QUERY_KEY, conversationId],
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
      queryClient.invalidateQueries({ queryKey: CONVERSATIONS_QUERY_KEY })
    },
  })
}

export function useDeleteMessage(conversationId: string | null) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (messageId: string) => deleteMessage(messageId),

    onSuccess: (_data, messageId) => {
      queryClient.setQueryData(
        [MESSAGES_QUERY_KEY, conversationId],
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
      queryClient.invalidateQueries({ queryKey: CONVERSATIONS_QUERY_KEY })
    },
  })
}
