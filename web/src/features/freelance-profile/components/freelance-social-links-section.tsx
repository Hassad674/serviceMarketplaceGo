"use client"

import { useHasPermission } from "@/shared/hooks/use-permissions"
import { SocialLinksCard } from "@/shared/components/profile/social-links-card"
import {
  useDeleteFreelanceSocialLink,
  useMyFreelanceSocialLinks,
  useUpsertFreelanceSocialLink,
  usePublicFreelanceSocialLinks,
} from "../hooks/use-freelance-social-links"

// FreelanceSocialLinksSection mounts the shared social-links card on
// the authenticated /profile page for provider_personal users. It
// owns the TanStack Query wiring so the shared card stays decoupled
// from the query layer.
export function FreelanceSocialLinksSection() {
  const canEdit = useHasPermission("org_profile.edit")
  const { data: links = [], isLoading } = useMyFreelanceSocialLinks()
  const upsertMutation = useUpsertFreelanceSocialLink()
  const deleteMutation = useDeleteFreelanceSocialLink()

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

interface PublicFreelanceSocialLinksProps {
  orgId: string
}

// PublicFreelanceSocialLinks is the read-only variant used on the
// /freelancers/[id] public page. Collapses to nothing when the set
// is empty — that logic lives inside the shared card.
export function PublicFreelanceSocialLinks({
  orgId,
}: PublicFreelanceSocialLinksProps) {
  const { data: links = [], isLoading } = usePublicFreelanceSocialLinks(orgId)

  return <SocialLinksCard links={links} isLoading={isLoading} />
}
