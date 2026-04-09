import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { listDisputes, getDispute, resolveDispute, countDisputes } from "../api/disputes-api"
import type { DisputeFilters } from "../types"

export function disputesQueryKey(filters: DisputeFilters) {
  return ["admin", "disputes", filters] as const
}

export function disputeQueryKey(id: string) {
  return ["admin", "disputes", id] as const
}

export function useDisputes(filters: DisputeFilters) {
  return useQuery({
    queryKey: disputesQueryKey(filters),
    queryFn: () => listDisputes(filters),
    staleTime: 30_000,
  })
}

export function useDispute(id: string) {
  return useQuery({
    queryKey: disputeQueryKey(id),
    queryFn: () => getDispute(id),
    staleTime: 30_000,
  })
}

export function useResolveDispute(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: { amount_client: number; amount_provider: number; note: string }) =>
      resolveDispute(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["admin", "disputes"] })
    },
  })
}

export function useDisputeCount() {
  return useQuery({
    queryKey: ["admin", "disputes", "count"],
    queryFn: countDisputes,
    staleTime: 60_000,
  })
}
