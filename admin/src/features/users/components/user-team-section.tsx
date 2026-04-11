import { useState } from "react"
import { Users2, AlertTriangle, Building2 } from "lucide-react"
import { Card, CardContent } from "@/shared/components/ui/card"
import { Button } from "@/shared/components/ui/button"
import { Skeleton } from "@/shared/components/ui/skeleton"
import { Badge } from "@/shared/components/ui/badge"
import { formatDate } from "@/shared/lib/utils"
import { useUserOrganization } from "../hooks/use-users"
import { UserTeamMembersList } from "./user-team-members-list"
import { UserTeamInvitationsList } from "./user-team-invitations-list"
import { UserTeamForceTransferDialog } from "./user-team-force-transfer-dialog"
import type { AdminUser } from "../types"

// Rendered in the user detail page when the selected user has an
// organization (either as Owner or Operator). Fetches the team
// aggregate lazily and composes the three sub-lists plus a danger
// zone with the force-transfer override. Solo providers and users
// without an org never render this — the parent gates on
// user.organization_id.

type UserTeamSectionProps = {
  user: AdminUser
}

export function UserTeamSection({ user }: UserTeamSectionProps) {
  const [showForceTransferDialog, setShowForceTransferDialog] = useState(false)

  // Only fetch when we actually have an org to look up. The hook
  // still runs with enabled=false on providers so the query key is
  // registered but no request is fired.
  const hasOrg = !!user.organization_id
  const { data, isLoading, error } = useUserOrganization(user.id, hasOrg)

  if (!hasOrg) return null

  if (isLoading) {
    return (
      <Card>
        <CardContent className="space-y-4 pt-6">
          <SectionHeading />
          <Skeleton className="h-32" />
        </CardContent>
      </Card>
    )
  }

  if (error || !data) {
    return (
      <Card>
        <CardContent className="space-y-4 pt-6">
          <SectionHeading />
          <div className="rounded-lg border border-destructive/20 bg-destructive/5 p-4 text-sm text-destructive">
            Impossible de charger l&apos;organisation de cet utilisateur.
          </div>
        </CardContent>
      </Card>
    )
  }

  const { organization, members, pending_invitations, viewing_role } = data
  const isOwnerView = viewing_role === "owner"

  return (
    <>
      <Card>
        <CardContent className="space-y-6 pt-6">
          <SectionHeading />

          <OrganizationSummary
            organization={organization}
            memberCount={members.length}
            pendingCount={pending_invitations.length}
            isOwnerView={isOwnerView}
          />

          <div>
            <h4 className="mb-3 text-sm font-medium text-foreground">
              Membres ({members.length})
            </h4>
            <UserTeamMembersList
              userID={user.id}
              orgID={organization.id}
              members={members}
              currentOwnerID={organization.owner_user_id}
            />
          </div>

          <div>
            <h4 className="mb-3 text-sm font-medium text-foreground">
              Invitations en attente ({pending_invitations.length})
            </h4>
            <UserTeamInvitationsList
              userID={user.id}
              orgID={organization.id}
              invitations={pending_invitations}
            />
          </div>
        </CardContent>
      </Card>

      {/* Danger zone — only shown on the Owner view. Operators have no
          ownership-related actions attached to their detail page. */}
      {isOwnerView && (
        <Card className="border-destructive/30">
          <CardContent className="space-y-4 pt-6">
            <div className="flex items-center gap-2">
              <AlertTriangle className="h-4 w-4 text-destructive" />
              <h3 className="text-sm font-semibold uppercase tracking-wider text-destructive">
                Zone dangereuse
              </h3>
            </div>
            <p className="text-sm text-muted-foreground">
              Forcer le transfert d&apos;ownership est une action r&eacute;serv&eacute;e aux cas de blocage
              (Owner inaccessible, compte compromis). Elle d&eacute;clenche la m&ecirc;me r&eacute;vocation
              de session qu&apos;un transfert normal.
            </p>
            <Button variant="destructive" onClick={() => setShowForceTransferDialog(true)}>
              Forcer le transfert d&apos;ownership
            </Button>
          </CardContent>
        </Card>
      )}

      {showForceTransferDialog && (
        <UserTeamForceTransferDialog
          open={showForceTransferDialog}
          onClose={() => setShowForceTransferDialog(false)}
          userID={user.id}
          orgID={organization.id}
          currentOwnerID={organization.owner_user_id}
          members={members}
        />
      )}
    </>
  )
}

function SectionHeading() {
  return (
    <div className="flex items-center gap-2">
      <Users2 className="h-4 w-4 text-muted-foreground" />
      <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
        &Eacute;quipe
      </h3>
    </div>
  )
}

type OrganizationSummaryProps = {
  organization: {
    id: string
    type: string
    created_at: string
    pending_transfer_to_user_id?: string
    pending_transfer_expires_at?: string
  }
  memberCount: number
  pendingCount: number
  isOwnerView: boolean
}

function OrganizationSummary({
  organization,
  memberCount,
  pendingCount,
  isOwnerView,
}: OrganizationSummaryProps) {
  const typeLabel = organization.type === "agency" ? "Agence" : "Entreprise"
  return (
    <div className="rounded-lg border border-border bg-muted/30 p-4">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex items-start gap-3">
          <div className="rounded-lg bg-primary/10 p-2">
            <Building2 className="h-5 w-5 text-primary" />
          </div>
          <div>
            <p className="text-sm font-semibold text-foreground">{typeLabel}</p>
            <p className="text-xs text-muted-foreground">
              Cr&eacute;&eacute;e le {formatDate(organization.created_at)}
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              {organization.id.slice(0, 8)}&hellip;
            </p>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          <Badge variant="default">{memberCount} membres</Badge>
          {pendingCount > 0 && <Badge variant="warning">{pendingCount} invitations</Badge>}
          {!isOwnerView && <Badge variant="outline">Vue op&eacute;rateur</Badge>}
          {organization.pending_transfer_to_user_id && (
            <Badge variant="warning">Transfert en cours</Badge>
          )}
        </div>
      </div>
    </div>
  )
}
