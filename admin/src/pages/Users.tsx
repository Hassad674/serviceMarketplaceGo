import { formatDate } from "@/lib/utils.ts"

const placeholderUsers = [
  { id: "1", name: "Marie Dupont", email: "marie@example.com", role: "provider", created_at: "2025-11-15", status: "active" },
  { id: "2", name: "Pierre Martin", email: "pierre@example.com", role: "enterprise", created_at: "2025-12-01", status: "active" },
  { id: "3", name: "Sophie Bernard", email: "sophie@example.com", role: "provider", created_at: "2026-01-10", status: "suspended" },
  { id: "4", name: "Lucas Moreau", email: "lucas@example.com", role: "enterprise", created_at: "2026-02-20", status: "active" },
  { id: "5", name: "Camille Leroy", email: "camille@example.com", role: "provider", created_at: "2026-03-05", status: "pending" },
]

const roleLabels: Record<string, string> = { provider: "Prestataire", enterprise: "Entreprise", admin: "Admin" }
const statusColors: Record<string, string> = {
  active: "bg-success/10 text-success",
  suspended: "bg-destructive/10 text-destructive",
  pending: "bg-warning/10 text-warning",
}
const statusLabels: Record<string, string> = { active: "Actif", suspended: "Suspendu", pending: "En attente" }

export default function UsersPage() {
  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold text-foreground">Utilisateurs</h1>
        <span className="text-sm text-muted-foreground">{placeholderUsers.length} utilisateurs</span>
      </div>

      <div className="overflow-hidden rounded-xl border border-border bg-card shadow-sm">
        <table className="w-full text-left text-sm">
          <thead className="border-b border-border bg-muted/50">
            <tr>
              <th className="px-6 py-3 font-medium text-muted-foreground">Nom</th>
              <th className="px-6 py-3 font-medium text-muted-foreground">Email</th>
              <th className="px-6 py-3 font-medium text-muted-foreground">Role</th>
              <th className="px-6 py-3 font-medium text-muted-foreground">Inscrit le</th>
              <th className="px-6 py-3 font-medium text-muted-foreground">Statut</th>
              <th className="px-6 py-3 font-medium text-muted-foreground">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {placeholderUsers.map((user) => (
              <tr key={user.id} className="hover:bg-muted/30">
                <td className="px-6 py-4 font-medium text-card-foreground">{user.name}</td>
                <td className="px-6 py-4 text-muted-foreground">{user.email}</td>
                <td className="px-6 py-4">{roleLabels[user.role] ?? user.role}</td>
                <td className="px-6 py-4 text-muted-foreground">{formatDate(user.created_at)}</td>
                <td className="px-6 py-4">
                  <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusColors[user.status]}`}>
                    {statusLabels[user.status]}
                  </span>
                </td>
                <td className="px-6 py-4">
                  <button className="text-sm text-primary hover:underline">Voir</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
