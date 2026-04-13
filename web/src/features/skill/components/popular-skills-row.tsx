"use client"

import { useMemo } from "react"
import { useQueries } from "@tanstack/react-query"
import { fetchCatalog } from "../api/skill-api"
import { POPULAR_SKILLS_LIMIT, SKILLS_QUERY_KEY } from "../constants"
import type { CatalogResponse, SkillResponse } from "../types"

interface PopularSkillsRowProps {
  expertiseKeys: string[]
  alreadySelected: Set<string>
  onAdd: (skill: SkillResponse) => void
  limit?: number
}

// Row of the highest-usage skills across the user's expertise
// domains. Uses `useQueries` so each expertise fetches in parallel
// with the same cache keys as `useSkillCatalog`, which means the
// data is shared with the collapsible panels below — switching a
// panel open reuses the cache entry already populated here.
export function PopularSkillsRow({
  expertiseKeys,
  alreadySelected,
  onAdd,
  limit = POPULAR_SKILLS_LIMIT,
}: PopularSkillsRowProps) {
  const queries = useQueries({
    queries: expertiseKeys.map((key) => ({
      queryKey: SKILLS_QUERY_KEY.catalog(key),
      queryFn: () => fetchCatalog(key),
      staleTime: 10 * 60 * 1000,
    })),
  })

  const popularSkills = useMemo(
    () =>
      flattenAndRank(queries.map((q) => q.data), alreadySelected).slice(
        0,
        limit,
      ),
    [queries, alreadySelected, limit],
  )

  if (popularSkills.length === 0) return null

  return (
    <div className="flex flex-wrap gap-2">
      {popularSkills.map((skill) => (
        <button
          key={skill.skill_text}
          type="button"
          onClick={() => onAdd(skill)}
          className="inline-flex items-center gap-1.5 rounded-full border border-border bg-background px-3 py-1 text-xs font-medium text-foreground transition-colors duration-150 hover:border-primary/60 hover:bg-primary/5 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
        >
          {skill.display_text}
        </button>
      ))}
    </div>
  )
}

// Merge catalog responses from several expertise domains, drop any
// skill the user already has, deduplicate by `skill_text`, and sort
// by descending `usage_count` so the hottest skills surface first.
function flattenAndRank(
  catalogs: Array<CatalogResponse | undefined>,
  alreadySelected: Set<string>,
): SkillResponse[] {
  const byText = new Map<string, SkillResponse>()
  for (const catalog of catalogs) {
    if (!catalog) continue
    for (const skill of catalog.skills) {
      if (alreadySelected.has(skill.skill_text)) continue
      const existing = byText.get(skill.skill_text)
      if (!existing || skill.usage_count > existing.usage_count) {
        byText.set(skill.skill_text, skill)
      }
    }
  }
  return Array.from(byText.values()).sort(
    (a, b) => b.usage_count - a.usage_count,
  )
}
