import { ClipboardList } from "lucide-react"

export default function MissionsPage() {
  return (
    <div>
      <div className="mb-6 flex items-center gap-3">
        <ClipboardList className="h-6 w-6 text-primary" />
        <h1 className="text-2xl font-bold text-foreground">Suivi des missions</h1>
      </div>
      <div className="rounded-xl border border-border bg-card p-8 text-center text-muted-foreground shadow-sm">
        <p>Vue d'ensemble des missions, statuts, affectations, litiges.</p>
        <p className="mt-2 text-sm">Connecter aux endpoints API pour afficher les données.</p>
      </div>
    </div>
  )
}
