"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  getMySocialLinks,
  getPublicSocialLinks,
  upsertSocialLink,
  deleteSocialLink,
} from "../api/social-links-api"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

function mySocialLinksKey(uid: string | undefined) {
  return ["user", uid, "my-social-links"] as const
}

export function useMySocialLinks() {
  const uid = useCurrentUserId()

  return useQuery({
    queryKey: mySocialLinksKey(uid),
    queryFn: getMySocialLinks,
    staleTime: 5 * 60 * 1000,
  })
}

export function usePublicSocialLinks(userId: string) {
  return useQuery({
    queryKey: ["public-social-links", userId],
    queryFn: () => getPublicSocialLinks(userId),
    staleTime: 5 * 60 * 1000,
    enabled: !!userId,
  })
}

export function useUpsertSocialLink() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: ({ platform, url }: { platform: string; url: string }) =>
      upsertSocialLink(platform, url),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: mySocialLinksKey(uid) }),
  })
}

export function useDeleteSocialLink() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()

  return useMutation({
    mutationFn: (platform: string) => deleteSocialLink(platform),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: mySocialLinksKey(uid) }),
  })
}
