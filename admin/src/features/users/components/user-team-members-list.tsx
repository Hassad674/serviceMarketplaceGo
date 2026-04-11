import { useState } from "react"
import { Crown, Shield, UserMinus, UserCog } from "lucide-react"
import { Badge } from "@/shared/components/ui/badge"
import { Button } from "@/shared/components/ui/button"
import { Dialog, DialogTitle, DialogDescription, DialogFooter } from "@/shared/components/ui/dialog"
import { Select } from "@/shared/components/ui/select"
import { formatDate } from "@/shared/lib/utils"
import { useForceUpdateMemberRole, useForceRemoveMember } from "../hooks/use-users"
import type { AdminOrganizationMember } from "../types"

// Stacked card rendering the full member list for an org. Each row
// exposes two admin overrides via small inline buttons: change role
// (opens a dialog) and remove (opens a confirmation dialog). The
// current Owner row hides both actions — the only way to touch the
// Owner is via the force-transfer flow.

type UserTeamMembersListProps = {
  userID: string // the user whose detail page hosts this section
  orgID: string
  members: AdminOrganizationMember[]
  currentOwnerID: string
}

export function UserTeamMembersList({
  userID,
  orgID,
  members,
  currentOwnerID,
}: UserTeamMembersListProps) {
  const [editTarget, setEditTarget] = useState<AdminOrganizationMember | null>(null)
  const [removeTarget, setRemoveTarget] = useState<AdminOrganizationMember | null>(null)

  if (members.length === 0) {
    return (
      <div className="rounded-lg border border-border bg-white p-6 text-center text-sm text-muted-foreground">
        Aucun membre
      </div>
    )
  }

  return (
    <div className="overflow-hidden rounded-lg border border-border bg-white">
      <table className="w-full">
        <thead>
          <tr className="border-b border-border bg-muted/50 text-left text-xs font-medium uppercase tracking-wider text-muted-foreground">
            <th className="px-4 py-3">Membre</th>
            <th className="px-4 py-3">Titre</th>
            <th className="px-4 py-3">Arriv&eacute;</th>
            <th className="px-4 py-3 text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {members.map((m) => {
            const isOwner = m.user_id === currentOwnerID
            return (
              <tr key={m.id} className="text-sm">
                <td className="px-4 py-3">
                  <div className="flex items-center gap-2">
                    {isOwner ? (
                      <Crown className="h-4 w-4 text-amber-500" />
                    ) : (
                      <Shield className="h-4 w-4 text-muted-foreground" />
                    )}
                    <MemberRoleBadge role={m.role} />
                  </div>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {m.user_id.slice(0, 8)}&hellip;
                  </p>
                </td>
                <td className="px-4 py-3 text-foreground">
                  {m.title || <span className="text-muted-foreground">&mdash;</span>}
                </td>
                <td className="px-4 py-3 text-muted-foreground">
                  {formatDate(m.joined_at)}
                </td>
                <td className="px-4 py-3">
                  <div className="flex justify-end gap-2">
                    {!isOwner && (
                      <>
                        <Button size="sm" variant="outline" onClick={() => setEditTarget(m)}>
                          <UserCog className="h-4 w-4" /> R&ocirc;le
                        </Button>
                        <Button
                          size="sm"
                          variant="destructive"
                          onClick={() => setRemoveTarget(m)}
                        >
                          <UserMinus className="h-4 w-4" /> Retirer
                        </Button>
                      </>
                    )}
                  </div>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>

      {editTarget && (
        <EditRoleDialog
          userID={userID}
          orgID={orgID}
          member={editTarget}
          onClose={() => setEditTarget(null)}
        />
      )}
      {removeTarget && (
        <RemoveMemberDialog
          userID={userID}
          orgID={orgID}
          member={removeTarget}
          onClose={() => setRemoveTarget(null)}
        />
      )}
    </div>
  )
}

function MemberRoleBadge({ role }: { role: string }) {
  const variantMap: Record<string, "default" | "success" | "warning" | "outline"> = {
    owner: "warning",
    admin: "default",
    member: "success",
    viewer: "outline",
  }
  const labelMap: Record<string, string> = {
    owner: "Owner",
    admin: "Admin",
    member: "Member",
    viewer: "Viewer",
  }
  return <Badge variant={variantMap[role] || "outline"}>{labelMap[role] || role}</Badge>
}

type EditRoleDialogProps = {
  userID: string
  orgID: string
  member: AdminOrganizationMember
  onClose: () => void
}

function EditRoleDialog({ userID, orgID, member, onClose }: EditRoleDialogProps) {
  const [role, setRole] = useState<"admin" | "member" | "viewer">(
    member.role === "owner" ? "admin" : (member.role as "admin" | "member" | "viewer"),
  )
  const mutation = useForceUpdateMemberRole(userID, orgID, member.user_id)

  return (
    <Dialog open onClose={onClose}>
      <DialogTitle>Changer le r&ocirc;le</DialogTitle>
      <DialogDescription>
        Op&eacute;rateur : {member.user_id.slice(0, 8)}&hellip; · r&ocirc;le actuel : <strong>{member.role}</strong>
      </DialogDescription>
      <div className="mt-4">
        <Select
          label="Nouveau r&ocirc;le"
          options={[
            { value: "admin", label: "Admin" },
            { value: "member", label: "Member" },
            { value: "viewer", label: "Viewer" },
          ]}
          value={role}
          onChange={(e) => setRole(e.target.value as "admin" | "member" | "viewer")}
        />
        {mutation.isError && (
          <p className="mt-2 text-sm text-destructive">
            Erreur lors du changement de r&ocirc;le
          </p>
        )}
      </div>
      <DialogFooter>
        <Button variant="outline" onClick={onClose}>Annuler</Button>
        <Button
          variant="primary"
          onClick={() => {
            mutation.mutate({ role }, { onSuccess: onClose })
          }}
          disabled={mutation.isPending || role === member.role}
        >
          {mutation.isPending ? "Enregistrement..." : "Confirmer"}
        </Button>
      </DialogFooter>
    </Dialog>
  )
}

type RemoveMemberDialogProps = {
  userID: string
  orgID: string
  member: AdminOrganizationMember
  onClose: () => void
}

function RemoveMemberDialog({ userID, orgID, member, onClose }: RemoveMemberDialogProps) {
  const mutation = useForceRemoveMember(userID, orgID, member.user_id)

  return (
    <Dialog open onClose={onClose}>
      <DialogTitle>Retirer le membre</DialogTitle>
      <DialogDescription>
        Cette action retire le membre de l&apos;organisation. Si c&apos;est un op&eacute;rateur, son compte sera supprim&eacute; enti&egrave;rement.
      </DialogDescription>
      {mutation.isError && (
        <p className="mt-2 text-sm text-destructive">Erreur lors du retrait</p>
      )}
      <DialogFooter>
        <Button variant="outline" onClick={onClose}>Annuler</Button>
        <Button
          variant="destructive"
          onClick={() => mutation.mutate(undefined, { onSuccess: onClose })}
          disabled={mutation.isPending}
        >
          {mutation.isPending ? "Retrait..." : "Retirer"}
        </Button>
      </DialogFooter>
    </Dialog>
  )
}
