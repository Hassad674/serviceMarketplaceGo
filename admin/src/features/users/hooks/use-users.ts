import { useQuery } from "@tanstack/react-query"
import { listUsers, getUser } from "../api/users-api"
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
