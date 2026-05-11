import { apiClient } from "@/shared/lib/api-client"

// Stats API client — wraps the /me/stats/* endpoints exposed by the
// backend (agent A landing in commit 5715cf42). The backend's OpenAPI
// schema types every payload as `{[key:string]:unknown}`, so we
// duplicate the contract types here as the frontend's source of
// truth. They MUST stay in lockstep with `internal/handler/stats_*.go`.
//
// Empty arrays (never null) come back from the API per the contract;
// callers can treat them as terminal "no data" states without
// defensive null checks.

export type StatsPeriodDays = 7 | 30 | 90 | 365

export interface StatsTimeBucket {
  date: string // RFC3339 day boundary (UTC midnight)
  count: number
  // unique fingerprint count for this day. Always present after D3
  // (defaults to count when the source can't deduplicate, e.g. job
  // applications). Marked optional for backwards-compat with cached
  // responses captured before the contract bump.
  unique?: number
}

export interface VisibilityStats {
  organization_id: string
  period_days: number
  total_views: number
  unique_viewers: number
  search_appearances: number
  avg_search_position: number | null
  series: StatsTimeBucket[]
}

export interface KeywordStat {
  keyword: string
  count: number
  avg_position: number | null
}

export interface EnterpriseApplicationsStats {
  organization_id: string
  period_days: number
  total_count: number
  series: StatsTimeBucket[]
}

interface DataEnvelope<T> {
  data: T
}

export async function fetchVisibilityStats(
  days: StatsPeriodDays,
  signal?: AbortSignal,
): Promise<VisibilityStats> {
  const res = await apiClient<DataEnvelope<VisibilityStats>>(
    `/api/v1/me/stats/visibility?days=${days}`,
    { signal },
  )
  return normaliseVisibility(res.data)
}

export async function fetchKeywordStats(
  days: StatsPeriodDays,
  limit = 10,
  signal?: AbortSignal,
): Promise<KeywordStat[]> {
  const safeLimit = Math.max(1, Math.min(100, Math.trunc(limit)))
  const res = await apiClient<DataEnvelope<KeywordStat[]>>(
    `/api/v1/me/stats/keywords?days=${days}&limit=${safeLimit}`,
    { signal },
  )
  return Array.isArray(res.data) ? res.data : []
}

export async function fetchEnterpriseApplicationsStats(
  days: StatsPeriodDays,
  signal?: AbortSignal,
): Promise<EnterpriseApplicationsStats> {
  const res = await apiClient<DataEnvelope<EnterpriseApplicationsStats>>(
    `/api/v1/me/stats/enterprise-applications?days=${days}`,
    { signal },
  )
  return normaliseApplications(res.data)
}

// normaliseVisibility coerces undefined / missing fields into safe
// defaults so the consumer code can treat the response as a
// fully-populated value without per-field null checks. Defensive
// against backend regressions that drop fields silently.
function normaliseVisibility(raw: VisibilityStats): VisibilityStats {
  return {
    organization_id: raw.organization_id ?? "",
    period_days: raw.period_days ?? 0,
    total_views: raw.total_views ?? 0,
    unique_viewers: raw.unique_viewers ?? 0,
    search_appearances: raw.search_appearances ?? 0,
    avg_search_position:
      typeof raw.avg_search_position === "number" ? raw.avg_search_position : null,
    series: Array.isArray(raw.series) ? raw.series : [],
  }
}

function normaliseApplications(
  raw: EnterpriseApplicationsStats,
): EnterpriseApplicationsStats {
  return {
    organization_id: raw.organization_id ?? "",
    period_days: raw.period_days ?? 0,
    total_count: raw.total_count ?? 0,
    series: Array.isArray(raw.series) ? raw.series : [],
  }
}
