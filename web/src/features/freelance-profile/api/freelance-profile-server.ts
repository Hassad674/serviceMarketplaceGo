// Server-only fetcher used by Next.js `generateMetadata`. Runs in
// the Node.js runtime with no cookies — the public endpoint does not
// require authentication so the SEO metadata builder can pull the
// profile directly without forwarding credentials.
//
// Kept in its own file (not co-located with the client API) so the
// bundler tree-shakes it out of the client bundle.

import { API_BASE_URL } from "@/shared/lib/api-client"
import type { FreelanceProfile } from "./freelance-profile-api"

export async function fetchFreelanceProfileForMetadata(
  orgId: string,
): Promise<FreelanceProfile | null> {
  try {
    const res = await fetch(
      `${API_BASE_URL || "http://localhost:8080"}/api/v1/freelance-profiles/${orgId}`,
      { next: { revalidate: 120 } },
    )
    if (!res.ok) return null
    return (await res.json()) as FreelanceProfile
  } catch {
    return null
  }
}
