"use client"

import { useUser } from "./use-user"

export function useCurrentUserId(): string | undefined {
  const { data } = useUser()
  return data?.id
}
