"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import { createUserSkill } from "../api/skill-api"
import { SKILLS_QUERY_KEY } from "../constants"

// POST /api/v1/skills wrapper. On success we invalidate every
// autocomplete query so the newly created skill starts appearing in
// search results immediately. The mutation returns the canonical
// `SkillResponse` (with the server-normalised `skill_text`), which
// callers use to add the entry to their local selection.
export function useCreateUserSkill() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (displayText: string) => createUserSkill(displayText),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["skills", "autocomplete"],
      })
      queryClient.invalidateQueries({
        queryKey: SKILLS_QUERY_KEY.profile,
      })
    },
  })
}
