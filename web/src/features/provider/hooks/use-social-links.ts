"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  getMySocialLinks,
  getPublicSocialLinks,
  upsertSocialLink,
  deleteSocialLink,
} from "../api/social-links-api"

const MY_SOCIAL_LINKS_KEY = ["my-social-links"]

export function useMySocialLinks() {
  return useQuery({
    queryKey: MY_SOCIAL_LINKS_KEY,
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

  return useMutation({
    mutationFn: ({ platform, url }: { platform: string; url: string }) =>
      upsertSocialLink(platform, url),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: MY_SOCIAL_LINKS_KEY }),
  })
}

export function useDeleteSocialLink() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (platform: string) => deleteSocialLink(platform),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: MY_SOCIAL_LINKS_KEY }),
  })
}
