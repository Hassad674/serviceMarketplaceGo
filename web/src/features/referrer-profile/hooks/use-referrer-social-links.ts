"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  deleteReferrerSocialLink,
  getMyReferrerSocialLinks,
  getPublicReferrerSocialLinks,
  upsertReferrerSocialLink,
} from "../api/referrer-social-links-api"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

function myReferrerLinksKey(uid: string | undefined) {
  return ["user", uid, "referrer-social-links"] as const
}

export function useMyReferrerSocialLinks() {
  const uid = useCurrentUserId()
  return useQuery({
    queryKey: myReferrerLinksKey(uid),
    queryFn: getMyReferrerSocialLinks,
    staleTime: 5 * 60 * 1000,
  })
}

export function usePublicReferrerSocialLinks(orgId: string) {
  return useQuery({
    queryKey: ["public-referrer-social-links", orgId],
    queryFn: () => getPublicReferrerSocialLinks(orgId),
    staleTime: 5 * 60 * 1000,
    enabled: Boolean(orgId),
  })
}

export function useUpsertReferrerSocialLink() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  return useMutation({
    mutationFn: ({ platform, url }: { platform: string; url: string }) =>
      upsertReferrerSocialLink(platform, url),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: myReferrerLinksKey(uid) }),
  })
}

export function useDeleteReferrerSocialLink() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  return useMutation({
    mutationFn: (platform: string) => deleteReferrerSocialLink(platform),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: myReferrerLinksKey(uid) }),
  })
}
