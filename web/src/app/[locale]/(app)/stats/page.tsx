"use client"

import { useEffect } from "react"
import { useRouter } from "@i18n/navigation"
import { useUser } from "@/shared/hooks/use-user"
import { useWorkspace } from "@/shared/hooks/use-workspace"
import { StatsOverview } from "@/features/stats/components/stats-overview"

// /stats is the deep-dive analytics page for Provider + Agency roles.
// Enterprise and Referrer redirect to /dashboard since the underlying
// /me/stats/visibility + /me/stats/keywords endpoints are scoped to
// the Typesense index records the public listings populate (and an
// Enterprise org has no public listing). The redirect happens at the
// page level — middleware would have to read role context, which we
// avoid to keep middleware fast.

export default function StatsPage() {
  const router = useRouter()
  const { data: user, isLoading } = useUser()
  const { isReferrerMode } = useWorkspace()

  const role = user?.role
  const isProviderOrAgency = role === "agency" || role === "provider"
  const inReferrerMode = role === "provider" && isReferrerMode

  useEffect(() => {
    if (isLoading) return
    if (!user) return // useUser hook will already have redirected on a 401
    if (!isProviderOrAgency || inReferrerMode) {
      router.replace("/dashboard")
    }
  }, [isLoading, user, isProviderOrAgency, inReferrerMode, router])

  if (isLoading || !isProviderOrAgency || inReferrerMode) {
    return (
      <div className="space-y-4" aria-busy="true">
        <div className="h-10 w-48 animate-pulse rounded-full bg-muted" />
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <div className="h-56 animate-pulse rounded-2xl bg-muted/60" />
          <div className="h-56 animate-pulse rounded-2xl bg-muted/60" />
        </div>
      </div>
    )
  }

  return <StatsOverview />
}
