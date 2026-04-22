// Server-only fetcher for Next.js `generateMetadata`. Runs in the
// Node.js runtime with no cookies — the public /clients endpoint
// does not require authentication. Kept in its own file so the
// bundler tree-shakes it out of the client bundle.

import { API_BASE_URL } from "@/shared/lib/api-client"
import type { PublicClientProfile } from "./client-profile-api"

// Fallback host must mirror the dev-default used by api-client.ts
// when NEXT_PUBLIC_API_URL is unset. Hard-coding the fallback here
// is cheaper than importing the (client-side) api-client helper.
const SERVER_FALLBACK_HOST = "http://localhost:8080"

export async function fetchPublicClientProfileForMetadata(
  orgId: string,
): Promise<PublicClientProfile | null> {
  try {
    const base = API_BASE_URL || SERVER_FALLBACK_HOST
    const res = await fetch(`${base}/api/v1/clients/${orgId}`, {
      // ISR window: 2 minutes matches TanStack staleTime so the
      // metadata fetch and the client refetch stay in lockstep.
      next: { revalidate: 120 },
    })
    if (!res.ok) return null
    // Backend writes PublicClientProfile directly to the body (no
    // `{ data: ... }` envelope), mirroring the rest of the profile
    // endpoints.
    return (await res.json()) as PublicClientProfile
  } catch {
    return null
  }
}
