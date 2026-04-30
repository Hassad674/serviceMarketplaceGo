/**
 * job-server.ts is the server-only fetcher used by
 * `app/[locale]/(public)/opportunities/[id]/page.tsx` to mint the
 * JSON-LD JobPosting schema (PERF-W-06) and dynamic SEO metadata.
 *
 * Returns null on any error so the route can fall back to generic
 * metadata without crashing — opportunities are core SEO surfaces
 * and a 500 here would lose Google for Jobs visibility.
 */

import { API_BASE_URL } from "@/shared/lib/api-client"
import type { JobResponse } from "../types"

export async function fetchJobForMetadata(
  jobId: string,
): Promise<JobResponse | null> {
  try {
    const res = await fetch(
      `${API_BASE_URL || "http://localhost:8080"}/api/v1/jobs/${jobId}`,
      { next: { revalidate: 120 } },
    )
    if (!res.ok) return null
    return (await res.json()) as JobResponse
  } catch {
    return null
  }
}
