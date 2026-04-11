import { Mail, X } from "lucide-react"
import { Button } from "@/shared/components/ui/button"
import { Badge } from "@/shared/components/ui/badge"
import { formatDate } from "@/shared/lib/utils"
import { useForceCancelInvitation } from "../hooks/use-users"
import type { AdminOrganizationInvitation } from "../types"

// Pending invitations section of the team view. Each row exposes a
// single admin override: cancel the invitation. Already-accepted and
// already-cancelled rows never reach this component (the backend
// filters on status=pending) so there is no need for status badges.

type UserTeamInvitationsListProps = {
  userID: string // the user whose detail page hosts this section
  orgID: string
  invitations: AdminOrganizationInvitation[]
}

export function UserTeamInvitationsList({
  userID,
  orgID,
  invitations,
}: UserTeamInvitationsListProps) {
  if (invitations.length === 0) {
    return (
      <div className="rounded-lg border border-border bg-white p-6 text-center text-sm text-muted-foreground">
        Aucune invitation en attente
      </div>
    )
  }

  return (
    <div className="overflow-hidden rounded-lg border border-border bg-white">
      <table className="w-full">
        <thead>
          <tr className="border-b border-border bg-muted/50 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
            <th className="px-4 py-3">Destinataire</th>
            <th className="px-4 py-3">R&ocirc;le propos&eacute;</th>
            <th className="px-4 py-3">Envoy&eacute;e</th>
            <th className="px-4 py-3">Expire</th>
            <th className="px-4 py-3 text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {invitations.map((inv) => (
            <InvitationRow
              key={inv.id}
              invitation={inv}
              userID={userID}
              orgID={orgID}
            />
          ))}
        </tbody>
      </table>
    </div>
  )
}

type InvitationRowProps = {
  invitation: AdminOrganizationInvitation
  userID: string
  orgID: string
}

function InvitationRow({ invitation, userID, orgID }: InvitationRowProps) {
  const mutation = useForceCancelInvitation(userID, orgID, invitation.id)
  const fullName = `${invitation.first_name} ${invitation.last_name}`.trim()

  return (
    <tr className="text-sm">
      <td className="px-4 py-3">
        <div className="flex items-center gap-2">
          <Mail className="h-4 w-4 text-muted-foreground" />
          <div>
            <p className="font-medium text-foreground">{fullName || invitation.email}</p>
            <p className="text-xs text-muted-foreground">{invitation.email}</p>
          </div>
        </div>
      </td>
      <td className="px-4 py-3">
        <Badge variant="outline">{invitation.role}</Badge>
      </td>
      <td className="px-4 py-3 text-muted-foreground">{formatDate(invitation.created_at)}</td>
      <td className="px-4 py-3 text-muted-foreground">{formatDate(invitation.expires_at)}</td>
      <td className="px-4 py-3 text-right">
        <Button
          size="sm"
          variant="destructive"
          onClick={() => mutation.mutate()}
          disabled={mutation.isPending}
        >
          <X className="h-4 w-4" />
          {mutation.isPending ? "Annulation..." : "Annuler"}
        </Button>
      </td>
    </tr>
  )
}
