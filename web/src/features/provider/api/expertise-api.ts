import { apiClient } from "@/shared/lib/api-client"

// Backend wraps the response body in a `data` envelope specifically for
// this endpoint — existing profile endpoints return flat objects, so we
// normalize here and expose only the meaningful payload to callers.
type UpdateExpertiseResponse = {
  data: {
    expertise_domains: string[]
  }
}

export type UpdateExpertiseResult = {
  expertise_domains: string[]
}

// PUT /api/v1/profile/expertise — replaces the full list. The array
// order sent in the request becomes the canonical display order.
export async function updateExpertiseDomains(
  domains: string[],
): Promise<UpdateExpertiseResult> {
  const response = await apiClient<UpdateExpertiseResponse>(
    "/api/v1/profile/expertise",
    {
      method: "PUT",
      body: { domains },
    },
  )
  return { expertise_domains: response.data.expertise_domains }
}
