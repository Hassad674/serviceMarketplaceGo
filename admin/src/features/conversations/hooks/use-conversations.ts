import { useQuery } from "@tanstack/react-query"
import {
  listConversations,
  getConversation,
  getConversationMessages,
} from "../api/conversations-api"

type ConversationsQueryParams = {
  cursor: string
  sort: string
  filter: string
}

export function conversationsQueryKey(params: ConversationsQueryParams) {
  return ["admin", "conversations", params] as const
}

export function useConversations(params: ConversationsQueryParams) {
  return useQuery({
    queryKey: conversationsQueryKey(params),
    queryFn: () => listConversations({
      cursor: params.cursor || undefined,
      sort: params.sort || undefined,
      filter: params.filter || undefined,
    }),
    staleTime: 30 * 1000,
  })
}

export function conversationQueryKey(id: string) {
  return ["admin", "conversations", id] as const
}

export function useConversation(id: string) {
  return useQuery({
    queryKey: conversationQueryKey(id),
    queryFn: () => getConversation(id),
    enabled: !!id,
    staleTime: 60 * 1000,
  })
}

export function conversationMessagesQueryKey(id: string, cursor: string) {
  return ["admin", "conversations", id, "messages", { cursor }] as const
}

export function useConversationMessages(id: string, cursor: string) {
  return useQuery({
    queryKey: conversationMessagesQueryKey(id, cursor),
    queryFn: () => getConversationMessages(id, cursor || undefined),
    enabled: !!id,
    staleTime: 30 * 1000,
  })
}
