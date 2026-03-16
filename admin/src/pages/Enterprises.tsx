import { Building2 } from "lucide-react"

export default function EnterprisesPage() {
  return (
    <div>
      <div className="mb-6 flex items-center gap-3">
        <Building2 className="h-6 w-6 text-primary" />
        <h1 className="text-2xl font-bold text-foreground">Gestion des entreprises</h1>
      </div>
      <div className="rounded-xl border border-border bg-card p-8 text-center text-muted-foreground shadow-sm">
        <p>Liste des entreprises clientes, contrats, abonnements.</p>
        <p className="mt-2 text-sm">Connecter aux endpoints API pour afficher les données.</p>
      </div>
    </div>
  )
}
