"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useRouter } from "@i18n/navigation"
import { startConversation } from "../api/messaging-api"
import { CONVERSATIONS_QUERY_KEY } from "./use-conversations"

export function useStartConversation() {
  const queryClient = useQueryClient()
  const router = useRouter()

  return useMutation({
    mutationFn: ({ otherUserId, content }: { otherUserId: string; content: string }) =>
      startConversation(otherUserId, content),

    onSuccess: (response) => {
      queryClient.invalidateQueries({ queryKey: CONVERSATIONS_QUERY_KEY })
      router.push(`/messages?id=${response.conversation_id}`)
    },
  })
}
