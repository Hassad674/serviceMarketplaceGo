/**
 * sitemap-server.ts is the server-only fetcher used by
 * `app/sitemap.ts` to enumerate open opportunities for the public
 * sitemap (PERF-W-04).
 *
 * Returns a compact `{ id, updated_at }` projection — sitemap
 * entries don't need the full job payload, and trimming the wire
 * format keeps the sitemap render path under the rendering budget
 * for `/sitemap.xml`.
 *
 * Returns an empty list on any error so a transient backend hiccup
 * never strands Google with an empty sitemap.
 */

import { API_BASE_URL } from "@/shared/lib/api-client"

export interface SitemapJob {
  id: string
  updated_at: string
}

interface PublicJobsEnvelope {
  data?: Array<{ id?: string; updated_at?: string }>
  jobs?: Array<{ id?: string; updated_at?: string }>
}

export async function fetchSitemapJobs(): Promise<SitemapJob[]> {
  try {
    const res = await fetch(
      `${API_BASE_URL || "http://localhost:8080"}/api/v1/jobs?per_page=200&status=open`,
      { next: { revalidate: 300 } },
    )
    if (!res.ok) return []
    const json = (await res.json()) as PublicJobsEnvelope
    const list = json.data ?? json.jobs ?? []
    return list
      .filter((j): j is { id: string; updated_at?: string } => Boolean(j.id))
      .map((j) => ({
        id: j.id,
        updated_at: j.updated_at ?? "",
      }))
  } catch {
    return []
  }
}
