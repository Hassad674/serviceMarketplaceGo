// Server-only fetcher used by Next.js `generateMetadata`. Same
// rationale as the freelance-profile-server counterpart.

import { API_BASE_URL } from "@/shared/lib/api-client"
import type { ReferrerProfile } from "./referrer-profile-api"

export async function fetchReferrerProfileForMetadata(
  orgId: string,
): Promise<ReferrerProfile | null> {
  try {
    const res = await fetch(
      `${API_BASE_URL || "http://localhost:8080"}/api/v1/referrer-profiles/${orgId}`,
      { next: { revalidate: 120 } },
    )
    if (!res.ok) return null
    return (await res.json()) as ReferrerProfile
  } catch {
    return null
  }
}
