import { useState } from "react"
import { Dialog, DialogTitle, DialogDescription, DialogFooter } from "@/shared/components/ui/dialog"
import { Button } from "@/shared/components/ui/button"
import { Select } from "@/shared/components/ui/select"
import { useForceTransferOwnership } from "../hooks/use-users"
import type { AdminOrganizationMember } from "../types"

// Danger-zone dialog. The admin picks an existing member (any role)
// as the new owner and confirms. The backend bypasses the regular
// "target must be Admin" guardrail — this is the emergency escape
// hatch for locked organizations.

type UserTeamForceTransferDialogProps = {
  open: boolean
  onClose: () => void
  userID: string // the user whose detail page triggered this
  orgID: string
  currentOwnerID: string
  members: AdminOrganizationMember[]
}

export function UserTeamForceTransferDialog({
  open,
  onClose,
  userID,
  orgID,
  currentOwnerID,
  members,
}: UserTeamForceTransferDialogProps) {
  const [targetUserID, setTargetUserID] = useState("")
  const mutation = useForceTransferOwnership(userID, orgID)

  // Only members who aren't already the Owner are eligible — the
  // backend also rejects same-owner transfers but pruning here keeps
  // the dropdown honest.
  const eligibleMembers = members.filter((m) => m.user_id !== currentOwnerID)

  const options = eligibleMembers.map((m) => ({
    value: m.user_id,
    label: `${m.role} · ${m.title || "(sans titre)"} · ${m.user_id.slice(0, 8)}…`,
  }))

  const handleConfirm = () => {
    if (!targetUserID) return
    mutation.mutate(
      { target_user_id: targetUserID },
      {
        onSuccess: () => {
          setTargetUserID("")
          onClose()
        },
      },
    )
  }

  return (
    <Dialog open={open} onClose={onClose}>
      <DialogTitle>Forcer le transfert d&apos;ownership</DialogTitle>
      <DialogDescription>
        Action r&eacute;serv&eacute;e aux cas de blocage (Owner inaccessible). Le nouveau propri&eacute;taire doit d&eacute;j&agrave; &ecirc;tre membre de l&apos;organisation.
        L&apos;ancien Owner sera r&eacute;trograd&eacute; au rang d&apos;Admin.
      </DialogDescription>
      <div className="mt-4 space-y-4">
        {options.length === 0 ? (
          <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800">
            Aucun membre &eacute;ligible. Invitez d&apos;abord un nouvel op&eacute;rateur dans l&apos;organisation.
          </div>
        ) : (
          <Select
            label="Nouvel Owner"
            options={options}
            placeholder="S&eacute;lectionnez un membre"
            value={targetUserID}
            onChange={(e) => setTargetUserID(e.target.value)}
          />
        )}
        {mutation.isError && (
          <p className="text-sm text-destructive">
            Erreur : {mutation.error instanceof Error ? mutation.error.message : "inconnue"}
          </p>
        )}
      </div>
      <DialogFooter>
        <Button variant="outline" onClick={onClose}>Annuler</Button>
        <Button
          variant="destructive"
          onClick={handleConfirm}
          disabled={!targetUserID || mutation.isPending}
        >
          {mutation.isPending ? "Transfert..." : "Forcer le transfert"}
        </Button>
      </DialogFooter>
    </Dialog>
  )
}
