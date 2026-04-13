"use client"

import { useQuery } from "@tanstack/react-query"
import { fetchProfileSkills } from "../api/skill-api"
import { SKILLS_QUERY_KEY } from "../constants"

// Thin TanStack Query wrapper around GET /api/v1/profile/skills.
// The stale time mirrors `useProfile`: the current operator's own
// data rarely changes out-of-band, so 5 minutes is generous.
export function useProfileSkills() {
  return useQuery({
    queryKey: SKILLS_QUERY_KEY.profile,
    queryFn: () => fetchProfileSkills(),
    staleTime: 5 * 60 * 1000,
  })
}
