"use client"

import { useHasPermission } from "@/shared/hooks/use-permissions"
import { SocialLinksCard } from "@/shared/components/profile/social-links-card"
import {
  useDeleteReferrerSocialLink,
  useMyReferrerSocialLinks,
  useUpsertReferrerSocialLink,
  usePublicReferrerSocialLinks,
} from "../hooks/use-referrer-social-links"

// ReferrerSocialLinksSection mounts the shared social-links card on
// the authenticated /referral page. Every provider_personal user
// gets an independent set here from their freelance persona.
export function ReferrerSocialLinksSection() {
  const canEdit = useHasPermission("org_profile.edit")
  const { data: links = [], isLoading } = useMyReferrerSocialLinks()
  const upsertMutation = useUpsertReferrerSocialLink()
  const deleteMutation = useDeleteReferrerSocialLink()

  return (
    <SocialLinksCard
      links={links}
      isLoading={isLoading}
      editor={{
        canEdit,
        onUpsert: async (platform, url) => {
          await upsertMutation.mutateAsync({ platform, url })
        },
        onDelete: async (platform) => {
          await deleteMutation.mutateAsync(platform)
        },
      }}
    />
  )
}

interface PublicReferrerSocialLinksProps {
  orgId: string
}

// PublicReferrerSocialLinks is the read-only variant used on the
// /referrers/[id] public page.
export function PublicReferrerSocialLinks({
  orgId,
}: PublicReferrerSocialLinksProps) {
  const { data: links = [], isLoading } = usePublicReferrerSocialLinks(orgId)

  return <SocialLinksCard links={links} isLoading={isLoading} />
}
