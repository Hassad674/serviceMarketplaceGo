"use client"

import { keepPreviousData, useQuery } from "@tanstack/react-query"
import { searchSkillsAutocomplete } from "../api/skill-api"
import {
  SKILLS_QUERY_KEY,
  SKILL_AUTOCOMPLETE_DEBOUNCE_MS,
} from "../constants"
import { useDebouncedValue } from "./use-debounced-value"

// Debounced autocomplete query. The caller passes the raw input
// value; we debounce internally and drive the query key with the
// debounced value so React Query deduplicates identical searches.
// `keepPreviousData` prevents the dropdown from flickering to empty
// between keystrokes.
export function useSkillAutocomplete(rawQuery: string) {
  const query = useDebouncedValue(
    rawQuery.trim(),
    SKILL_AUTOCOMPLETE_DEBOUNCE_MS,
  )

  return useQuery({
    queryKey: SKILLS_QUERY_KEY.autocomplete(query),
    queryFn: () => searchSkillsAutocomplete(query),
    enabled: query.length >= 1,
    placeholderData: keepPreviousData,
    staleTime: 30 * 1000,
  })
}
