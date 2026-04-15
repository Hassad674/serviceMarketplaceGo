"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  deleteFreelanceSocialLink,
  getMyFreelanceSocialLinks,
  getPublicFreelanceSocialLinks,
  upsertFreelanceSocialLink,
} from "../api/freelance-social-links-api"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"

function myFreelanceLinksKey(uid: string | undefined) {
  return ["user", uid, "freelance-social-links"] as const
}

export function useMyFreelanceSocialLinks() {
  const uid = useCurrentUserId()
  return useQuery({
    queryKey: myFreelanceLinksKey(uid),
    queryFn: getMyFreelanceSocialLinks,
    staleTime: 5 * 60 * 1000,
  })
}

export function usePublicFreelanceSocialLinks(orgId: string) {
  return useQuery({
    queryKey: ["public-freelance-social-links", orgId],
    queryFn: () => getPublicFreelanceSocialLinks(orgId),
    staleTime: 5 * 60 * 1000,
    enabled: Boolean(orgId),
  })
}

export function useUpsertFreelanceSocialLink() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  return useMutation({
    mutationFn: ({ platform, url }: { platform: string; url: string }) =>
      upsertFreelanceSocialLink(platform, url),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: myFreelanceLinksKey(uid) }),
  })
}

export function useDeleteFreelanceSocialLink() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  return useMutation({
    mutationFn: (platform: string) => deleteFreelanceSocialLink(platform),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: myFreelanceLinksKey(uid) }),
  })
}
