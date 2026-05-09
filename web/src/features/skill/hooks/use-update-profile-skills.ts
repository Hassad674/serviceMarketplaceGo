"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { updateProfileSkills } from "../api/skill-api"
import { SKILLS_QUERY_KEY } from "../constants"
import type { ProfileSkillResponse } from "../types"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import { profileCompletionQueryKey } from "@/features/profile-completion/hooks/use-profile-completion"

// Optimistic mutation for the current operator's profile skill list.
//
// Flow:
//   1. Snapshot the current cache entry.
//   2. Patch the cache with the new ordered skill list so the UI
//      flips to the new state immediately.
//   3. Send the PUT. On error we restore the snapshot so the editor
//      reverts and can surface the failure. On success we mark the
//      query as stale and let it refetch for canonical truth
//      (display_text can differ if the backend normalised anything).
//
// We patch with a locally-built array using the input `skill_texts`
// because the PUT endpoint returns only `{ status: "ok" }`. The
// optimistic display_text is looked up from the snapshot when
// possible, otherwise falls back to the raw skill_text — the refetch
// in `onSettled` fixes any gaps.
export function useUpdateProfileSkills() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  const key = SKILLS_QUERY_KEY.profile

  return useMutation({
    mutationFn: (skillTexts: string[]) => updateProfileSkills(skillTexts),
    onMutate: async (skillTexts) => {
      await queryClient.cancelQueries({ queryKey: key })
      const previous =
        queryClient.getQueryData<ProfileSkillResponse[]>(key) ?? []
      const displayLookup = new Map(
        previous.map((entry) => [entry.skill_text, entry.display_text]),
      )
      const optimistic: ProfileSkillResponse[] = skillTexts.map(
        (skillText, index) => ({
          skill_text: skillText,
          display_text: displayLookup.get(skillText) ?? skillText,
          position: index,
        }),
      )
      queryClient.setQueryData<ProfileSkillResponse[]>(key, optimistic)
      return { previous }
    },
    onError: (_error, _skillTexts, context) => {
      if (context?.previous) {
        queryClient.setQueryData<ProfileSkillResponse[]>(key, context.previous)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: key })
      // Skills are tracked by the freelance + agency completion
      // checklists — refresh the bar on every save (success or
      // rollback so a user-visible refetch always brings the count
      // back in sync with the backend).
      queryClient.invalidateQueries({
        queryKey: profileCompletionQueryKey(uid),
      })
    },
  })
}
