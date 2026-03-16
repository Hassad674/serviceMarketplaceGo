import { ShieldAlert } from "lucide-react"

export default function ReportsPage() {
  return (
    <div>
      <div className="mb-6 flex items-center gap-3">
        <ShieldAlert className="h-6 w-6 text-primary" />
        <h1 className="text-2xl font-bold text-foreground">Signalements & modération</h1>
      </div>
      <div className="rounded-xl border border-border bg-card p-8 text-center text-muted-foreground shadow-sm">
        <p>Signalements utilisateurs, modération de contenu, résolution des litiges.</p>
        <p className="mt-2 text-sm">Connecter aux endpoints API pour afficher les données.</p>
      </div>
    </div>
  )
}
