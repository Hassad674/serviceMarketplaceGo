import { useState } from "react"
import { Dialog, DialogTitle, DialogDescription, DialogFooter } from "@/shared/components/ui/dialog"
import { Button } from "@/shared/components/ui/button"
import { Textarea } from "@/shared/components/ui/textarea"

// Hard-ban confirmation modal. Severe action — the copy warns that the
// user loses all access. Kept stateless: the parent owns isPending +
// mutate.

type UserBanDialogProps = {
  open: boolean
  onClose: () => void
  onConfirm: (reason: string) => void
  isPending: boolean
}

export function UserBanDialog({ open, onClose, onConfirm, isPending }: UserBanDialogProps) {
  const [reason, setReason] = useState("")

  return (
    <Dialog open={open} onClose={onClose}>
      <DialogTitle>Bannir l&apos;utilisateur</DialogTitle>
      <DialogDescription>
        Cette action est s&eacute;v&egrave;re. L&apos;utilisateur ne pourra plus acc&eacute;der &agrave; la plateforme.
      </DialogDescription>
      <div className="mt-4">
        <Textarea
          label="Raison"
          required
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="Raison du bannissement..."
          rows={3}
        />
      </div>
      <DialogFooter>
        <Button variant="outline" onClick={onClose}>Annuler</Button>
        <Button
          variant="destructive"
          onClick={() => {
            if (!reason.trim()) return
            onConfirm(reason.trim())
          }}
          disabled={!reason.trim() || isPending}
        >
          {isPending ? "En cours..." : "Bannir"}
        </Button>
      </DialogFooter>
    </Dialog>
  )
}
