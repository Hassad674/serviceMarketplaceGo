import { useQuery } from "@tanstack/react-query"
import { getCyclePreview } from "../api/subscription-api"
import { subscriptionQueryKey } from "./keys"
import type { BillingCycle, CyclePreview } from "../types"

/**
 * Preview hook fired by the manage modal when the user clicks
 * "Passer à l'annuel" / "Repasser en mensuel". Disabled until an
 * explicit target cycle is set so we never pre-fetch unused data.
 *
 * The response is cached for the length of the modal interaction —
 * a fresh click to switch direction busts the cache key and the
 * user always sees a number that matches the exact cycle they
 * picked.
 */
export function useCyclePreview(target: BillingCycle | null) {
	return useQuery<CyclePreview>({
		queryKey: subscriptionQueryKey.cyclePreview(target),
		queryFn: () => {
			if (!target) {
				throw new Error("useCyclePreview: target cycle is required")
			}
			return getCyclePreview(target)
		},
		enabled: target !== null,
		staleTime: 30 * 1000,
		retry: 1,
	})
}
