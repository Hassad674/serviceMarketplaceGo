"use client"

import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  updateClientProfile,
  type UpdateClientProfileInput,
} from "../api/client-profile-api"

// useUpdateClientProfile wraps PUT /api/v1/profile/client. On success
// we invalidate the session + private profile caches so the edited
// `client_description` / `company_name` propagates everywhere — the
// private client-profile page reads from `useProfile()` (provider
// feature) and any sidebar/avatar surface reads from `useSession()`.
export function useUpdateClientProfile() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: UpdateClientProfileInput) => updateClientProfile(input),
    onSuccess: () => {
      // Profile shared across features. Invalidate generously — these
      // caches are cheap to refetch and mis-stale data confuses users.
      queryClient.invalidateQueries({ queryKey: ["client-profile"] })
      // Cross-feature caches that mirror /api/v1/profile data. Using
      // `predicate` keeps this invalidation resilient to the provider
      // feature's internal queryKey shape (`["user", uid, "profile"]`)
      // without creating an import dependency on that feature.
      queryClient.invalidateQueries({
        predicate: (query) =>
          Array.isArray(query.queryKey) && query.queryKey.includes("profile"),
      })
      queryClient.invalidateQueries({ queryKey: ["session"] })
      queryClient.invalidateQueries({ queryKey: ["public-client-profile"] })
    },
  })
}
