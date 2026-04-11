import { useParams, useNavigate } from "react-router-dom"
import { ArrowLeft } from "lucide-react"
import { PageHeader } from "@/shared/components/layouts/page-header"
import { Button } from "@/shared/components/ui/button"
import { useUser } from "../hooks/use-users"
import { UserProfileCard, UserDetailsCard } from "./user-info-cards"
import { SuspensionInfoCard, BanInfoCard } from "./user-suspension-ban-info"
import { UserModerationActions } from "./user-moderation-actions"
import { UserReportsSection } from "./user-reports-section"
import { UserDetailSkeleton } from "./user-detail-skeleton"
import { UserTeamSection } from "./user-team-section"

// Thin composition shell for /users/:id. Data fetching happens once
// via useUser, then each section owns its own mutations and local
// state. Keeping this file small makes the page easy to rearrange
// (e.g. adding the team section in Phase 6d) without touching any
// of the sub-components.

export function UserDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data, isLoading, error } = useUser(id!)

  if (isLoading) return <UserDetailSkeleton />

  if (error || !data) {
    return (
      <div className="space-y-6">
        <Button variant="ghost" size="sm" onClick={() => navigate(-1)}>
          <ArrowLeft className="h-4 w-4" /> Retour
        </Button>
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center text-sm text-destructive">
          Utilisateur introuvable
        </div>
      </div>
    )
  }

  const user = data.data
  const name = user.display_name || `${user.first_name} ${user.last_name}`

  return (
    <div className="space-y-6">
      <Button variant="ghost" size="sm" onClick={() => navigate(-1)}>
        <ArrowLeft className="h-4 w-4" /> Retour aux utilisateurs
      </Button>

      <PageHeader title={name} />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <UserProfileCard user={user} name={name} />
        <UserDetailsCard user={user} />
      </div>

      {user.status === "suspended" && user.suspension_reason && (
        <SuspensionInfoCard
          reason={user.suspension_reason}
          suspendedAt={user.suspended_at}
          expiresAt={user.suspension_expires_at}
        />
      )}

      {user.status === "banned" && user.ban_reason && (
        <BanInfoCard reason={user.ban_reason} bannedAt={user.banned_at} />
      )}

      <UserModerationActions user={user} />

      {/* Team section: appears whenever the user belongs to an org,
          either as Owner (agency/enterprise marketplace owner) or as
          Operator (invited into another org). Solo providers and
          unprovisioned users render nothing here. */}
      <UserTeamSection user={user} />

      <UserReportsSection userId={user.id} />
    </div>
  )
}
