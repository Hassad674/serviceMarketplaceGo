"use client"

import { useOrganization } from "@/shared/hooks/use-user"
import { AgencyProfilePage } from "./agency-profile-page"
import { FreelanceOwnProfilePage } from "./freelance-own-profile-page"

// Dispatcher for the /profile route. Agency orgs still run on the
// legacy profile backend (single aggregate, shared columns on
// profiles), while provider_personal orgs have been migrated to the
// split freelance/referrer aggregates. Picking the right subtree on
// the client keeps each path focused and lets the agency UI evolve
// separately in a follow-up refactor.
export default function ProfilePage() {
  const { data: org } = useOrganization()

  if (!org) return <Skeleton />

  if (org.type === "agency") return <AgencyProfilePage />
  return <FreelanceOwnProfilePage />
}

function Skeleton() {
  return (
    <div className="space-y-6" role="status" aria-live="polite">
      <div className="h-32 rounded-xl border border-border bg-muted/40 animate-shimmer" />
      <div className="h-40 rounded-xl border border-border bg-muted/40 animate-shimmer" />
      <div className="h-64 rounded-xl border border-border bg-muted/40 animate-shimmer" />
    </div>
  )
}
