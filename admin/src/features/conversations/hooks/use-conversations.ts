import { useQuery } from "@tanstack/react-query"
import {
  listConversations,
  getConversation,
  getConversationMessages,
} from "../api/conversations-api"

export function conversationsQueryKey(cursor: string) {
  return ["admin", "conversations", { cursor }] as const
}

export function useConversations(cursor: string) {
  return useQuery({
    queryKey: conversationsQueryKey(cursor),
    queryFn: () => listConversations(cursor || undefined),
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
