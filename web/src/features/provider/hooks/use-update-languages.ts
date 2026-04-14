"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  updateLanguages,
  type Profile,
  type UpdateLanguagesInput,
} from "../api/profile-api"
import { profileQueryKey } from "./use-profile"

// Optimistic mutation for the two language lists (professional +
// conversational). The backend invariant is that a language never
// appears in both lists, but we trust the editor component to maintain
// that invariant before the call lands here.
export function useUpdateLanguages() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const key = profileQueryKey(uid)

  return useMutation({
    mutationFn: (input: UpdateLanguagesInput) => updateLanguages(input),
    onMutate: async (input) => {
      await queryClient.cancelQueries({ queryKey: key })
      const previous = queryClient.getQueryData<Profile>(key)
      if (previous) {
        queryClient.setQueryData<Profile>(key, {
          ...previous,
          languages_professional: input.professional,
          languages_conversational: input.conversational,
        })
      }
      return { previous }
    },
    onError: (_error, _input, context) => {
      if (context?.previous) {
        queryClient.setQueryData<Profile>(key, context.previous)
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: key })
    },
  })
}
