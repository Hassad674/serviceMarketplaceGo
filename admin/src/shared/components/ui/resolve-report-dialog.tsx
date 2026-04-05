import { useState } from "react"
import { Dialog, DialogTitle, DialogDescription, DialogFooter } from "./dialog"
import { Textarea } from "./textarea"
import { Button } from "./button"

type ResolveReportDialogProps = {
  open: boolean
  onClose: () => void
  onConfirm: (status: "resolved" | "dismissed", adminNote: string) => void
  isPending: boolean
  defaultStatus?: "resolved" | "dismissed"
}

export function ResolveReportDialog({
  open,
  onClose,
  onConfirm,
  isPending,
  defaultStatus = "resolved",
}: ResolveReportDialogProps) {
  const [status, setStatus] = useState<"resolved" | "dismissed">(defaultStatus)
  const [note, setNote] = useState("")

  function handleConfirm() {
    if (!note.trim()) return
    onConfirm(status, note.trim())
  }

  const title = status === "resolved" ? "Resoudre le signalement" : "Rejeter le signalement"

  return (
    <Dialog open={open} onClose={onClose}>
      <DialogTitle>{title}</DialogTitle>
      <DialogDescription>
        Ajoutez une note expliquant votre decision.
      </DialogDescription>

      <div className="mt-4 space-y-4">
        <div>
          <label className="mb-1 block text-sm font-medium text-foreground">Decision</label>
          <div className="flex gap-3">
            <label className="flex items-center gap-2 text-sm cursor-pointer">
              <input
                type="radio"
                name="status"
                value="resolved"
                checked={status === "resolved"}
                onChange={() => setStatus("resolved")}
                className="accent-primary"
              />
              Resolu
            </label>
            <label className="flex items-center gap-2 text-sm cursor-pointer">
              <input
                type="radio"
                name="status"
                value="dismissed"
                checked={status === "dismissed"}
                onChange={() => setStatus("dismissed")}
                className="accent-primary"
              />
              Rejete
            </label>
          </div>
        </div>

        <Textarea
          label="Note admin"
          required
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder="Expliquez votre decision..."
          rows={3}
        />
      </div>

      <DialogFooter>
        <Button variant="outline" onClick={onClose}>Annuler</Button>
        <Button
          variant={status === "resolved" ? "primary" : "destructive"}
          onClick={handleConfirm}
          disabled={!note.trim() || isPending}
        >
          {isPending ? "En cours..." : "Confirmer"}
        </Button>
      </DialogFooter>
    </Dialog>
  )
}
