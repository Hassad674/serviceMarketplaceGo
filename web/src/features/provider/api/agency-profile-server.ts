/**
 * agency-profile-server.ts is the server-only fetcher used by
 * `app/[locale]/(public)/agencies/[id]/page.tsx` for SEO metadata
 * (PERF-W-06) and JSON-LD `Organization` schema (PERF-W-02).
 *
 * Mirrors the pattern set by `freelance-profile-server.ts`: hits the
 * public `/api/v1/profiles/:id` aggregate with no cookies, returns
 * null on any error so a transient backend hiccup never tanks the
 * crawl budget. ISR (`revalidate: 120`) keeps the fetch cheap on
 * re-renders.
 */

import { API_BASE_URL } from "@/shared/lib/api-client"
import type { Profile } from "./profile-api"

export async function fetchAgencyProfileForMetadata(
  orgId: string,
): Promise<Profile | null> {
  try {
    const res = await fetch(
      `${API_BASE_URL || "http://localhost:8080"}/api/v1/profiles/${orgId}`,
      { next: { revalidate: 120 } },
    )
    if (!res.ok) return null
    return (await res.json()) as Profile
  } catch {
    return null
  }
}
