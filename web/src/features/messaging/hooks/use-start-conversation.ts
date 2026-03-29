"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useRouter } from "@i18n/navigation"
import { startConversation } from "../api/messaging-api"
import { conversationsQueryKey } from "./use-conversations"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

export function useStartConversation() {
  const queryClient = useQueryClient()
  const router = useRouter()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: ({ otherUserId, content }: { otherUserId: string; content: string }) =>
      startConversation(otherUserId, content),

    onSuccess: (response) => {
      queryClient.invalidateQueries({ queryKey: conversationsQueryKey(uid) })
      router.push(`/messages?id=${response.conversation_id}`)
    },
  })
}
