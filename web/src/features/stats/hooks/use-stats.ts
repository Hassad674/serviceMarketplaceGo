"use client"

import { useQuery } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  fetchEnterpriseApplicationsStats,
  fetchKeywordStats,
  fetchVisibilityStats,
  type StatsPeriodDays,
} from "../api/stats-api"

// Stats hooks — TanStack Query wrappers around the /me/stats/*
// endpoints. All keyed off the current user so a logout clears the
// cache automatically. A 5-minute staleTime is correct here:
//   * tracking middleware writes happen at most a few times per
//     visitor — the dashboard does not need second-by-second freshness;
//   * the backend caps each query at <100 ms, so a stale read costs
//     nothing in user-perceived latency vs. a refetch;
//   * cuts the contribution to the global IP rate limit when 4-5
//     widgets read the same org's stats in parallel.
//
// `enabled: false` lets a caller withhold the request when the user
// role does not need the corresponding dataset (e.g. an Enterprise
// reading visibility stats — the backend would 200 with zeros, but
// firing the request is wasteful).

const STATS_STALE_TIME_MS = 5 * 60 * 1000
const STATS_GC_TIME_MS = 15 * 60 * 1000

export function visibilityStatsKey(
  uid: string | undefined,
  days: StatsPeriodDays,
) {
  return ["user", uid, "stats", "visibility", days] as const
}

export function keywordStatsKey(
  uid: string | undefined,
  days: StatsPeriodDays,
  limit: number,
) {
  return ["user", uid, "stats", "keywords", days, limit] as const
}

export function applicationsStatsKey(
  uid: string | undefined,
  days: StatsPeriodDays,
) {
  return ["user", uid, "stats", "enterprise-applications", days] as const
}

export interface UseStatsOptions {
  enabled?: boolean
}

export function useVisibilityStats(
  days: StatsPeriodDays,
  options: UseStatsOptions = {},
) {
  const uid = useCurrentUserId()
  return useQuery({
    queryKey: visibilityStatsKey(uid, days),
    queryFn: ({ signal }) => fetchVisibilityStats(days, signal),
    staleTime: STATS_STALE_TIME_MS,
    gcTime: STATS_GC_TIME_MS,
    enabled: Boolean(uid) && (options.enabled ?? true),
    refetchOnWindowFocus: false,
  })
}

export function useKeywordStats(
  days: StatsPeriodDays,
  limit = 10,
  options: UseStatsOptions = {},
) {
  const uid = useCurrentUserId()
  return useQuery({
    queryKey: keywordStatsKey(uid, days, limit),
    queryFn: ({ signal }) => fetchKeywordStats(days, limit, signal),
    staleTime: STATS_STALE_TIME_MS,
    gcTime: STATS_GC_TIME_MS,
    enabled: Boolean(uid) && (options.enabled ?? true),
    refetchOnWindowFocus: false,
  })
}

export function useApplicationsStats(
  days: StatsPeriodDays,
  options: UseStatsOptions = {},
) {
  const uid = useCurrentUserId()
  return useQuery({
    queryKey: applicationsStatsKey(uid, days),
    queryFn: ({ signal }) => fetchEnterpriseApplicationsStats(days, signal),
    staleTime: STATS_STALE_TIME_MS,
    gcTime: STATS_GC_TIME_MS,
    enabled: Boolean(uid) && (options.enabled ?? true),
    refetchOnWindowFocus: false,
  })
}
