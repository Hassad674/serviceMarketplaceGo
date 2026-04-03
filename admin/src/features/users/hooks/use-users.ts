import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  listUsers,
  getUser,
  suspendUser,
  unsuspendUser,
  banUser,
  unbanUser,
} from "../api/users-api"
import type { SuspendUserPayload, BanUserPayload } from "../api/users-api"
import type { UserFilters } from "../types"

export function usersQueryKey(filters: UserFilters) {
  return ["admin", "users", filters] as const
}

export function useUsers(filters: UserFilters) {
  return useQuery({
    queryKey: usersQueryKey(filters),
    queryFn: () => listUsers(filters),
    staleTime: 30 * 1000,
  })
}

export function userQueryKey(id: string) {
  return ["admin", "users", id] as const
}

export function useUser(id: string) {
  return useQuery({
    queryKey: userQueryKey(id),
    queryFn: () => getUser(id),
    enabled: !!id,
  })
}

function useInvalidateUser(id: string) {
  const queryClient = useQueryClient()
  return () => {
    queryClient.invalidateQueries({ queryKey: ["admin", "users", id] })
    queryClient.invalidateQueries({ queryKey: ["admin", "users"] })
  }
}

export function useSuspendUser(id: string) {
  const invalidate = useInvalidateUser(id)
  return useMutation({
    mutationFn: (payload: SuspendUserPayload) => suspendUser(id, payload),
    onSuccess: invalidate,
  })
}

export function useUnsuspendUser(id: string) {
  const invalidate = useInvalidateUser(id)
  return useMutation({
    mutationFn: () => unsuspendUser(id),
    onSuccess: invalidate,
  })
}

export function useBanUser(id: string) {
  const invalidate = useInvalidateUser(id)
  return useMutation({
    mutationFn: (payload: BanUserPayload) => banUser(id, payload),
    onSuccess: invalidate,
  })
}

export function useUnbanUser(id: string) {
  const invalidate = useInvalidateUser(id)
  return useMutation({
    mutationFn: () => unbanUser(id),
    onSuccess: invalidate,
  })
}
