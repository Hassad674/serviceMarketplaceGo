import { useState } from "react"
import { Dialog, DialogTitle, DialogDescription, DialogFooter } from "@/shared/components/ui/dialog"
import { Button } from "@/shared/components/ui/button"
import { Textarea } from "@/shared/components/ui/textarea"
import { Input } from "@/shared/components/ui/input"

// Modal used by the action bar on the user detail page to collect a
// suspension reason and optional expiry date before firing the
// mutation. Kept dumb: the parent owns the isPending + mutate call.

type UserSuspendDialogProps = {
  open: boolean
  onClose: () => void
  onConfirm: (reason: string, expiresAt: string) => void
  isPending: boolean
}

export function UserSuspendDialog({ open, onClose, onConfirm, isPending }: UserSuspendDialogProps) {
  const [reason, setReason] = useState("")
  const [expiresAt, setExpiresAt] = useState("")

  return (
    <Dialog open={open} onClose={onClose}>
      <DialogTitle>Suspendre l&apos;utilisateur</DialogTitle>
      <DialogDescription>
        Indiquez la raison et la dur&eacute;e de la suspension.
      </DialogDescription>
      <div className="mt-4 space-y-4">
        <Textarea
          label="Raison"
          required
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="Raison de la suspension..."
          rows={3}
        />
        <Input
          type="datetime-local"
          label="Date d'expiration (optionnel)"
          value={expiresAt}
          onChange={(e) => setExpiresAt(e.target.value)}
        />
      </div>
      <DialogFooter>
        <Button variant="outline" onClick={onClose}>Annuler</Button>
        <Button
          variant="destructive"
          onClick={() => {
            if (!reason.trim()) return
            const iso = expiresAt ? new Date(expiresAt).toISOString() : ""
            onConfirm(reason.trim(), iso)
          }}
          disabled={!reason.trim() || isPending}
        >
          {isPending ? "En cours..." : "Suspendre"}
        </Button>
      </DialogFooter>
    </Dialog>
  )
}
