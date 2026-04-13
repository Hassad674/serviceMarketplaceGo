"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { updateProfileSkills } from "../api/skill-api"
import { SKILLS_QUERY_KEY } from "../constants"
import type { ProfileSkillResponse } from "../types"

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
    },
  })
}
